package report_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/report"
)

func TestCanonicalJSONAndMarkdownGolden(t *testing.T) {
	value := goldenReport()
	jsonData, err := report.JSON(value)
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "report.json", jsonData)
	assertGolden(t, "report.md", report.Markdown(value))
	var decoded domain.Report
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(report.Markdown(decoded)) != string(report.Markdown(value)) {
		t.Fatal("Markdown was not stable after canonical JSON round trip")
	}
}

func TestCurrentHybridMarkdownExplainsTwoRankings(t *testing.T) {
	value := goldenReport()
	value.MethodMetadata = map[string]string{"left_ranking_version": "context-bm25-v1"}
	value.Changes[0].Impacts[0].ScoreBreakdown = domain.ScoreBreakdown{BM25: 0.7, Semantic: 0.4, RRF: 0.9, Total: 0.9}
	value.Changes[0].Impacts[0].RequiredAction = domain.ActionVerify
	markdown := string(report.Markdown(value))
	if !strings.Contains(markdown, "BM25 word match: 0.7000") || !strings.Contains(markdown, "IDs, function names, paths, and code patterns are optional evidence") || strings.Contains(markdown, "current code-evidence weights are not recalibrated") {
		t.Fatalf("current Hybrid explanation is misleading: %s", markdown)
	}
}

func goldenReport() domain.Report {
	evidenceScore := 0.8
	return domain.Report{
		AnalysisID: "abc123", InputFingerprint: "input123", Method: domain.MethodHybrid, Repository: domain.Repository{Name: "synthetic", Revision: "fixture-v1"},
		Changes: []domain.Change{{
			ChangeType: domain.AddTask, OldElement: nil,
			NewElement: &domain.Element{ID: "TASK-VALIDATE", Name: "Validate request", Type: "serviceTask"},
			Context: &domain.ChangeContext{Pattern: domain.PatternInsertTask,
				Predecessors:   []domain.ElementContext{{ID: "TASK-CREATE", Name: "Create request", Type: "userTask"}},
				Successors:     []domain.ElementContext{{ID: "TASK-REGISTER", Name: "Register request", Type: "serviceTask"}},
				RelatedFlowIDs: []string{"FLOW-CREATE-VALIDATE", "FLOW-VALIDATE-REGISTER"}},
			Impacts: []domain.Impact{{Module: "internal", Layer: "application", File: "internal/application/request.go", Symbol: "ValidateRequest", RequiredAction: domain.ActionAdd,
				Evidence:       []domain.Evidence{{Type: "TRACE_ID", Location: "internal/application/request.go:4", Excerpt: "TASK-VALIDATE"}},
				ScoreBreakdown: domain.ScoreBreakdown{EvidenceScore: &evidenceScore, TraceID: 1, Symbol: 0.5, Path: 0.25, Lexical: 0.5, Structured: 1, BM25: 1, Total: 0.8}, Confidence: 0.8}},
			UnresolvedImpact: []string{}, Reason: "task added",
		}},
		Warnings: []domain.Warning{{Code: "EXAMPLE", Message: "synthetic warning"}},
	}
}

func assertGolden(t *testing.T, name string, actual []byte) {
	t.Helper()
	expected, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	if string(expected) != string(actual) {
		t.Fatalf("golden %s mismatch\n--- expected ---\n%s\n--- actual ---\n%s", name, expected, actual)
	}
}
