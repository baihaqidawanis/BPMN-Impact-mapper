package review_test

import (
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
)

func TestReconcileKeepsOnlyExactAgreement(t *testing.T) {
	agreed := evaluation.Case{ChangeType: domain.AddTask, ElementID: "A", Expected: []evaluation.ExpectedImpact{{File: "a.go", Action: domain.ActionAdd}}}
	leftConflict := evaluation.Case{ChangeType: domain.ModifyTaskLabel, ElementID: "B", Expected: []evaluation.ExpectedImpact{{File: "b.go", Action: domain.ActionModify}}}
	rightConflict := evaluation.Case{ChangeType: domain.ModifyTaskLabel, ElementID: "B", Expected: []evaluation.ExpectedImpact{{File: "b.go", Action: domain.ActionVerify}}}
	result, err := review.Reconcile(review.Annotation{Version: "1", ReviewerID: "a", Cases: []evaluation.Case{agreed, leftConflict}}, review.Annotation{Version: "1", ReviewerID: "b", Cases: []evaluation.Case{rightConflict, agreed}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Agreed.Cases) != 1 || result.Agreed.Cases[0].ElementID != "A" || len(result.Disagreements) != 1 || result.Disagreements[0].ElementID != "B" {
		t.Fatalf("result=%#v", result)
	}
}

func TestReconcileRejectsSameReviewer(t *testing.T) {
	annotation := review.Annotation{Version: "1", ReviewerID: "same", Cases: []evaluation.Case{}}
	if _, err := review.Reconcile(annotation, annotation); err == nil {
		t.Fatal("expected independent reviewer error")
	}
}
