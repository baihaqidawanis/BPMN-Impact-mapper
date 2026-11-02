package diff_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestConditionDeltaUsesStableFlowWordsAndKeepsFullContext(t *testing.T) {
	old := contextGraph(t, `<exclusiveGateway id="G" name="Copy allowed?"/>`, `<sequenceFlow id="B" sourceRef="Before" targetRef="G"/><sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>status REJECTED</conditionExpression></sequenceFlow>`)
	next := contextGraph(t, `<exclusiveGateway id="G" name="Copy allowed?"/>`, `<sequenceFlow id="B" sourceRef="Before" targetRef="G"/><sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>status REJECTED CANCELLED</conditionExpression></sequenceFlow>`)
	changes := diff.Compare(old, next)
	var original *domain.ChangeContext
	for _, c := range changes {
		if c.ChangeType == domain.ModifyGatewayCondition {
			original = c.Context
		}
	}
	before, _ := json.Marshal(original)
	diff.AnnotateConditionDelta(old, next, changes)
	for _, c := range changes {
		if c.ChangeType != domain.ModifyGatewayCondition {
			continue
		}
		if !reflect.DeepEqual(c.Context.ConditionDelta, []string{"cancelled"}) {
			t.Fatalf("delta: %v", c.Context.ConditionDelta)
		}
		if !reflect.DeepEqual(c.Context.BranchTerms, original.BranchTerms) {
			t.Fatal("lost canonical branch conditions")
		}
	}
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("mutated shared original context")
	}
}

func TestConditionDeltaFallsBackForOperatorOnlyAndStructuralChanges(t *testing.T) {
	for _, altered := range []string{
		`<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>amount &gt;= 10</conditionExpression></sequenceFlow>`,
		`<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>amount &gt; 20</conditionExpression></sequenceFlow><sequenceFlow id="New" sourceRef="G" targetRef="Before" name="otherwise"/>`,
		`<sequenceFlow id="F" sourceRef="G" targetRef="Before"><conditionExpression>amount &gt; 20</conditionExpression></sequenceFlow>`,
	} {
		old := contextGraph(t, `<exclusiveGateway id="G"/>`, `<sequenceFlow id="F" sourceRef="G" targetRef="After"><conditionExpression>amount &gt; 10</conditionExpression></sequenceFlow>`)
		next := contextGraph(t, `<exclusiveGateway id="G"/>`, altered)
		changes := diff.Compare(old, next)
		diff.AnnotateConditionDelta(old, next, changes)
		found := false
		for _, c := range changes {
			if c.ChangeType == domain.ModifyGatewayCondition {
				found = true
				if len(c.Context.ConditionDelta) != 0 {
					t.Fatalf("unsafe focus: %v", c.Context.ConditionDelta)
				}
			}
		}
		if !found {
			t.Fatal("fixture produced no condition change")
		}
	}
}

func TestConditionDeltaFallsBackForBooleanOnlyRewrite(t *testing.T) {
	old := contextGraph(t, `<exclusiveGateway id="G"/>`, `<sequenceFlow id="F" sourceRef="G" targetRef="After" name="approved and active"/>`)
	next := contextGraph(t, `<exclusiveGateway id="G"/>`, `<sequenceFlow id="F" sourceRef="G" targetRef="After" name="approved or active"/>`)
	changes := diff.Compare(old, next)
	diff.AnnotateConditionDelta(old, next, changes)
	found := false
	for _, c := range changes {
		if c.ChangeType == domain.ModifyGatewayCondition {
			found = true
			if len(c.Context.ConditionDelta) != 0 {
				t.Fatal("discarded operands during boolean-only rewrite")
			}
		}
	}
	if !found {
		t.Fatal("missing boolean condition change")
	}
}
