package main

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/selection"
	"testing"
)

func TestScoreUsesActualSelectedCountAndPreservesDeferredFiles(t *testing.T) {
	a := domain.Impact{File: "a.go", Evidence: []domain.Evidence{{Type: "RRF_FUSION", Excerpt: "left_rank=2;right_rank=3"}}}
	b := domain.Impact{File: "b.go", Evidence: []domain.Evidence{{Type: "RRF_FUSION", Excerpt: "left_rank=2;right_rank=30"}}}
	c := sample{scenario: "fixture", prediction: domain.Report{Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "A"}, Impacts: []domain.Impact{a, a, b}}}}, truth: evaluation.GroundTruth{Cases: []evaluation.Case{{ChangeType: domain.AddTask, ElementID: "A", Expected: []evaluation.ExpectedImpact{{File: "a.go"}}}}}}
	all, _, err := score([]sample{c}, selection.Policy{Name: "all"})
	if err != nil {
		t.Fatal(err)
	}
	narrow, records, err := score([]sample{c}, selection.Policy{Name: "both-within", Bound: 5})
	if err != nil {
		t.Fatal(err)
	}
	if all.Selected != 2 || all.Precision != 0.5 || narrow.Selected != 1 || narrow.Precision != 1 || narrow.Recall != 1 || len(records[0].Deferred) != 1 || records[0].Deferred[0] != "b.go" {
		t.Fatal(all, narrow, records)
	}
	if len(c.prediction.Changes[0].Impacts) != 3 {
		t.Fatal("mutated saved report")
	}
}
