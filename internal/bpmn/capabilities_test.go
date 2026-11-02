package bpmn_test

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"strings"
	"testing"
)

func TestAdvancedFeaturesAreExplicitlyPartial(t *testing.T) {
	input := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><userTask id="A"><multiInstanceLoopCharacteristics isSequential="true"/></userTask><serviceTask id="B"><standardLoopCharacteristics/></serviceTask><boundaryEvent id="Timeout" attachedToRef="A" cancelActivity="false"><timerEventDefinition><timeDuration>PT1H</timeDuration></timerEventDefinition></boundaryEvent><subProcess id="Events" triggeredByEvent="true"><startEvent id="Message"><messageEventDefinition messageRef="M"/></startEvent></subProcess><eventBasedGateway id="Race"/><callActivity id="Call" calledElement="OtherProcess"/></process><process id="Q"><task id="Remote"/></process><collaboration id="C"><participant id="Pool1" processRef="P"/><participant id="Pool2" processRef="Q"/><messageFlow id="Communication" sourceRef="A" targetRef="Remote"/></collaboration></definitions>`
	g, err := bpmn.Parse([]byte(input), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(g.Warnings, "\n")
	for _, feature := range []string{"multiInstanceLoopCharacteristics", "standardLoopCharacteristics", "boundaryEvent", "timerEventDefinition", "messageEventDefinition", "event-subprocess", "eventBasedGateway", "callActivity", "messageFlow", "Multiple processes"} {
		if !strings.Contains(joined, feature) {
			t.Fatal("silently omitted", feature, g.Warnings)
		}
	}
	if g.Elements["Timeout"].AttachedToRef != "A" || g.Elements["Message"].ParentID != "Events" || g.Elements["Remote"].ProcessID != "Q" {
		t.Fatal("partial support must still preserve available context")
	}
	before, _ := json.Marshal(g.Warnings)
	for i := 0; i < 5; i++ {
		next, err := bpmn.Parse([]byte(input), bpmn.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(next.Warnings)
		if string(before) != string(data) {
			t.Fatal("warnings changed order")
		}
	}
	found := false
	for _, f := range bpmn.Check(g, bpmn.CheckOptions{}) {
		if f.Code == "BPMN_PARTIAL_SUPPORT" {
			found = true
		}
	}
	if !found {
		t.Fatal("checker hid partial support")
	}
}

func TestPlainGatewayAndTaskTypesDoNotClaimPartialSupport(t *testing.T) {
	input := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><startEvent id="S"/><userTask id="A"/><serviceTask id="B"/><sendTask id="C"/><exclusiveGateway id="X"/><parallelGateway id="And"/><inclusiveGateway id="Or"/><endEvent id="E"/></process></definitions>`
	g, err := bpmn.Parse([]byte(input), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Warnings) != 0 {
		t.Fatal(g.Warnings)
	}
}

func TestCompensationFlagIsNotSilentlyIgnored(t *testing.T) {
	input := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><serviceTask id="Undo" isForCompensation="true"/></process></definitions>`
	g, err := bpmn.Parse([]byte(input), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Warnings) != 1 || !strings.Contains(g.Warnings[0], "compensation activity") {
		t.Fatal(g.Warnings)
	}
}
