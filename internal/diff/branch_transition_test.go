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
