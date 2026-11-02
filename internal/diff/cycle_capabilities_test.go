package diff_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strings"
	"testing"
)

func TestCyclicSequenceFlowDiffAndTypedConditionText(t *testing.T) {
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><userTask id="A" name="Review claim"/><exclusiveGateway id="G"/><task id="B" name="Rework claim"/><endEvent id="E"/><sequenceFlow id="Input" sourceRef="A" targetRef="G"/><sequenceFlow id="Retry" sourceRef="G" targetRef="B"><conditionExpression>{"left":{"ref":"decision.outcome"},"op":"eq","right":{"literal":"REJECTED"}}</conditionExpression></sequenceFlow><sequenceFlow id="Back" sourceRef="B" targetRef="A"/><sequenceFlow id="Finish" sourceRef="G" targetRef="E" name="accepted"/></process></definitions>`
	old, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	next, err := bpmn.Parse([]byte(strings.Replace(xml, "REJECTED", "REVISION_REQUIRED", 1)), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	changes := diff.Compare(old, next)
	found := false
	for _, c := range changes {
		if c.ChangeType == domain.ModifyGatewayCondition {
			found = true
		}
	}
	if !found {
		t.Fatal("typed JSON condition text change missed in cyclic graph", changes)
	}
	if len(diff.Compare(old, old)) != 0 {
		t.Fatal("unchanged cycle generated changes")
	}
	if next.Flows["Back"].TargetRef != "A" {
		t.Fatal("back edge lost")
	}
}
