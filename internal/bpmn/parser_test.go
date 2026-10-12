package bpmn_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

func TestParseSyntheticFixture(t *testing.T) {
	data := fixture(t, "new.bpmn")
	graph, err := bpmn.Parse(data, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if graph.Elements["TASK-CREATE"].Actor != "Coordinator" {
		t.Fatalf("actor=%q", graph.Elements["TASK-CREATE"].Actor)
	}
	if got := graph.Flows["FLOW-3"].Condition; got != "${score >= 70}" {
		t.Fatalf("condition=%q", got)
	}
	if got := graph.Elements["DATA-ORDER"].State; got != "REVIEWED" {
		t.Fatalf("state=%q", got)
	}
	if len(graph.Elements["TASK-CREATE"].DataRequirements) != 2 {
		t.Fatalf("requirements=%v", graph.Elements["TASK-CREATE"].DataRequirements)
	}
}

func TestParseRejectsDirectiveAndLimits(t *testing.T) {
	withDirective := []byte(`<!DOCTYPE definitions><definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="p"/></definitions>`)
	if _, err := bpmn.Parse(withDirective, bpmn.Limits{}); err == nil {
		t.Fatal("expected directive rejection")
	}
	if _, err := bpmn.Parse(fixture(t, "old.bpmn"), bpmn.Limits{MaxBytes: 10}); err == nil {
		t.Fatal("expected byte limit rejection")
	}
}

func TestParsePreservesProcessSubprocessAndEventContext(t *testing.T) {
	data := []byte(`<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
		<process id="PROCESS-1">
			<subProcess id="SUB-1" name="Review">
				<userTask id="TASK-1" name="Check" default="FLOW-DEFAULT"/>
				<boundaryEvent id="EVENT-TIMER" attachedToRef="TASK-1"><timerEventDefinition/></boundaryEvent>
			</subProcess>
			<exclusiveGateway id="GW-1" gatewayDirection="Diverging" default="FLOW-DEFAULT"/>
		</process>
	</definitions>`)
	graph, err := bpmn.Parse(data, bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	task := graph.Elements["TASK-1"]
	if task.ProcessID != "PROCESS-1" || task.ParentID != "SUB-1" || task.DefaultFlow != "FLOW-DEFAULT" {
		t.Fatalf("task context=%+v", task)
	}
	event := graph.Elements["EVENT-TIMER"]
	if event.ParentID != "SUB-1" || event.AttachedToRef != "TASK-1" || len(event.EventDefinitions) != 1 || event.EventDefinitions[0] != "timerEventDefinition" {
		t.Fatalf("event context=%+v", event)
	}
	if got := graph.Elements["GW-1"].GatewayDirection; got != "Diverging" {
		t.Fatalf("gateway direction=%q", got)
	}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bpmn", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
