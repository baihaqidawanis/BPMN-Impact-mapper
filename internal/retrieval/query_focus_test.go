package retrieval

import (
	"slices"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestNamedAddedTaskQueryExcludesLongInstructionButKeepsProcessContext(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "GENERATED-77", Name: "Upload required supporting PDF", Type: "userTask",
			Documentation: "Staff must store the document atomically with the purchase request draft."},
		Context: &domain.ChangeContext{Pattern: domain.PatternInsertTask,
			Predecessors: []domain.ElementContext{{Name: "Enter purchase request details"}},
			Successors:   []domain.ElementContext{{Name: "Validate request details"}}},
	}
	query, ids := queryFor(change)
	for _, token := range []string{"upload", "supporting", "pdf", "purchase", "request"} {
		if !slices.Contains(query, token) {
			t.Fatalf("query lost %q: %v", token, query)
		}
	}
	for _, noise := range []string{"staff", "atomically", "document", "draft"} {
		if slices.Contains(query, noise) {
			t.Fatalf("instruction noise %q entered query: %v", noise, query)
		}
	}
	if len(ids) != 1 || ids[0] != "GENERATED-77" {
		t.Fatalf("direct IDs changed: %v", ids)
	}
	structural := structuralQueryFor(change, query)
	if !slices.Contains(structural, "purchase") || !slices.Contains(structural, "request") || slices.Contains(structural, "atomically") {
		t.Fatalf("structural query leaked instruction text: %v", structural)
	}
}
