package diff_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"testing"
)

func TestGatewayContextKeepsOldAndNewConditionsDirected(t *testing.T) {
	old := contextGraph(t, `<exclusiveGateway id="G"/>`, `<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>state == REJECTED</conditionExpression></sequenceFlow>`)
	next := contextGraph(t, `<exclusiveGateway id="G"/>`, `<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>state == REJECTED || state == CANCELLED</conditionExpression></sequenceFlow>`)
	found := false
	for _, c := range diff.Compare(old, next) {
		if c.ChangeType == domain.ModifyGatewayCondition {
			found = true
			if len(c.Context.BranchChanges) != 1 {
				t.Fatal("missing branch direction")
			}
			b := c.Context.BranchChanges[0]
			if b.FlowID != "F" || b.OldCondition != "state == REJECTED" || b.NewCondition != "state == REJECTED || state == CANCELLED" {
				t.Fatalf("wrong requirement: %+v", b)
			}
		}
	}
	if !found {
		t.Fatal("fixture did not produce gateway condition change")
	}
}

func TestGatewayRerouteKeepsUnchangedGuardAndBothTargets(t *testing.T) {
	nodes := `<exclusiveGateway id="G" name="Eligible?"/><serviceTask id="Alternative" name="Request review"/>`
	before := contextGraph(t, nodes, `<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>eligible</conditionExpression></sequenceFlow>`)
	after := contextGraph(t, nodes, `<sequenceFlow id="F" sourceRef="G" targetRef="Alternative"><conditionExpression>eligible</conditionExpression></sequenceFlow>`)
	changes := diff.Compare(before, after)
	if len(changes) != 1 || changes[0].ChangeType != domain.ModifySequenceFlow {
		t.Fatalf("reroute should stay a single flow change: %+v", changes)
	}
	context := changes[0].Context
	if context == nil || len(context.BranchChanges) != 1 || len(context.Successors) != 2 {
		t.Fatalf("missing directed branch context: %+v", context)
	}
	branch := context.BranchChanges[0]
	if branch.OldCondition != "eligible" || branch.NewCondition != "eligible" || branch.OldTarget != "After" || branch.NewTarget != "Alternative" ||
		context.Successors[0].Name != "Validate claim DRAFT version" || context.Successors[1].Name != "Request review" {
		t.Fatalf("reroute lost a condition or destination: %+v", context)
	}
}
