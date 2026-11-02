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
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

type constantTestEmbedder struct{ calls int }

func (*constantTestEmbedder) Identity() semantic.Identity {
	return semantic.Identity{Provider: "test", Model: "constant-test"}
}

func (embedder *constantTestEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	embedder.calls++
	vectors := make([][]float64, len(texts))
	for index := range texts {
		vectors[index] = []float64{1, 0}
	}
	return vectors, nil
}

func TestFlowLabelEligibilityAcrossPublicRetrievalMethods(t *testing.T) {
	oldPath := filepath.Join("..", "..", "testdata", "bpmn", "old.bpmn")
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	if err := os.MkdirAll(repository, 0o750); err != nil {
		t.Fatal(err)
	}
	// The source has an exact reference to one possible display label, plus
	// broad process vocabulary that must not qualify a different label edit.
	source := "package review\n// Request approved is displayed in the review screen.\nfunc ApproveRequest() bool { return true }\n// valid request budget approval\n"
	if err := os.WriteFile(filepath.Join(repository, "review.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	variants := []struct {
		name       string
		old        string
		new        string
		wantImpact bool
	}{
		{name: "label without direct source reference", old: `name="yes"`, new: `name="Data missing"`},
		{name: "label with direct source reference", old: `name="yes"`, new: `name="Request approved"`, wantImpact: true},
		{name: "behavioral condition change", old: `${valid == true}`, new: `${valid == true &amp;&amp; budget &lt; 100}`, wantImpact: true},
	}
	for _, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion, domain.MethodHybrid} {
		for _, variant := range variants {
			t.Run(string(method)+"/"+variant.name, func(t *testing.T) {
				newData := strings.Replace(string(oldData), variant.old, variant.new, 1)
				if newData == string(oldData) {
					t.Fatalf("fixture mutation did not apply: %q", variant.old)
				}
				newPath := filepath.Join(root, "changed-"+strings.ReplaceAll(variant.name, " ", "-")+".bpmn")
				if err := os.WriteFile(newPath, []byte(newData), 0o600); err != nil {
					t.Fatal(err)
				}
				embedder := &constantTestEmbedder{}
				request := app.Request{RepositoryPath: repository, OldBPMNPath: oldPath, NewBPMNPath: newPath, Method: method}
				first, err := wiring.NewAnalyzerWithSemanticEmbedder(embedder).Analyze(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				second, err := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{}).Analyze(context.Background(), request)
				if err != nil || !reflect.DeepEqual(first, second) {
					t.Fatalf("non-deterministic scenario result: %v", err)
				}
				if len(first.Changes) == 0 {
					t.Fatal("BPMN change not detected")
				}
				matched := false
				for _, change := range first.Changes {
					if change.ChangeType != domain.ModifySequenceFlow {
						continue
					}
					matched = true
					if variant.name == "behavioral condition change" {
						if change.PartOf == "" || len(change.Impacts) != 0 {
							t.Fatalf("condition flow must be represented under its gateway: %#v", change)
						}
						foundGatewayImpact := false
						for _, primary := range first.Changes {
							if primary.ChangeType == domain.ModifyGatewayCondition && primary.NewElement.ID == change.PartOf && len(primary.Impacts) > 0 {
								foundGatewayImpact = true
							}
						}
						if !foundGatewayImpact {
							t.Fatal("grouping lost gateway retrieval")
						}
						continue
					}
					if variant.wantImpact && len(change.Impacts) == 0 || !variant.wantImpact && (len(change.Impacts) != 0 || len(change.UnresolvedImpact) == 0) {
						t.Fatalf("unexpected flow candidates: %#v", change)
					}
					if !variant.wantImpact && embedder.calls != 0 {
						t.Fatalf("no-reference label edit invoked the semantic runtime %d times", embedder.calls)
					}
					if variant.name == "label with direct source reference" && len(change.Impacts) > 0 {
						foundReference := false
						for _, evidence := range change.Impacts[0].Evidence {
							foundReference = foundReference || evidence.Type == "FLOW_LABEL_TEXT"
						}
						if !foundReference {
							t.Fatalf("direct label evidence missing: %#v", change.Impacts[0])
						}
					}
				}
				if !matched {
					t.Fatal("sequence-flow change missing")
				}
			})
		}
	}
}

