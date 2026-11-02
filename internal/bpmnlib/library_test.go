package bpmnlib

import (
	"strings"
	"testing"
)

func TestContentIdentityAndConventions(t *testing.T) {
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="p"><startEvent id="s"/><exclusiveGateway id="g1"/><exclusiveGateway id="g2"/><task id="t"/><endEvent id="e"/><sequenceFlow id="a" sourceRef="s" targetRef="g1"/><sequenceFlow id="b" sourceRef="g1" targetRef="g2"/><sequenceFlow id="c" sourceRef="g2" targetRef="t"/><sequenceFlow id="d" sourceRef="t" targetRef="e"/></process></definitions>`
	meta := Metadata{Repository: "Demo", Process: "Approval", Level: 2, Version: "1"}
	first, err := Prepare(meta, xml)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		next, err := Prepare(meta, xml)
		if err != nil || next.ID != first.ID {
			t.Fatal("identity not deterministic")
		}
	}
	found := false
	for _, f := range first.Findings {
		if f.Code == "ACTIVITY_AFTER_GATEWAY" && f.ElementID == "g1" {
			found = true
		}
	}
	if !found {
		t.Fatal("direct gateway pair not flagged")
	}
	meta.Version = "2"
	second, _ := Prepare(meta, xml)
	if second.ID == first.ID {
		t.Fatal("metadata version must distinguish entries")
	}
	if _, err = Prepare(meta, strings.Repeat("x", 8<<20+1)); err == nil {
		t.Fatal("oversized XML accepted")
	}
}
