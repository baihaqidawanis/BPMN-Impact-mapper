package experiment_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/experiment"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func TestRunWritesDeterministicEvaluationArtifacts(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	prediction := t.TempDir()
	request := app.Request{RepositoryPath: filepath.Join(root, "repository"), OldBPMNPath: filepath.Join(root, "bpmn", "old.bpmn"), NewBPMNPath: filepath.Join(root, "bpmn", "new.bpmn"), RepositoryName: "synthetic", Revision: "fixture-v2", Method: domain.MethodBM25}
	value, err := wiring.NewAnalyzer().Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.WriteArtifacts(prediction, value); err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	first, err := experiment.Run(filepath.Join(prediction, "impact-report.json"), filepath.Join(root, "evaluation", "ground-truth.json"), output)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := os.ReadFile(filepath.Join(output, "evaluation-report.json"))
	second, err := experiment.Run(filepath.Join(prediction, "impact-report.json"), filepath.Join(root, "evaluation", "ground-truth.json"), output)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, _ := os.ReadFile(filepath.Join(output, "evaluation-report.json"))
	if first.EvaluationID != second.EvaluationID || string(firstJSON) != string(secondJSON) {
		t.Fatal("evaluation output is not deterministic")
	}
	for _, name := range []string{"evaluation-report.json", "evaluation-report.md"} {
		if info, err := os.Stat(filepath.Join(output, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}
