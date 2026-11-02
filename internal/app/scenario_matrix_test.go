package app_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

type controlMatrix struct {
	Version        string `json:"version"`
	RepositoryPath string `json:"repository_path"`
	RepositoryName string `json:"repository_name"`
	Revision       string `json:"revision"`
	TopK           int    `json:"top_k"`
	Scenarios      []struct {
		ID                       string            `json:"id"`
		OldBPMNPath              string            `json:"old_bpmn_path"`
		OldBPMNSHA256            string            `json:"old_bpmn_sha256"`
		NewBPMNPath              string            `json:"new_bpmn_path"`
		NewBPMNSHA256            string            `json:"new_bpmn_sha256"`
		ExpectedInputFingerprint string            `json:"expected_input_fingerprint"`
		ChangeType               domain.ChangeType `json:"change_type"`
		ElementID                string            `json:"element_id"`
		ExpectCandidates         bool              `json:"expect_candidates"`
		ExpectAbstention         bool              `json:"expect_abstention"`
	} `json:"scenarios"`
}

// This matrix checks deterministic behavior and taxonomy coverage, not mapping
// accuracy. The constant test embedder deliberately does not model meaning.
func TestFrozenSyntheticScenarioMatrixAcrossMethods(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "testdata", "benchmark-controls", "scenario-matrix.json"))
	if err != nil {
		t.Fatal(err)
	}
	var matrix controlMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatal(err)
	}
	if matrix.Version != "1" || len(matrix.Scenarios) != 4 {
		t.Fatalf("unexpected control matrix version or coverage: %#v", matrix)
	}
	for _, scenario := range matrix.Scenarios {
		oldPath := filepath.Join(root, filepath.FromSlash(scenario.OldBPMNPath))
		newPath := filepath.Join(root, filepath.FromSlash(scenario.NewBPMNPath))
		assertFixtureHash(t, oldPath, scenario.OldBPMNSHA256)
		assertFixtureHash(t, newPath, scenario.NewBPMNSHA256)
		for _, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion, domain.MethodHybrid} {
			t.Run(scenario.ID+"/"+string(method), func(t *testing.T) {
				request := app.Request{
					RepositoryPath: filepath.Join(root, filepath.FromSlash(matrix.RepositoryPath)),
					RepositoryName: matrix.RepositoryName, Revision: matrix.Revision,
					OldBPMNPath: oldPath, NewBPMNPath: newPath, Method: method,
					Retrieval: mapping.Options{TopK: matrix.TopK},
				}
				first, err := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{}).Analyze(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				second, err := wiring.NewAnalyzerWithSemanticEmbedder(&constantTestEmbedder{}).Analyze(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				firstJSON, err := report.JSON(first)
				if err != nil {
					t.Fatal(err)
				}
				secondJSON, err := report.JSON(second)
				if err != nil || string(firstJSON) != string(secondJSON) {
					t.Fatalf("prediction changed across identical runs: %v", err)
				}
				if first.InputFingerprint != scenario.ExpectedInputFingerprint {
					t.Fatalf("input snapshot drift: got %s want %s", first.InputFingerprint, scenario.ExpectedInputFingerprint)
				}
				matched := false
				for _, change := range first.Changes {
					id := ""
					if change.NewElement != nil {
						id = change.NewElement.ID
					} else if change.OldElement != nil {
						id = change.OldElement.ID
					}
					for _, impact := range change.Impacts {
						if len(impact.Evidence) == 0 || filepath.IsAbs(impact.File) {
							t.Fatalf("candidate lacks bounded relative evidence: %#v", impact)
						}
					}
					if change.ChangeType != scenario.ChangeType || id != scenario.ElementID {
						continue
					}
					matched = true
					if scenario.ExpectCandidates && len(change.Impacts) == 0 {
						t.Fatalf("expected a retrieval candidate for %s/%s", scenario.ID, method)
					}
					if scenario.ExpectAbstention && (len(change.Impacts) != 0 || len(change.UnresolvedImpact) == 0) {
						t.Fatalf("negative control did not abstain: %#v", change)
					}
				}
				if !matched {
					t.Fatalf("expected BPMN change %s:%s missing", scenario.ChangeType, scenario.ElementID)
				}
			})
		}
	}
}

func assertFixtureHash(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if actual := fmt.Sprintf("%x", sha256.Sum256(data)); actual != expected {
		t.Fatalf("fixture %s changed: got SHA-256 %s want %s", path, actual, expected)
	}
}
