package review_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
	"testing"
)

func TestAdjudicationRequiresMissingCaseDecisionAndRejectsExtra(t *testing.T) {
	item := evaluation.Case{ChangeType: domain.AddTask, ElementID: "x", Expected: []evaluation.ExpectedImpact{}}
	a := review.Annotation{Version: "1", ReviewerID: "a", Cases: []evaluation.Case{item}}
	b := review.Annotation{Version: "1", ReviewerID: "b", Cases: []evaluation.Case{}}
	if _, err := review.Adjudicate(a, b, nil); err == nil {
		t.Fatal("missing case treated as negative")
	}
	if _, err := review.Adjudicate(a, b, []evaluation.Case{item}); err != nil {
		t.Fatal(err)
	}
	if _, err := review.Adjudicate(a, b, []evaluation.Case{item, item}); err == nil {
		t.Fatal("duplicate decision accepted")
	}
	b.Cases = []evaluation.Case{item}
	if _, err := review.Adjudicate(a, b, []evaluation.Case{item}); err == nil {
		t.Fatal("agreement overwritten")
	}
	for _, file := range []string{"C:/private/key", ".", "../private", "/private"} {
		item.Expected = []evaluation.ExpectedImpact{{File: file, Action: domain.ActionModify}}
		if err := (evaluation.GroundTruth{Version: "1", Cases: []evaluation.Case{item}}).Validate(); err == nil {
			t.Fatalf("unsafe path accepted: %s", file)
		}
	}
}
