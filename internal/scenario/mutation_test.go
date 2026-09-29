package scenario

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

const sampleBPMN = `<?xml version="1.0"?><definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><laneSet><lane id="L" name="Operator"><flowNodeRef>T1</flowNodeRef></lane></laneSet><startEvent id="S"/><task id="T1" name="Old"/><endEvent id="E"/><sequenceFlow id="F1" sourceRef="S" targetRef="T1"/><sequenceFlow id="F2" sourceRef="T1" targetRef="E"><conditionExpression>old = true</conditionExpression></sequenceFlow></process></definitions>`

func TestMutateAppliesOperationsAndProducesValidBPMN(t *testing.T) {
	sum := sha256.Sum256([]byte(sampleBPMN))
	spec := Spec{Version: Version, ScenarioID: "case-1", BaselineSHA256: fmt.Sprintf("%x", sum), Operations: []Operation{
		{Kind: "set_attribute", ElementID: "T1", Attribute: "name", Value: "New"},
		{Kind: "set_child_text", ElementID: "F2", Child: "conditionExpression", Value: "new < 3"},
	}}
	result, err := Mutate([]byte(sampleBPMN), spec)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := bpmn.Parse(result, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if graph.Elements["T1"].Name != "New" || graph.Flows["F2"].Condition != "new < 3" {
		t.Fatalf("unexpected mutation result: %#v %#v", graph.Elements["T1"], graph.Flows["F2"])
	}
	if !strings.HasPrefix(string(result), `<?xml version="1.0"?><definitions`) || !strings.Contains(string(result), `<task id="T1" name="New"/>`) {
		t.Fatalf("unrelated XML representation changed: %s", result)
	}
}

func TestMutateRemovesAndRewires(t *testing.T) {
	sum := sha256.Sum256([]byte(sampleBPMN))
	spec := Spec{Version: Version, ScenarioID: "case-2", BaselineSHA256: fmt.Sprintf("%x", sum), Operations: []Operation{
		{Kind: "remove_element", ElementID: "T1"},
		{Kind: "remove_element", ElementID: "F2"},
		{Kind: "set_attribute", ElementID: "F1", Attribute: "targetRef", Value: "E"},
	}}
	result, err := Mutate([]byte(sampleBPMN), spec)
	if err != nil {
		t.Fatal(err)
	}
	graph, _ := bpmn.Parse(result, bpmn.Limits{})
	if graph.Elements["T1"] != nil || graph.Flows["F1"].TargetRef != "E" {
		t.Fatalf("remove/rewire did not apply: %#v", graph)
	}
}

func TestMutateFailsClosed(t *testing.T) {
	sum := sha256.Sum256([]byte(sampleBPMN))
	base := Spec{Version: Version, ScenarioID: "case", BaselineSHA256: fmt.Sprintf("%x", sum)}
	tests := []struct {
		name string
		spec Spec
		want string
	}{
		{"hash", Spec{Version: Version, ScenarioID: "case", BaselineSHA256: strings.Repeat("0", 64), Operations: []Operation{{Kind: "remove_element", ElementID: "T1"}}}, "mismatch"},
		{"missing target", withOperations(base, Operation{Kind: "remove_element", ElementID: "missing"}), "applied 0 times"},
		{"missing attribute", withOperations(base, Operation{Kind: "set_attribute", ElementID: "T1", Attribute: "missing", Value: "x"}), "has no attribute"},
		{"invalid result", withOperations(base, Operation{Kind: "remove_element", ElementID: "T1"}), "generated BPMN is invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Mutate([]byte(sampleBPMN), test.spec)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestDecodeSpecRejectsUnknownAndDuplicateOperations(t *testing.T) {
	_, err := DecodeSpec([]byte(`{"version":"1","scenario_id":"x","baseline_sha256":"` + strings.Repeat("0", 64) + `","operations":[],"unknown":true}`))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	spec := Spec{Version: Version, ScenarioID: "x", BaselineSHA256: strings.Repeat("0", 64), Operations: []Operation{
		{Kind: "remove_element", ElementID: "T1"}, {Kind: "remove_element", ElementID: "T1"},
	}}
	if err := spec.Validate(); err == nil {
		t.Fatal("expected duplicate operation error")
	}
}

func withOperations(spec Spec, operations ...Operation) Spec {
	spec.Operations = operations
	return spec
}
