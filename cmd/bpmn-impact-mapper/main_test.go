package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAnalyzeWritesBothArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	output := t.TempDir()
	arguments := []string{"analyze", "--repo", filepath.Join(root, "repository"), "--old-bpmn", filepath.Join(root, "bpmn", "old.bpmn"), "--new-bpmn", filepath.Join(root, "bpmn", "new.bpmn"), "--output", output, "--repository-name", "synthetic", "--revision", "fixture-v1"}
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

func TestRunFrozenBenchmark(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	output := t.TempDir()
	arguments := []string{"benchmark", "--manifest", filepath.Join("testdata", "benchmark", "manifest.json"), "--output", output}
	if err := run(arguments, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"benchmark-report.json", "benchmark-report.md", "benchmark-timing.json"} {
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
