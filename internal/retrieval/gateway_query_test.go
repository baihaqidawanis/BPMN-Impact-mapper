package retrieval_test

import (
	"context"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
)

func TestBM25GatewayUsesBranchConditionsWithoutNeighbourExpansion(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "rule", File: "eligibility.go", Content: "budget available", Tokens: textutil.Tokenize("budget available")},
		{ID: "neighbor", File: "payment.go", Content: "transfer claim", Tokens: textutil.Tokenize("transfer claim")},
		{ID: "process", File: "refund.go", Content: "expense reimbursement", Tokens: textutil.Tokenize("expense reimbursement")},
	}
	for _, kind := range []domain.ChangeType{domain.AddGateway, domain.RemoveGateway, domain.ModifyGatewayCondition} {
		change := domain.Change{ChangeType: kind, OldElement: &domain.Element{ID: "G", Type: "exclusiveGateway", Name: "Eligible?"}, NewElement: &domain.Element{ID: "G", Type: "exclusiveGateway", Name: "Eligible?"}, Context: &domain.ChangeContext{BranchTerms: []string{"budget available"}, ProcessName: "Expense reimbursement", Successors: []domain.ElementContext{{Name: "Transfer claim"}}}}
		result, _, err := (retrieval.BM25Only{}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.DefaultConfig())
		if err != nil || len(result[0].Impacts) != 1 || result[0].Impacts[0].File != "eligibility.go" {
			t.Fatalf("%s lost condition or included neighbours: %#v %v", kind, result, err)
		}
		if change.Context.ProcessName != "Expense reimbursement" {
			t.Fatal("input context mutated")
		}
	}
}
