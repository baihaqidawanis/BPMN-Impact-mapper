package retrieval_test

import (
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestAttachRequiresEvidenceAndExplainsStrongAction(t *testing.T) {
	content := `// ValidateRequest handles TASK-VALIDATE and writes an audit action.
func ValidateRequest() string { return "REVIEWED" }`
	artifact := domain.Artifact{
		ID: "internal/application/request.go#ValidateRequest", File: "internal/application/request.go",
		Module: "internal", Layer: "application", Language: "Go",
		Symbol:  domain.Symbol{Name: "ValidateRequest", Kind: "function", StartLine: 10},
		Content: content, Tokens: textutil.Tokenize("internal application request ValidateRequest " + content),
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-VALIDATE", Name: "Validate request", Type: "serviceTask"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	changes, _ := retrieval.Attach([]domain.Change{change}, []domain.Artifact{artifact}, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 1 {
		t.Fatalf("impacts=%#v", changes[0].Impacts)
	}
	impact := changes[0].Impacts[0]
	if len(impact.Evidence) == 0 || impact.ScoreBreakdown.TraceID != 1 {
		t.Fatalf("impact lacks evidence: %#v", impact)
	}
	if impact.RequiredAction != domain.ActionAdd {
		t.Fatalf("action=%s confidence=%.4f breakdown=%#v", impact.RequiredAction, impact.Confidence, impact.ScoreBreakdown)
	}
}

func TestAttachAbstainsWithoutEvidence(t *testing.T) {
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, NewElement: &domain.Element{ID: "TASK-X", Name: "Rare phrase", Type: "task"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	changes, _ := retrieval.Attach([]domain.Change{change}, nil, retrieval.DefaultConfig())
	if len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) == 0 {
		t.Fatalf("expected abstention: %#v", changes[0])
	}
}

func TestBM25OnlyProducesVerifyCandidatesDeterministically(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "b", File: "b.go", Module: ".", Layer: "unknown", Content: "validate request", Tokens: []string{"validate", "request"}},
		{ID: "a", File: "a.go", Module: ".", Layer: "unknown", Content: "validate request", Tokens: []string{"validate", "request"}},
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK", Name: "Validate request", Type: "task"}, Impacts: []domain.Impact{}, UnresolvedImpact: []string{}}
	first, warnings := (retrieval.BM25Only{}).Attach([]domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	second, _ := (retrieval.BM25Only{}).Attach([]domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if len(warnings) != 0 || len(first[0].Impacts) != 2 {
		t.Fatalf("impacts=%#v warnings=%#v", first[0].Impacts, warnings)
	}
	if first[0].Impacts[0].File != "a.go" || first[0].Impacts[0].RequiredAction != domain.ActionVerify || first[0].Impacts[0].Evidence[0].Type != "BM25" {
		t.Fatalf("first candidate=%#v", first[0].Impacts[0])
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("BM25-only output is not deterministic")
	}
}
