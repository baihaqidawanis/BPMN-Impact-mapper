package benchmark_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	impactreport "github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
)

func TestCSVIsDeterministicAndSpreadsheetReady(t *testing.T) {
	report := benchmark.Report{BenchmarkID: "bench", Results: []benchmark.ScenarioMethodResult{{ScenarioID: "case,one", Method: domain.MethodHybrid, Repetitions: 2, Reproducible: true}}, Aggregates: []benchmark.MethodAggregate{{Method: domain.MethodHybrid, Scenarios: 1}}}
	first, err := benchmark.CSV(report)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := benchmark.CSV(report)
	if !bytes.Equal(first, second) || !bytes.Contains(first, []byte(`"case,one"`)) || !bytes.Contains(first, []byte("false_discovery_proportion")) {
		t.Fatalf("unexpected CSV: %s", first)
	}
}

func TestBenchmarkShowsNegativeControlsSeparately(t *testing.T) {
	metrics := evaluation.Metrics{NegativeControls: 2, CorrectAbstentions: 1, NegativeControlAccuracy: 0.5, UnresolvedRate: 0.5}
	report := benchmark.Report{BenchmarkID: "bench", Results: []benchmark.ScenarioMethodResult{{ScenarioID: "control", Method: domain.MethodBM25, Metrics: metrics}}, Aggregates: []benchmark.MethodAggregate{{Method: domain.MethodBM25, Scenarios: 1, Metrics: metrics}}}
	markdown := string(benchmark.Markdown(report))
	csvData, err := benchmark.CSV(report)
	if err != nil || !strings.Contains(markdown, "1/2 (0.5000)") || !bytes.Contains(csvData, []byte("negative_control_accuracy")) {
		t.Fatalf("negative controls missing from benchmark exports: markdown=%s err=%v", markdown, err)
	}
}

func TestTimingComparisonSummarizesMetricsAndVolatileLatency(t *testing.T) {
	report := benchmark.Report{BenchmarkID: "bench", Aggregates: []benchmark.MethodAggregate{{Method: domain.MethodBM25, Scenarios: 1, Metrics: evaluation.Metrics{Recall: evaluation.AtK{At5: 0.75}, MRR: 0.5, FileF1: 0.6}}}}
	timing := benchmark.Timing{BenchmarkID: "bench", Runs: []benchmark.TimingEntry{
		{Method: domain.MethodBM25, ScenarioID: "one", Repetition: 1, Milliseconds: 30},
		{Method: domain.MethodBM25, ScenarioID: "one", Repetition: 2, Milliseconds: 10},
	}}
	markdown := string(benchmark.TimingMarkdown(report, timing))
	csvData, err := benchmark.TimingCSV(report, timing)
	if err != nil || !strings.Contains(markdown, "| bm25 | 2 | 0.7500 | 0.5000 | 0.6000 | 10 | 20.0 | 20.0 | 30 |") || !strings.Contains(markdown, "first embedding method includes cold model startup") || !bytes.Contains(csvData, []byte("first embedding method includes cold startup")) {
		t.Fatalf("markdown=%s csv=%s err=%v", markdown, csvData, err)
	}
}

func TestAblationReportUsesHybridAsDescriptiveReference(t *testing.T) {
	report := benchmark.Report{BenchmarkID: "bench", Aggregates: []benchmark.MethodAggregate{
		{Method: domain.MethodBM25, Scenarios: 2, Metrics: evaluation.Metrics{Recall: evaluation.AtK{At5: 0.5}, MRR: 0.4, FileF1: 0.3, ActionMacroF1: 0.2}},
		{Method: domain.MethodHybrid, Scenarios: 2, Metrics: evaluation.Metrics{Recall: evaluation.AtK{At5: 0.8}, MRR: 0.7, FileF1: 0.6, ActionMacroF1: 0.5}},
	}}
	markdown := string(benchmark.AblationMarkdown(report))
	csvData, err := benchmark.AblationCSV(report)
	if err != nil || !strings.Contains(markdown, "| bm25 | 0.5000 | -0.3000 |") || !strings.Contains(markdown, "descriptive, not causal") || !bytes.Contains(csvData, []byte("delta_action_macro_f1")) {
		t.Fatalf("markdown=%s csv=%s err=%v", markdown, csvData, err)
	}
}

