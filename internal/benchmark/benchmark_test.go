package benchmark_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/benchmark"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	impactreport "github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
)

func TestManifestValidationEnforcesFairFrozenRuns(t *testing.T) {
	hash := strings.Repeat("a", 64)
	valid := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5, Methods: []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodConfiguration}, Scenarios: []benchmark.Scenario{{ID: "one", RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hash, NewBPMNPath: "new", NewBPMNSHA256: hash, MappingConfigPath: "mapping", MappingConfigSHA256: hash, GroundTruthPath: "truth", GroundTruthSHA256: hash}}}
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
}

func TestRunnerLoadsGroundTruthAfterInferenceAndRejectsDrift(t *testing.T) {
	truth := []byte(`{"version":"1","cases":[]}`)
	files := &fakeFiles{data: map[string][]byte{"old": []byte("old"), "new": []byte("new"), "mapping": []byte("mapping"), "truth": truth}}
	manifest := benchmark.Manifest{Version: "1", Repetitions: 2, TopK: 5, Methods: []domain.AnalysisMethod{domain.MethodHybrid, domain.MethodBM25, domain.MethodConfiguration}, Scenarios: []benchmark.Scenario{{ID: "one", RepositoryPath: "repo", OldBPMNPath: "old", OldBPMNSHA256: hashOf([]byte("old")), NewBPMNPath: "new", NewBPMNSHA256: hashOf([]byte("new")), MappingConfigPath: "mapping", MappingConfigSHA256: hashOf([]byte("mapping")), GroundTruthPath: "truth", GroundTruthSHA256: hashOf(truth)}}}
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
	events *[]string
	calls  int
	drift  bool
}

func (analyzer *fakeAnalyzer) Analyze(_ context.Context, request app.Request) (domain.Report, error) {
	*analyzer.events = append(*analyzer.events, "analyze")
	analyzer.calls++
	id := string(request.Method)
	if analyzer.drift {
		id += string(rune('a' + analyzer.calls))
	}
	return domain.Report{AnalysisID: id, InputFingerprint: "shared-fingerprint", Method: request.Method, Repository: domain.Repository{Name: "fixture", Revision: "one"}, Changes: []domain.Change{}, Warnings: []domain.Warning{}}, nil
}

type renderer struct{}

func (renderer) JSON(value domain.Report) ([]byte, error) { return impactreport.JSON(value) }
func (renderer) Markdown(value domain.Report) []byte      { return impactreport.Markdown(value) }

func hashOf(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
