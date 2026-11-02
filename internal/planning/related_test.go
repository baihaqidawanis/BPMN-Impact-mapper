package planning_test

import (
	"context"
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"strings"
	"testing"
)

func TestPlanningRelatedFilesAreBoundedDeduplicatedAndUnranked(t *testing.T) {
	c := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "T"}, Impacts: []domain.Impact{{File: "store.go", Confidence: .8}}, RelatedFiles: []domain.RelatedFile{
		{File: "store.go", Evidence: []domain.Evidence{{Excerpt: "duplicate"}}},
		{File: "ui.tsx", Evidence: []domain.Evidence{{Type: "HTTP_ROUTE_PATTERN", Location: "ui.tsx:2", Excerpt: strings.Repeat("α", 500)}}},
		{File: "ui.tsx", Evidence: []domain.Evidence{{Excerpt: "duplicate"}}},
		{File: "no-proof.ts"},
	}}
	r := domain.Report{Changes: []domain.Change{c}}
	b := planning.BuildEvidenceBundle(r, planning.DefaultLimits())
	candidates := b.Changes[0].Candidates
	if len(candidates) != 2 || candidates[0].RetrievalScore == nil || candidates[1].RetrievalScore != nil || candidates[1].Origin != "source-caller" || candidates[1].SuggestedAction != domain.ActionVerify {
		t.Fatalf("ranked and related inputs mixed: %+v", candidates)
	}
	if len([]rune(candidates[1].Evidence[0].Excerpt)) != 240 || candidates[1].Evidence[0].ID == "" {
		t.Fatal("missing bounded evidence")
	}
	data, _ := json.Marshal(candidates[1])
	if strings.Contains(string(data), "retrieval_score") {
		t.Fatal("invented zero confidence")
	}
	disabled := planning.BuildEvidenceBundle(r, planning.Limits{RelatedPerChange: -1})
	if len(disabled.Changes[0].Candidates) != 1 {
		t.Fatal("related limit disable failed")
	}
}

func TestPlanningGetsDirectedConditionsAndCannotApproveIncompleteContext(t *testing.T) {
	r := fixtureReport()
	r.Changes[0].ChangeType = domain.ModifyGatewayCondition
	r.Changes[0].OldElement = &domain.Element{ID: "G", Name: "Allowed?", Type: "exclusiveGateway", Actor: "Reviewer"}
	r.Changes[0].NewElement = &domain.Element{ID: "G", Name: "Allowed?", Type: "exclusiveGateway", Actor: "Reviewer"}
	r.Changes[0].Context = &domain.ChangeContext{BranchChanges: []domain.BranchChange{{FlowID: "F", OldCondition: "state == REJECTED", NewCondition: "state == REJECTED || state == CANCELLED"}}}
	b := planning.BuildEvidenceBundle(r, planning.DefaultLimits())
	if b.Changes[0].Before.Actor != "Reviewer" || b.Changes[0].BranchChanges[0].OldCondition == b.Changes[0].BranchChanges[0].NewCondition || b.Changes[0].ContextIncomplete {
		t.Fatal("lost directed requirement")
	}
	for _, kind := range []string{"missing", "truncated", "partial"} {
		copy := r
		copy.Changes = append([]domain.Change(nil), r.Changes...)
		if kind == "missing" {
			copy.Changes[0].Context = nil
		}
		if kind == "truncated" {
			element := *copy.Changes[0].NewElement
			element.Documentation = strings.Repeat("x", 2000)
			copy.Changes[0].NewElement = &element
		}
		if kind == "partial" {
			copy.Warnings = []domain.Warning{{Code: "BPMN_PARTIAL_SUPPORT", Message: "unsupported execution semantics"}}
		}
		plan, err := (planning.Orchestrator{Provider: &fakeProvider{}, Limits: planning.DefaultLimits()}).Run(context.Background(), copy)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Status != planning.StatusNeedsReview || len(plan.ReviewIssues) == 0 {
			t.Fatal("model approval overrode incomplete analysis", kind)
		}
	}
}
