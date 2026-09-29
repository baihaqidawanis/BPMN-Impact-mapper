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

func graphWithElements(elements ...*domain.Element) *domain.Graph {
	graph := &domain.Graph{Elements: map[string]*domain.Element{}, Flows: map[string]*domain.Flow{}, Incoming: map[string][]string{}, Outgoing: map[string][]string{}}
	for _, element := range elements {
		graph.Elements[element.ID] = element
	}
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
