package semantic

import (
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestSemanticQueryFocusesOnMeaningfulProcessWords(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "ACT-773", Name: "Clarify service request", Type: "userTask", Actor: "Operator"},
		Context:    &domain.ChangeContext{Predecessors: []domain.ElementContext{{ID: "ACT-101", Name: "Submit service request"}}},
	}
	query := queryText(change)
	for _, expected := range []string{"Clarify service request"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("missing process meaning in %q", query)
		}
	}
	for _, noise := range []string{"ACT-773", "ADD_TASK", "userTask", "Operator", "BPMN change", "Submit service request"} {
		if strings.Contains(query, noise) {
			t.Fatalf("boilerplate in semantic query: %q", query)
		}
	}
	change.ChangeType = domain.ModifyActorRole
	if !strings.Contains(queryText(change), "Operator") {
		t.Fatal("role change lost actor meaning")
	}
}

func TestSemanticQueryRetainsContextForUnlabelledOrTopologyChange(t *testing.T) {
	context := &domain.ChangeContext{Predecessors: []domain.ElementContext{{Name: "Submit service request"}}, Successors: []domain.ElementContext{{Name: "Review service request"}}}
	for _, change := range []domain.Change{
		{ChangeType: domain.AddTask, NewElement: &domain.Element{Type: "userTask"}, Context: context},
		{ChangeType: domain.ModifySequenceFlow, Context: context},
	} {
		query := queryText(change)
		if !strings.Contains(query, "Submit service request") || !strings.Contains(query, "Review service request") {
			t.Fatalf("topology context lost for %s: %q", change.ChangeType, query)
		}
	}
}

func TestNamedTaskQueryKeepsShortProcessAnchorWithoutInstructionNoise(t *testing.T) {
	change := domain.Change{
		ChangeType: domain.AddTask,
		NewElement: &domain.Element{ID: "GENERATED-77", Name: "Upload required supporting PDF", Type: "userTask",
			Documentation: "Scenario instructions for Staff: store one PDF with the purchase request draft; reject invalid input atomically."},
		Context: &domain.ChangeContext{Pattern: domain.PatternInsertTask,
			Predecessors: []domain.ElementContext{{Name: "Enter purchase request details"}},
			Successors:   []domain.ElementContext{{Name: "Validate request details"}}},
	}
	got := queryText(change)
	if got != "Upload required supporting PDF purchase request" {
		t.Fatalf("query = %q", got)
	}
}

func TestHybridCanUseExplicitProcessNameWithoutChangingSemanticBaseline(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{Name: "Attach supporting document", Type: "userTask"}, Context: &domain.ChangeContext{ProcessName: "Expense reimbursement", Predecessors: []domain.ElementContext{{ID: "GENERATED-88", Name: "Start"}}}}
	plain := (Strategy{}).queryText(change)
	contextual := (Strategy{IncludeProcessContext: true}).queryText(change)
	if plain != "Attach supporting document" || !strings.Contains(contextual, "Expense reimbursement") || strings.Contains(contextual, "GENERATED-88") {
		t.Fatalf("plain=%q contextual=%q", plain, contextual)
	}
	change.Context.ProcessName = strings.Repeat("x", 500)
	if len((Strategy{IncludeProcessContext: true}).queryText(change)) > len(plain)+171 {
		t.Fatal("unbounded process context")
	}
}