func TestManifestValidationEnforcesFairFrozenRuns(t *testing.T) {
	hash := strings.Repeat("a", 64)
	valid := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5, Methods: []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion}, Scenarios: []benchmark.Scenario{{ID: "one", RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hash, NewBPMNPath: "new", NewBPMNSHA256: hash, GroundTruthPath: "truth", GroundTruthSHA256: hash}}}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	clone := func() benchmark.Manifest {
		copy := valid
		copy.Methods = append([]domain.AnalysisMethod(nil), valid.Methods...)
		copy.Scenarios = append([]benchmark.Scenario(nil), valid.Scenarios...)
		return copy
	}
	tests := []benchmark.Manifest{clone(), clone(), clone(), clone()}
	tests[0].Repetitions = 1
	tests[1].Methods = append(tests[1].Methods, domain.MethodHybrid)
	tests[2].Scenarios = append(tests[2].Scenarios, tests[2].Scenarios[0])
	tests[3].Scenarios[0].GroundTruthSHA256 = "unfrozen"
	for index, manifest := range tests {
		if err := manifest.Validate(); err == nil {
			t.Errorf("invalid manifest %d accepted", index)
		}
	}
	withConfiguration := clone()
	withConfiguration.Methods = append(withConfiguration.Methods, domain.MethodConfiguration)
	if err := withConfiguration.Validate(); err == nil {
		t.Fatal("configuration benchmark accepted without frozen mapping")
	}
	withConfiguration.Scenarios[0].MappingConfigPath = "mapping"
	withConfiguration.Scenarios[0].MappingConfigSHA256 = hash
	if err := withConfiguration.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRunnerLoadsGroundTruthAfterInferenceAndRejectsDrift(t *testing.T) {
	truth := []byte(`{"version":"1","cases":[]}`)
	files := &fakeFiles{data: map[string][]byte{"old": []byte("old"), "new": []byte("new"), "truth": truth}}
	manifest := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5, Methods: []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion}, Scenarios: []benchmark.Scenario{{ID: "one", RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hashOf([]byte("old")), NewBPMNPath: "new", NewBPMNSHA256: hashOf([]byte("new")), GroundTruthPath: "truth", GroundTruthSHA256: hashOf(truth)}}}
	analyzer := &fakeAnalyzer{events: &files.events}
	runner := benchmark.Runner{Analyzer: analyzer, Files: files, Predictions: renderer{}}
	if _, _, err := runner.Run(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	firstTruth, firstAnalyze := -1, -1
	for index, event := range files.events {
		if event == "analyze" && firstAnalyze < 0 {
			firstAnalyze = index
		}
		if event == "read:truth" && firstTruth < 0 {
			firstTruth = index
		}
	}
	if firstAnalyze < 0 || firstTruth < firstAnalyze {
		t.Fatalf("ground truth read before inference: %v", files.events)
	}
	analyzer.drift = true
	analyzer.calls = 0
	files.events = nil
	if _, _, err := runner.Run(context.Background(), manifest); err == nil {
		t.Fatal("expected repeated prediction drift failure")
	}
}

func TestRunnerRejectsIncompleteGroundTruthAfterInference(t *testing.T) {
	truth := []byte(`{"version":"1","cases":[]}`)
	files := &fakeFiles{data: map[string][]byte{"old": []byte("old"), "new": []byte("new"), "truth": truth}}
	manifest := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5,
		Methods:   []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion},
		Scenarios: []benchmark.Scenario{{ID: "one", RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hashOf([]byte("old")), NewBPMNPath: "new", NewBPMNSHA256: hashOf([]byte("new")), GroundTruthPath: "truth", GroundTruthSHA256: hashOf(truth)}},
	}
	analyzer := &fakeAnalyzer{events: &files.events, change: true}
	_, _, err := (benchmark.Runner{Analyzer: analyzer, Files: files, Predictions: renderer{}}).Run(context.Background(), manifest)
	if err == nil || !strings.Contains(err.Error(), "does not cover") {
		t.Fatalf("incomplete truth accepted: %v", err)
	}
	if len(files.events) < 4 || files.events[len(files.events)-2] != "analyze" || files.events[len(files.events)-1] != "read:truth" {
		t.Fatalf("ground truth was read before inference: %v", files.events)
	}
}

