package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
)

func TestRunAnalyzeWritesBothArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	output := t.TempDir()
	arguments := []string{"analyze", "--repo", filepath.Join(root, "repository"), "--old-bpmn", filepath.Join(root, "bpmn", "old.bpmn"), "--new-bpmn", filepath.Join(root, "bpmn", "new.bpmn"), "--output", output, "--repository-name", "synthetic", "--revision", "fixture-v1", "--method", "bm25"}
	var stdout, stderr bytes.Buffer
	if err := run(arguments, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v stderr=%s", err, stderr.String())
	}
	for _, name := range []string{"impact-report.json", "impact-report.md"} {
		if info, err := os.Stat(filepath.Join(output, name)); err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing or empty: %v", name, err)
		}
	}
}

func TestRunRequiresAnalyzeCommand(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected usage error")
	}
}

func TestRunConfigurationAndEvaluate(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	prediction := t.TempDir()
	analyze := []string{"analyze", "--repo", filepath.Join(root, "repository"), "--old-bpmn", filepath.Join(root, "bpmn", "old.bpmn"), "--new-bpmn", filepath.Join(root, "bpmn", "new.bpmn"), "--output", prediction, "--method", "configuration", "--mapping-config", filepath.Join(root, "config", "mapping.json")}
	if err := run(analyze, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	evaluation := t.TempDir()
	evaluate := []string{"evaluate", "--report", filepath.Join(prediction, "impact-report.json"), "--ground-truth", filepath.Join(root, "evaluation", "ground-truth.json"), "--output", evaluation}
	if err := run(evaluate, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evaluation-report.json", "evaluation-report.md"} {
		if _, err := os.Stat(filepath.Join(evaluation, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestRunBM25Method(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	output := t.TempDir()
	arguments := []string{"analyze", "--repo", filepath.Join(root, "repository"), "--old-bpmn", filepath.Join(root, "bpmn", "old.bpmn"), "--new-bpmn", filepath.Join(root, "bpmn", "new.bpmn"), "--output", output, "--method", "bm25"}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, "impact-report.json"))
	if err != nil || !bytes.Contains(data, []byte(`"method": "bm25"`)) {
		t.Fatalf("BM25 report missing method metadata: %v", err)
	}
}

func TestRunReconcile(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	output := t.TempDir()
	arguments := []string{"reconcile", "--reviewer-a", filepath.Join(root, "review", "reviewer-a.json"), "--reviewer-b", filepath.Join(root, "review", "reviewer-b.json"), "--output", output}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, "reconciliation.json"))
	if err != nil || !bytes.Contains(data, []byte(`"disagreements"`)) {
		t.Fatalf("reconciliation missing: %v", err)
	}
}

func TestRunPrepareReviewCreatesBlindImmutablePack(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	firstPrediction := t.TempDir()
	secondPrediction := t.TempDir()
	base := []string{"--repo", filepath.Join(root, "repository"), "--old-bpmn", filepath.Join(root, "bpmn", "old.bpmn"), "--new-bpmn", filepath.Join(root, "bpmn", "new.bpmn"), "--repository-name", "synthetic", "--revision", "fixture-v1"}
	for _, item := range []struct {
		method string
		output string
	}{{"configuration", firstPrediction}, {"bm25", secondPrediction}} {
		arguments := append([]string{"analyze"}, base...)
		arguments = append(arguments, "--method", item.method, "--output", item.output)
		if item.method == "configuration" {
			arguments = append(arguments, "--mapping-config", filepath.Join(root, "config", "mapping.json"))
		}
		if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(t.TempDir(), "blind-pack")
	arguments := []string{"prepare-review", "--report", filepath.Join(firstPrediction, "impact-report.json"), "--report", filepath.Join(secondPrediction, "impact-report.json"), "--output", output, "--reviewer-a", "alice", "--reviewer-b", "bob"}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"coordinator-manifest.json", "review-guide.md", "reviewer-a.json", "reviewer-b.json"} {
		data, err := os.ReadFile(filepath.Join(output, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("missing %s: %v", name, err)
		}
		if name != "coordinator-manifest.json" && (bytes.Contains(data, []byte(`"confidence"`)) || bytes.Contains(data, []byte(`"impacts"`))) {
			t.Fatalf("blind artifact %s leaked predictions", name)
		}
	}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected immutable output-directory error")
	}
	for _, name := range []string{"reviewer-a.json", "reviewer-b.json"} {
		path := filepath.Join(output, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var annotation review.Annotation
		if err := json.Unmarshal(data, &annotation); err != nil {
			t.Fatal(err)
		}
		for index := range annotation.Cases {
			annotation.Cases[index].Expected = annotation.Cases[index].Expected[:0]
		}
		data, err = json.MarshalIndent(annotation, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	reconciliation := filepath.Join(t.TempDir(), "reconciliation")
	reconcile := []string{"reconcile", "--reviewer-a", filepath.Join(output, "reviewer-a.json"), "--reviewer-b", filepath.Join(output, "reviewer-b.json"), "--manifest", filepath.Join(output, "coordinator-manifest.json"), "--output", reconciliation}
	if err := run(reconcile, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(reconciliation, "reconciliation.json"))
	if err != nil || !bytes.Contains(data, []byte(`"review_pack"`)) || !bytes.Contains(data, []byte(`"input_fingerprint"`)) {
		t.Fatalf("snapshot-bound reconciliation missing provenance: %v", err)
	}
}

func TestRunFrozenBenchmark(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	output := t.TempDir()
	arguments := []string{"benchmark", "--manifest", filepath.Join("testdata", "benchmark", "manifest.json"), "--output", output}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"benchmark-report.json", "benchmark-report.md", "benchmark-report.csv", "benchmark-timing.json"} {
		if info, err := os.Stat(filepath.Join(output, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestRunMutateWritesOnceAndRefusesOverwrite(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "scenario")
	output := filepath.Join(t.TempDir(), "new.bpmn")
	arguments := []string{"mutate", "--spec", filepath.Join(root, "mutation.json"), "--baseline", filepath.Join(root, "baseline.bpmn"), "--output", output}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("generated BPMN missing: %v", err)
	}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected existing-output error")
	}
}
