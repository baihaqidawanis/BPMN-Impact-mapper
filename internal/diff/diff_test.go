package diff_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestParsedProcessNameSurvivesInsertAndRemoveContext(t *testing.T) {
	parse := func(body string) *domain.Graph {
		t.Helper()
		xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P" name="Expense reimbursement"><startEvent id="S"/><endEvent id="E"/>` + body + `</process></definitions>`
		graph, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		return graph
	}
	oldGraph := parse(`<sequenceFlow id="F" sourceRef="S" targetRef="E"/>`)
	newGraph := parse(`<userTask id="T" name="Attach receipt"/><sequenceFlow id="F" sourceRef="S" targetRef="T"/><sequenceFlow id="G" sourceRef="T" targetRef="E"/>`)
	for _, test := range []struct {
		old, next *domain.Graph
		kind      domain.ChangeType
	}{
		{oldGraph, newGraph, domain.AddTask}, {newGraph, oldGraph, domain.RemoveTask},
	} {
		found := false
		for _, change := range diff.Compare(test.old, test.next) {
			if change.ChangeType != test.kind {
				continue
			}
			found = true
			if change.Context == nil || change.Context.ProcessName != "Expense reimbursement" {
				t.Fatalf("missing process context: %#v", change)
			}
		}
		if !found {
			t.Fatalf("missing %s", test.kind)
		}
	}
}

func TestModifiedGatewayConditionIncludesChangedBranchContext(t *testing.T) {
	parse := func(condition string) *domain.Graph {
		t.Helper()
		xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P" name="Expense approval"><task id="Submit" name="Submit claim"/><exclusiveGateway id="Decision" name="Eligible?"/><task id="Pay" name="Pay claim"/><task id="Reject" name="Reject claim"/><sequenceFlow id="F1" sourceRef="Submit" targetRef="Decision"/><sequenceFlow id="F2" sourceRef="Decision" targetRef="Pay" name="Approved"><conditionExpression>` + condition + `</conditionExpression></sequenceFlow><sequenceFlow id="F3" sourceRef="Decision" targetRef="Reject" name="Rejected"/></process></definitions>`
		graph, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		return graph
	}
	changes := diff.Compare(parse("amount approved"), parse("amount approved and within budget"))
	for _, change := range changes {
		if change.ChangeType != domain.ModifyGatewayCondition {
			continue
		}
		context := change.Context
		if context == nil || context.ProcessName != "Expense approval" || context.Pattern != domain.PatternModifyGatewayBranch {
			t.Fatalf("missing gateway context: %#v", change)
		}
		if !strings.Contains(strings.Join(context.BranchTerms, " "), "within budget") || !strings.Contains(strings.Join(context.BranchTerms, " "), "amount approved") {
			t.Fatalf("changed branch conditions missing: %#v", context.BranchTerms)
		}
		if len(context.Predecessors) != 1 || len(context.Successors) != 2 || !strings.Contains(strings.Join(context.RelatedFlowIDs, " "), "F2") {
			t.Fatalf("gateway neighborhood missing: %#v", context)
		}
		return
	}
	t.Fatal("MODIFY_GATEWAY_CONDITION not detected")
}

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

func TestCompareDetectsGatewaySemanticsWithStableID(t *testing.T) {
	oldGraph := graphWithElements(&domain.Element{ID: "GW", Name: "Review decision", Type: "exclusiveGateway", GatewayDirection: "Diverging", DefaultFlow: "FLOW-REJECT"})
	newGraph := graphWithElements(&domain.Element{ID: "GW", Name: "Review decision", Type: "parallelGateway", GatewayDirection: "Converging"})
	changes := diff.Compare(oldGraph, newGraph)
	if len(changes) != 1 || changes[0].ChangeType != domain.ModifyGatewaySemantics {
		t.Fatalf("expected gateway semantic change, got %#v", changes)
	}
	for _, term := range []string{"exclusiveGateway", "parallelGateway", "Diverging", "Converging", "FLOW-REJECT"} {
		if !strings.Contains(changes[0].Reason, term) {
			t.Fatalf("reason omitted %q: %s", term, changes[0].Reason)
		}
	}
	if got := diff.Compare(oldGraph, oldGraph); len(got) != 0 {
		t.Fatalf("unchanged gateway produced changes: %#v", got)
	}
	for _, test := range []struct {
		name    string
		changed domain.Element
	}{
		{name: "type only", changed: domain.Element{ID: "GW", Name: "Review decision", Type: "parallelGateway", GatewayDirection: "Diverging", DefaultFlow: "FLOW-REJECT"}},
		{name: "direction only", changed: domain.Element{ID: "GW", Name: "Review decision", Type: "exclusiveGateway", GatewayDirection: "Converging", DefaultFlow: "FLOW-REJECT"}},
		{name: "default only", changed: domain.Element{ID: "GW", Name: "Review decision", Type: "exclusiveGateway", GatewayDirection: "Diverging"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := diff.Compare(oldGraph, graphWithElements(&test.changed))
			if len(got) != 1 || got[0].ChangeType != domain.ModifyGatewaySemantics {
				t.Fatalf("gateway semantic change missed: %#v", got)
			}
		})
	}
}