func TestRunnerLabelsSubsetDiagnosticsAndExcludesThemFromAggregates(t *testing.T) {
	partial := []byte(`{"version":"1","cases":[{"change_type":"ADD_TASK","element_id":"TASK-NEW","expected":[]}]}`)
	complete := []byte(`{"version":"1","cases":[{"change_type":"ADD_TASK","element_id":"TASK-NEW","expected":[]},{"change_type":"ADD_TASK","element_id":"TASK-SECOND","expected":[]}]}`)
	files := &fakeFiles{data: map[string][]byte{"old": []byte("old"), "new": []byte("new"), "partial": partial, "complete": complete}}
	makeScenario := func(id, truthPath string) benchmark.Scenario {
		return benchmark.Scenario{ID: id, RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hashOf([]byte("old")), NewBPMNPath: "new", NewBPMNSHA256: hashOf([]byte("new")), GroundTruthPath: truthPath, GroundTruthSHA256: hashOf(files.data[truthPath])}
	}
	partialScenario := makeScenario("partial", "partial")
	partialScenario.TruthScope = "subset"
	manifest := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5, LabelProvenance: "fixture-authored",
		Methods:   []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion},
		Scenarios: []benchmark.Scenario{partialScenario, makeScenario("complete", "complete")},
	}
	analyzer := &fakeAnalyzer{events: &files.events, changeCount: 2}
	report, _, err := (benchmark.Runner{Analyzer: analyzer, Files: files, Predictions: renderer{}}).Run(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.PartialScenarios) != 1 || report.PartialScenarios[0] != "partial" || len(report.Results) != 8 || report.LabelProvenance != "fixture-authored" {
		t.Fatalf("partial scenario metadata missing: %#v", report)
	}
	for _, aggregate := range report.Aggregates {
		if aggregate.Scenarios != 1 || aggregate.Metrics.EvaluatedChanges != 2 {
			t.Fatalf("partial scenario entered method aggregate: %#v", aggregate)
		}
	}
	if !strings.Contains(string(benchmark.Markdown(report)), "excluded from all method aggregates") || !strings.Contains(string(benchmark.Markdown(report)), "Label provenance: `fixture-authored`") {
		t.Fatal("human report did not warn about partial labels")
	}
	csv, err := benchmark.CSV(report)
	if err != nil || !strings.Contains(string(csv), ",subset,fixture-authored,") || !strings.Contains(string(csv), ",complete-only,fixture-authored,") || !strings.Contains(string(csv), "r_precision") {
		t.Fatalf("CSV truth scope missing: %s error=%v", csv, err)
	}
	manifest.Scenarios = []benchmark.Scenario{partialScenario}
	if err := manifest.Validate(); err == nil {
		t.Fatal("subset-only manifest accepted without complete method aggregate")
	}
}

type fakeFiles struct {
	data   map[string][]byte
	events []string
}

func (files *fakeFiles) Read(path string, _ int64) ([]byte, error) {
	files.events = append(files.events, "read:"+path)
	return files.data[path], nil
}
func (*fakeFiles) Write(string, []byte) error { return nil }

type fakeAnalyzer struct {
	events      *[]string
	calls       int
	drift       bool
	change      bool
	changeCount int
}

func (analyzer *fakeAnalyzer) Analyze(_ context.Context, request app.Request) (domain.Report, error) {
	*analyzer.events = append(*analyzer.events, "analyze")
	analyzer.calls++
	id := string(request.Method)
	if analyzer.drift {
		id += string(rune('a' + analyzer.calls))
	}
	changes := []domain.Change{}
	if analyzer.change || analyzer.changeCount > 0 {
		changes = append(changes, domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-NEW", Name: "New task", Type: "userTask"}})
	}
	if analyzer.changeCount > 1 {
		changes = append(changes, domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-SECOND", Name: "Second task", Type: "userTask"}})
	}
	return domain.Report{AnalysisID: id, InputFingerprint: "shared-fingerprint", Method: request.Method, Repository: domain.Repository{Name: "fixture", Revision: "one"}, Changes: changes, Warnings: []domain.Warning{}}, nil
}

type renderer struct{}

func (renderer) JSON(value domain.Report) ([]byte, error) { return impactreport.JSON(value) }
func (renderer) Markdown(value domain.Report) []byte      { return impactreport.Markdown(value) }

func hashOf(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
