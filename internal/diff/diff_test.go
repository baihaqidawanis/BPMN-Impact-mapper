package diff_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestCompareDetectsRequiredTaxonomy(t *testing.T) {
	oldGraph := parseFixture(t, "old.bpmn")
	newGraph := parseFixture(t, "new.bpmn")
	changes := diff.Compare(oldGraph, newGraph)
	found := map[domain.ChangeType]bool{}
	for _, change := range changes {
		found[change.ChangeType] = true
	}
	for _, required := range []domain.ChangeType{domain.AddTask, domain.RemoveTask, domain.ModifyTaskLabel, domain.ModifyActorRole, domain.ModifyGatewayCondition, domain.ModifySequenceFlow, domain.AddDataRequirement, domain.AddState, domain.ModifyTerminalOutcome} {
		if !found[required] {
			t.Errorf("missing %s in %#v", required, found)
		}
	}
}

func TestCompareAbstainsOnAmbiguousStructuralMatch(t *testing.T) {
	oldGraph := graphWithElements(&domain.Element{ID: "old", Name: "Review", Type: "userTask", Actor: "Manager"})
	newGraph := graphWithElements(
		&domain.Element{ID: "new-a", Name: "Review", Type: "userTask", Actor: "Manager"},
		&domain.Element{ID: "new-b", Name: "Review", Type: "userTask", Actor: "Manager"},
	)
	changes := diff.Compare(oldGraph, newGraph)
	for _, change := range changes {
		if change.ChangeType == domain.UnresolvedMatch {
			return
		}
	}
	t.Fatalf("expected unresolved match, got %#v", changes)
}

func TestCompareRecognizesCompositeChangeContext(t *testing.T) {
	tests := []struct {
		name       string
		oldFixture string
		newFixture string
		changeType domain.ChangeType
		elementID  string
		pattern    domain.ChangePattern
	}{
		{name: "insert task", oldFixture: "insert-task-old.bpmn", newFixture: "insert-task-new.bpmn", changeType: domain.AddTask, elementID: "TASK-REVIEW", pattern: domain.PatternInsertTask},
		{name: "remove task with rewire", oldFixture: "remove-task-old.bpmn", newFixture: "remove-task-new.bpmn", changeType: domain.RemoveTask, elementID: "TASK-REVIEW", pattern: domain.PatternRemoveTaskWithRewire},
		{name: "add gateway branch", oldFixture: "add-branch-old.bpmn", newFixture: "add-branch-new.bpmn", changeType: domain.AddGateway, elementID: "GW-ROUTE", pattern: domain.PatternAddGatewayBranch},
		{name: "remove gateway branch", oldFixture: "remove-branch-old.bpmn", newFixture: "remove-branch-new.bpmn", changeType: domain.RemoveGateway, elementID: "GW-ROUTE", pattern: domain.PatternRemoveGatewayBranch},
		{name: "move task", oldFixture: "move-task-old.bpmn", newFixture: "move-task-new.bpmn", changeType: domain.ModifySequenceFlow, elementID: "F2", pattern: domain.PatternMoveTask},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changes := diff.Compare(parseFixture(t, test.oldFixture), parseFixture(t, test.newFixture))
			for _, change := range changes {
				element := change.NewElement
				if element == nil {
					element = change.OldElement
				}
				if change.ChangeType == test.changeType && element != nil && element.ID == test.elementID {
					if change.Context == nil || change.Context.Pattern != test.pattern {
						t.Fatalf("context=%+v", change.Context)
					}
					if len(change.Context.Predecessors) == 0 || len(change.Context.Successors) == 0 || len(change.Context.RelatedFlowIDs) == 0 {
						t.Fatalf("incomplete context=%+v", change.Context)
					}
					return
				}
			}
			t.Fatalf("missing %s for %s in %#v", test.pattern, test.elementID, changes)
		})
	}
}

func TestCompareDoesNotStructurallyMatchAcrossProcesses(t *testing.T) {
	oldGraph := graphWithElements(&domain.Element{ID: "OLD-TASK", Name: "Review", Type: "userTask", Actor: "Manager", ProcessID: "PROCESS-A"})
	newGraph := graphWithElements(&domain.Element{ID: "NEW-TASK", Name: "Review", Type: "userTask", Actor: "Manager", ProcessID: "PROCESS-B"})
	changes := diff.Compare(oldGraph, newGraph)
	foundAdd, foundRemove := false, false
	for _, change := range changes {
		foundAdd = foundAdd || change.ChangeType == domain.AddTask
		foundRemove = foundRemove || change.ChangeType == domain.RemoveTask
		if change.ChangeType == domain.UnresolvedMatch {
			t.Fatalf("cross-process elements must not become an ambiguous match: %#v", changes)
		}
	}
	if !foundAdd || !foundRemove {
		t.Fatalf("expected independent add/remove, got %#v", changes)
	}
}

func TestCompositeContextsDoNotOverclaimStableTopology(t *testing.T) {
	oldGraph := graphWithFlow("F", "A", "B", "old condition", &domain.Element{ID: "A", Name: "Prepare", Type: "task", ProcessID: "P"}, &domain.Element{ID: "B", Name: "Review", Type: "task", ProcessID: "P"})
	newGraph := graphWithFlow("F", "A", "B", "new condition", &domain.Element{ID: "A", Name: "Prepare", Type: "task", ProcessID: "P"}, &domain.Element{ID: "B", Name: "Review", Type: "task", ProcessID: "P"})
	changes := diff.Compare(oldGraph, newGraph)
	if len(changes) != 1 || changes[0].ChangeType != domain.ModifySequenceFlow || changes[0].Context != nil {
		t.Fatalf("condition-only flow change was overclassified: %#v", changes)
	}
	if unchanged := diff.Compare(oldGraph, oldGraph); len(unchanged) != 0 {
		t.Fatalf("unchanged graph produced changes: %#v", unchanged)
	}
}

func graphWithElements(elements ...*domain.Element) *domain.Graph {
	graph := &domain.Graph{Elements: map[string]*domain.Element{}, Flows: map[string]*domain.Flow{}, Incoming: map[string][]string{}, Outgoing: map[string][]string{}}
	for _, element := range elements {
		graph.Elements[element.ID] = element
	}
	return graph
}

func graphWithFlow(id, source, target, condition string, elements ...*domain.Element) *domain.Graph {
	graph := graphWithElements(elements...)
	graph.Flows[id] = &domain.Flow{ID: id, SourceRef: source, TargetRef: target, Condition: condition}
	graph.Outgoing[source] = []string{id}
	graph.Incoming[target] = []string{id}
	graph.Elements[source].Outgoing = []string{id}
	graph.Elements[target].Incoming = []string{id}
	return graph
}

func parseFixture(t *testing.T, name string) *domain.Graph {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bpmn", name))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := bpmn.Parse(data, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return graph
}
