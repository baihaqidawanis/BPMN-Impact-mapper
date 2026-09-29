package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func TestAnalyzeEndToEndIsDeterministicAndPrivate(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	request := app.Request{
		RepositoryPath: filepath.Join(root, "repository"),
		OldBPMNPath:    filepath.Join(root, "bpmn", "old.bpmn"),
		NewBPMNPath:    filepath.Join(root, "bpmn", "new.bpmn"),
		RepositoryName: "synthetic-repository", Revision: "fixture-v1",
	}
	first, err := wiring.NewAnalyzer().Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := wiring.NewAnalyzer().Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("analysis is not deterministic")
	}
	data, err := report.JSON(first)
	if err != nil {
		t.Fatal(err)
	}
	abs, _ := filepath.Abs(request.RepositoryPath)
	if strings.Contains(string(data), abs) || strings.Contains(string(data), strings.ReplaceAll(abs, "\\", "\\\\")) {
		t.Fatal("canonical report leaked absolute repository path")
	}
	for _, change := range first.Changes {
		for _, impact := range change.Impacts {
			if len(impact.Evidence) == 0 {
				t.Fatalf("candidate without evidence: %#v", impact)
			}
		}
	}
	output := t.TempDir()
	if err := app.WriteArtifacts(output, first); err != nil {
		t.Fatal(err)
	}
	writtenJSON, err := os.ReadFile(filepath.Join(output, "impact-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded domain.Report
	if err := json.Unmarshal(writtenJSON, &decoded); err != nil {
		t.Fatal(err)
	}
	writtenMarkdown, err := os.ReadFile(filepath.Join(output, "impact-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(writtenMarkdown) != string(report.Markdown(decoded)) {
		t.Fatal("Markdown was not rendered from the canonical report object")
	}
}

func TestAnalyzeHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := wiring.NewAnalyzer().Analyze(ctx, app.Request{RepositoryPath: ".", OldBPMNPath: "old", NewBPMNPath: "new"})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestAnalyzeSupportsIsolatedMethodsAndExcludesMethodInputs(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	base := app.Request{
		RepositoryPath: filepath.Join(root, "repository"),
		OldBPMNPath:    filepath.Join(root, "bpmn", "old.bpmn"),
		NewBPMNPath:    filepath.Join(root, "bpmn", "new.bpmn"),
		RepositoryName: "synthetic", Revision: "fixture-v2",
	}
	base.Method = domain.MethodBM25
	bm25, err := wiring.NewAnalyzer().Analyze(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if bm25.Method != domain.MethodBM25 {
		t.Fatalf("method=%s", bm25.Method)
	}
	for _, change := range bm25.Changes {
		for _, impact := range change.Impacts {
			if impact.RequiredAction != domain.ActionVerify {
				t.Fatalf("BM25 inferred action: %#v", impact)
			}
			if strings.Contains(impact.File, "ground-truth") || strings.HasSuffix(impact.File, ".bpmn") {
				t.Fatalf("method input or hidden truth leaked into candidates: %s", impact.File)
			}
		}
	}
	base.Method = domain.MethodConfiguration
	base.MappingConfigPath = filepath.Join(root, "config", "mapping.json")
	configured, err := wiring.NewAnalyzer().Analyze(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, change := range configured.Changes {
		if change.ChangeType == domain.AddTask && change.NewElement != nil && change.NewElement.ID == "TASK-VALIDATE" {
			found = len(change.Impacts) == 1 && change.Impacts[0].Evidence[0].Type == "CONFIG_MAPPING"
		}
	}
	if !found {
		t.Fatal("configuration method did not emit its indexed explicit mapping")
	}
	if configured.AnalysisID == bm25.AnalysisID {
		t.Fatal("analysis ID did not bind method/configuration")
	}
}

func TestAnalyzeRejectsInvalidMethodConfigurationCombination(t *testing.T) {
	request := app.Request{RepositoryPath: ".", OldBPMNPath: "old", NewBPMNPath: "new", Method: domain.MethodHybrid, MappingConfigPath: "mapping.json"}
	if _, err := wiring.NewAnalyzer().Analyze(context.Background(), request); err == nil {
		t.Fatal("expected configuration/method validation error")
	}
	request.MappingConfigPath = ""
	request.Method = "unknown"
	if _, err := wiring.NewAnalyzer().Analyze(context.Background(), request); err == nil {
		t.Fatal("expected unsupported method error")
	}
}
