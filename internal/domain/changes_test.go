package domain

import "testing"

func TestIndependentChangesKeepsOrphansAndRawInput(t *testing.T) {
	changes := []Change{
		{ChangeType: AddTask, NewElement: &Element{ID: "task"}},
		{ChangeType: ModifySequenceFlow, NewElement: &Element{ID: "flow"}, PartOf: "task"},
		{ChangeType: ModifySequenceFlow, NewElement: &Element{ID: "orphan"}, PartOf: "missing"},
	}
	independent := IndependentChanges(changes)
	if len(independent) != 2 || independent[0].NewElement.ID != "task" || independent[1].NewElement.ID != "orphan" {
		t.Fatalf("independent=%#v", independent)
	}
	if len(changes) != 3 || changes[1].PartOf != "task" {
		t.Fatalf("raw changes mutated: %#v", changes)
	}
}