func TestAnalyzeEndToEndIsDeterministicAndPrivate(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	request := app.Request{
		RepositoryPath: filepath.Join(root, "repository"),
		OldBPMNPath:    filepath.Join(root, "bpmn", "old.bpmn"),
		NewBPMNPath:    filepath.Join(root, "bpmn", "new.bpmn"),
		RepositoryName: "synthetic-repository", Revision: "fixture-v1", Method: domain.MethodBM25,
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

func TestOmittedAndExplicitDefaultIngestLimitsShareIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	request := app.Request{
		RepositoryPath: filepath.Join(root, "repository"),
		OldBPMNPath:    filepath.Join(root, "bpmn", "old.bpmn"),
		NewBPMNPath:    filepath.Join(root, "bpmn", "new.bpmn"),
		RepositoryName: "synthetic-repository", Revision: "fixture-v1", Method: domain.MethodBM25,
	}
	implicit, err := wiring.NewAnalyzer().Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Ingest = ingest.Options{MaxFileSize: 1 << 20, MaxTotalSize: 64 << 20, MaxFiles: 10000}
	explicit, err := wiring.NewAnalyzer().Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if implicit.InputFingerprint != explicit.InputFingerprint || implicit.AnalysisID != explicit.AnalysisID {
		t.Fatalf("equivalent ingest limits changed identity: implicit=%s/%s explicit=%s/%s", implicit.InputFingerprint, implicit.AnalysisID, explicit.InputFingerprint, explicit.AnalysisID)
	}
}

func TestInsertTaskKeepsFlowDiffWithoutDuplicateRetrieval(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	result, err := wiring.NewAnalyzer().Analyze(context.Background(), app.Request{
		RepositoryPath: filepath.Join(root, "repository"),
		OldBPMNPath:    filepath.Join(root, "bpmn", "insert-task-old.bpmn"),
		NewBPMNPath:    filepath.Join(root, "bpmn", "insert-task-new.bpmn"),
		Method:         domain.MethodBM25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 4 {
		t.Fatalf("raw changes lost: %d", len(result.Changes))
	}
	primary, details := 0, 0
	for _, change := range result.Changes {
		if change.PartOf != "" {
			details++
			if len(change.Impacts) != 0 || len(change.UnresolvedImpact) != 0 {
				t.Fatalf("supporting flow retrieved independently: %#v", change)
			}
		} else if change.ChangeType == domain.AddTask {
			primary++
		}
	}
	if primary != 1 || details != 3 {
		t.Fatalf("primary=%d details=%d", primary, details)
	}
	markdown := string(report.Markdown(result))
	if !strings.Contains(markdown, "Supporting sequence-flow edits (mapped with this change)") || strings.Count(markdown, "## 4. ADD_TASK") != 1 || strings.Contains(markdown, "## 1. MODIFY_SEQUENCE_FLOW") {
		t.Fatalf("Markdown did not group supporting edits: %s", markdown)
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

func TestAnalyzeExcludesRepositoryCopyOfUploadedBPMN(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata")
	oldData, err := os.ReadFile(filepath.Join(fixtureRoot, "bpmn", "old.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	repository := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repository, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "docs", "baseline.xml"), oldData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "source.go"), []byte("package sample\n// TASK-VALIDATE validates requests\nfunc ValidateRequest() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	reportValue, err := wiring.NewAnalyzer().Analyze(context.Background(), app.Request{
		RepositoryPath: repository,
		OldBPMNPath:    filepath.Join(fixtureRoot, "bpmn", "old.bpmn"),
		NewBPMNPath:    filepath.Join(fixtureRoot, "bpmn", "new.bpmn"),
		Method:         domain.MethodBM25,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range reportValue.Changes {
		for _, impact := range change.Impacts {
			if impact.File == "docs/baseline.xml" {
				t.Fatalf("uploaded BPMN copy leaked into candidates: %#v", impact)
			}
		}
	}
	foundWarning := false
	for _, warning := range reportValue.Warnings {
		foundWarning = foundWarning || strings.Contains(warning.Message, "skipped method input copy: docs/baseline.xml")
	}
	if !foundWarning {
		t.Fatalf("missing input-copy exclusion warning: %#v", reportValue.Warnings)
	}
}
