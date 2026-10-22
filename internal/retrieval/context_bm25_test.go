package retrieval_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestContextBM25UsesCluesOnlyAsEvidence(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW-TASK", Name: "Validate request", Type: "userTask"}}
	artifacts := []domain.Artifact{
		{ID: "a", File: "request.go", Symbol: domain.Symbol{Name: "ValidateRequest", Kind: "function", StartLine: 1}, Content: `func ValidateRequest() {}`, Tokens: textutil.Tokenize("validate request")},
		{ID: "b", File: "other.go", Content: `// NEW-TASK\nvalidate request`, Tokens: textutil.Tokenize("validate request")},
	}
	strategy := retrieval.ContextBM25{}
	first, _, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatalf("non-deterministic ranking: %v", err)
	}
	impacts := first[0].Impacts
	if len(impacts) != 2 || impacts[0].ScoreBreakdown.BM25 != impacts[1].ScoreBreakdown.BM25 {
		t.Fatalf("clues changed BM25 ranking: %#v", impacts)
	}
	if impacts[0].ScoreBreakdown.EvidenceScore != nil || impacts[1].ScoreBreakdown.EvidenceScore != nil {
		t.Fatalf("legacy weighted score leaked into hybrid: %#v", impacts)
	}
	if impacts[0].RequiredAction != domain.ActionVerify || impacts[1].RequiredAction != domain.ActionVerify {
		t.Fatalf("task ID alone promoted action: %#v", impacts)
	}
	if !hasEvidenceType(impacts[1].Evidence, "TRACE_ID") {
		t.Fatalf("direct ID clue was lost: %#v", impacts[1].Evidence)
	}
}

func TestContextBM25UsesNeighborsUnlikePlainBM25(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "NEW", Name: "Upload supporting document", Type: "userTask"},
		Context:    &domain.ChangeContext{Predecessors: []domain.ElementContext{{Name: "Create purchase request"}}},
	}
	artifacts := []domain.Artifact{
		{ID: "a", File: "purchase.go", Content: "create purchase request", Tokens: textutil.Tokenize("create purchase request")},
		{ID: "b", File: "upload.go", Content: "upload supporting document", Tokens: textutil.Tokenize("upload supporting document")},
	}
	contextResult, _, err := (retrieval.ContextBM25{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || len(contextResult[0].Impacts) != 2 {
		t.Fatalf("context query omitted a neighbor match: result=%#v err=%v", contextResult, err)
	}
	plainResult, _, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || len(plainResult[0].Impacts) != 1 || plainResult[0].Impacts[0].File != "upload.go" {
		t.Fatalf("plain BM25 unexpectedly used BPMN neighbors: result=%#v err=%v", plainResult, err)
	}
}

func hasEvidenceType(evidence []domain.Evidence, kind string) bool {
	for _, item := range evidence {
		if item.Type == kind {
			return true
		}
	}
	return false
}
