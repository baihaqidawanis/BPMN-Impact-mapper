package evaluation_test

import (
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

func TestPositiveRankingExcludesEmptyCasesAndChecksTopReferenceCount(t *testing.T) {
	p := domain.Report{Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "T"}, Impacts: []domain.Impact{{File: "noise"}, {File: "a"}, {File: "b"}, {File: "other"}, {File: "c"}}}}}
	truth := evaluation.GroundTruth{Version: "1", Cases: []evaluation.Case{{ChangeType: domain.AddTask, ElementID: "T", Expected: []evaluation.ExpectedImpact{{File: "a", Action: domain.ActionModify}, {File: "b", Action: domain.ActionModify}, {File: "c", Action: domain.ActionModify}}}, {ChangeType: domain.ModifySequenceFlow, ElementID: "F"}}}
	r, err := evaluation.Evaluate(p, []byte("p"), truth, []byte("t"))
	if err != nil {
		t.Fatal(err)
	}
	s := r.Metrics.PositiveRanking
	if s == nil || s.Count != 1 || s.RPrecision != 0.6667 || s.MAP != 0.5889 || s.MRR != 0.5 || s.Precision5 != 0.6 || s.Recall5 != 1 {
		t.Fatalf("metrics: %#v", s)
	}
	if r.Changes[1].RPrecision != nil {
		t.Fatal("empty case has a defined R-precision")
	}
	truth.Cases = truth.Cases[1:]
	r, err = evaluation.Evaluate(p, []byte("p"), truth, []byte("t"))
	if err != nil || r.Metrics.PositiveRanking != nil {
		t.Fatal("negative-only ranking must be undefined")
	}
}
