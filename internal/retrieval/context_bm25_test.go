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

func TestStartEndLabelAblationDoesNotMutateRecordedContext(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "NEW", Name: "Inspect material", Type: "userTask"},
		Context: &domain.ChangeContext{ProcessName: "Material inspection",
			Predecessors: []domain.ElementContext{{ID: "S", Name: "Generic trigger", Type: "startEvent"}},
			Successors:   []domain.ElementContext{{ID: "G", Name: "Accept shipment", Type: "exclusiveGateway"}, {ID: "E", Name: "Generic finish", Type: "endEvent"}}},
	}
	artifacts := []domain.Artifact{
		{ID: "noise", File: "noise.go", Content: "generic trigger finish", Tokens: textutil.Tokenize("generic trigger finish")},
		{ID: "target", File: "target.go", Content: "accept shipment", Tokens: textutil.Tokenize("accept shipment")},
	}
	baseline, _, err := (retrieval.ContextBM25{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 10})
	if err != nil || len(baseline[0].Impacts) != 2 {
		t.Fatalf("baseline=%#v err=%v", baseline, err)
	}
	strategy := retrieval.ContextBM25{OmitStartEndLabels: true}
	result, _, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 10})
	if err != nil || len(result[0].Impacts) != 1 || result[0].Impacts[0].File != "target.go" {
		t.Fatalf("ablation=%#v err=%v", result, err)
	}
	if change.Context.Predecessors[0].Name != "Generic trigger" || result[0].Context.Successors[1].Name != "Generic finish" {
		t.Fatal("query ablation mutated recorded BPMN context")
	}
	if strategy.MethodMetadata()["query_ablation"] == "" || (retrieval.ContextBM25{}).MethodMetadata()["query_ablation"] != "" {
		t.Fatal("ablation metadata missing or leaked into default")
	}
}

func TestStartEndLabelAblationKeepsChangedEventText(t *testing.T) {
	change := domain.Change{ChangeType: domain.ModifyTerminalOutcome,
		NewElement: &domain.Element{ID: "E", Name: "Refund completed", Type: "endEvent"}}
	artifacts := []domain.Artifact{{ID: "a", File: "refund.go", Content: "refund completed", Tokens: textutil.Tokenize("refund completed")}}
	result, _, err := (retrieval.ContextBM25{OmitStartEndLabels: true}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 10})
	if err != nil || len(result[0].Impacts) != 1 {
		t.Fatalf("changed event text was dropped: %#v err=%v", result, err)
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

func TestContextBM25IncludesProcessNameButPlainBaselineDoesNot(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "NEW", Name: "Upload document", Type: "userTask"},
		Context:    &domain.ChangeContext{ProcessName: "Expense reimbursement"},
	}
	artifacts := []domain.Artifact{
		{ID: "a", File: "expense.go", Content: "expense reimbursement", Tokens: textutil.Tokenize("expense reimbursement")},
		{ID: "b", File: "upload.go", Content: "upload document", Tokens: textutil.Tokenize("upload document")},
	}
	contextResult, _, err := (retrieval.ContextBM25{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || len(contextResult[0].Impacts) != 2 {
		t.Fatalf("process name omitted: result=%#v err=%v", contextResult, err)
	}
	plainResult, _, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || len(plainResult[0].Impacts) != 1 || plainResult[0].Impacts[0].File != "upload.go" {
		t.Fatalf("plain baseline used process context: result=%#v err=%v", plainResult, err)
	}
}

func TestContextBM25UsesChangedGatewayBranchWithoutInventingDirectTrace(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.ModifyGatewayCondition,
		OldElement: &domain.Element{ID: "G", Name: "Eligible?", Type: "exclusiveGateway"},
		NewElement: &domain.Element{ID: "G", Name: "Eligible?", Type: "exclusiveGateway"},
		Context:    &domain.ChangeContext{ProcessName: "Expense approval", BranchTerms: []string{"within budget"}, Successors: []domain.ElementContext{{ID: "T", Name: "Pay claim", Type: "task"}}},
	}
	artifacts := []domain.Artifact{
		{ID: "a", File: "approval.go", Content: "approve expense within budget before paying claim", Tokens: textutil.Tokenize("approve expense within budget before paying claim")},
		{ID: "b", File: "unrelated.go", Content: "render homepage", Tokens: textutil.Tokenize("render homepage")},
	}
	result, _, err := (retrieval.ContextBM25{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
	if err != nil || len(result[0].Impacts) != 1 || result[0].Impacts[0].File != "approval.go" {
		t.Fatalf("changed branch did not retrieve relevant source: result=%#v err=%v", result, err)
	}
	if hasEvidenceType(result[0].Impacts[0].Evidence, "TRACE_ID") || result[0].Impacts[0].RequiredAction != domain.ActionVerify {
		t.Fatalf("branch context was mistaken for direct proof: %#v", result[0].Impacts[0])
	}
}
