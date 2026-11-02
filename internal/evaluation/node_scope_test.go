package evaluation

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"testing"
)

func TestNodeUnionCountsSharedFileOnceAndDeduplicatesBeforeK(t *testing.T) {
	impact := func(file string) domain.Impact { return domain.Impact{File: file} }
	report := domain.Report{Changes: []domain.Change{
		{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "A"}, Impacts: []domain.Impact{impact("a.go"), impact("a.go"), impact("extra.go")}},
		{ChangeType: domain.AddGateway, NewElement: &domain.Element{ID: "B"}, Impacts: []domain.Impact{impact("a.go"), impact("b.go")}},
		{ChangeType: domain.ModifySequenceFlow, NewElement: &domain.Element{ID: "F"}, Impacts: []domain.Impact{impact("flow-only.go")}},
	}}
	truth := GroundTruth{Version: "1", Cases: []Case{
		{ChangeType: domain.AddTask, ElementID: "A", Expected: []ExpectedImpact{{File: "a.go", Action: domain.ActionModify}}},
		{ChangeType: domain.AddGateway, ElementID: "B", Expected: []ExpectedImpact{{File: "a.go", Action: domain.ActionModify}, {File: "b.go", Action: domain.ActionModify}}},
	}}
	union, err := NodeFileUnion(report, truth, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(union.Predicted) != 3 || union.Precision != 2.0/3 || union.Recall != 1 || len(union.Extra) != 1 || union.Extra[0] != "extra.go" {
		t.Fatal(union)
	}
	if len(report.Changes) != 3 {
		t.Fatal("projection mutated source report")
	}
	truth.Cases = truth.Cases[:1]
	if _, err := NodeFileUnion(report, truth, 2); err == nil {
		t.Fatal("incomplete node labels accepted")
	}
}