func TestGatewayBranchLabelDoesNotChangeExplicitCondition(t *testing.T) {
	elements := func() []*domain.Element {
		return []*domain.Element{
			{ID: "GW", Name: "Valid?", Type: "exclusiveGateway", ProcessID: "P"},
			{ID: "DONE", Name: "Done", Type: "endEvent", ProcessID: "P"},
		}
	}
	oldGraph := graphWithFlow("F", "GW", "DONE", "input valid", elements()...)
	newGraph := graphWithFlow("F", "GW", "DONE", "input valid", elements()...)
	oldGraph.Flows["F"].Name = "Ya"
	newGraph.Flows["F"].Name = "Data valid"
	changes := diff.Compare(oldGraph, newGraph)
	if len(changes) != 1 || changes[0].ChangeType != domain.ModifySequenceFlow {
		t.Fatalf("branch label alone changed gateway condition: %#v", changes)
	}
	if changes[0].Reason != "sequence flow label changed" {
		t.Fatalf("label-only flow not explained precisely: %q", changes[0].Reason)
	}

	newGraph.Flows["F"].Condition = "input valid and within budget"
	changes = diff.Compare(oldGraph, newGraph)
	if len(changes) != 2 || changes[0].ChangeType != domain.ModifySequenceFlow || changes[1].ChangeType != domain.ModifyGatewayCondition {
		t.Fatalf("explicit gateway condition change missed: %#v", changes)
	}
	if changes[0].Reason != "sequence flow condition, label changed" {
		t.Fatalf("condition and label edits not distinguished: %q", changes[0].Reason)
	}

	oldGraph.Flows["F"].Condition = ""
	newGraph.Flows["F"].Condition = ""
	changes = diff.Compare(oldGraph, newGraph)
	if len(changes) != 2 || changes[0].ChangeType != domain.ModifySequenceFlow || changes[1].ChangeType != domain.ModifyGatewayCondition {
		t.Fatalf("label-only gateway condition surrogate missed: %#v", changes)
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

func TestPlainInsertGroupsOnlyItsThreeFlowEdits(t *testing.T) {
	changes := diff.Compare(parseFixture(t, "insert-task-old.bpmn"), parseFixture(t, "insert-task-new.bpmn"))
	if len(changes) != 4 {
		t.Fatalf("expected task plus three raw flow changes, got %d", len(changes))
	}
	primary, details := 0, 0
	for _, change := range changes {
		if change.ChangeType == domain.AddTask {
			primary++
			if change.PartOf != "" {
				t.Fatalf("primary task grouped under another change: %#v", change)
			}
		}
		if change.ChangeType == domain.ModifySequenceFlow {
			if change.PartOf != "TASK-REVIEW" || change.Context != nil {
				t.Fatalf("plain inserted flow not grouped correctly: %#v", change)
			}
			details++
		}
	}
	if primary != 1 || details != 3 {
		t.Fatalf("primary=%d details=%d", primary, details)
	}

	oldGraph := graphWithFlow("F", "A", "B", "approved", &domain.Element{ID: "A", Name: "Prepare", Type: "task", ProcessID: "P"}, &domain.Element{ID: "B", Name: "Review", Type: "task", ProcessID: "P"})
	newGraph := graphWithFlow("F1", "A", "N", "approved", &domain.Element{ID: "A", Name: "Prepare", Type: "task", ProcessID: "P"}, &domain.Element{ID: "N", Name: "New step", Type: "task", ProcessID: "P"}, &domain.Element{ID: "B", Name: "Review", Type: "task", ProcessID: "P"})
	newGraph.Flows["F2"] = &domain.Flow{ID: "F2", SourceRef: "N", TargetRef: "B"}
	newGraph.Outgoing["N"] = []string{"F2"}
	newGraph.Incoming["B"] = []string{"F2"}
	for _, change := range diff.Compare(oldGraph, newGraph) {
		if change.ChangeType == domain.ModifySequenceFlow && change.PartOf != "" {
			t.Fatalf("conditional flow was grouped: %#v", change)
		}
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
