package bpmn_test

import (
	"encoding/json"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
)

func TestCheckDistinguishesConventionFromBPMNStructure(t *testing.T) {
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><startEvent id="S"/><exclusiveGateway id="X"/><endEvent id="E1"/><endEvent id="E2"/><sequenceFlow id="F1" sourceRef="S" targetRef="X"/><sequenceFlow id="F2" name="accepted" sourceRef="X" targetRef="E1"/><sequenceFlow id="F3" name="rejected" sourceRef="X" targetRef="E2"/></process></definitions>`
	g, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(g)
	if f := bpmn.Check(g, bpmn.CheckOptions{}); len(f) != 0 {
		t.Fatalf("valid multiple ends should not fail: %#v", f)
	}
	f := bpmn.Check(g, bpmn.CheckOptions{ResearchConvention: true})
	if len(f) != 1 || f[0].Code != "SINGLE_END" || f[0].Category != "convention" {
		t.Fatal(f)
	}
	after, _ := json.Marshal(g)
	if string(before) != string(after) {
		t.Fatal("checker mutated graph")
	}
}

func TestCheckScopesAndJoinActivity(t *testing.T) {
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><startEvent id="S"/><subProcess id="Sub"><startEvent id="SS"/><task id="T"/><endEvent id="SE"/><sequenceFlow id="a" sourceRef="SS" targetRef="T"/><sequenceFlow id="b" sourceRef="T" targetRef="SE"/></subProcess><parallelGateway id="Join"/><endEvent id="E"/><sequenceFlow id="c" sourceRef="S" targetRef="Join"/><sequenceFlow id="d" sourceRef="Sub" targetRef="Join"/><sequenceFlow id="e" sourceRef="Join" targetRef="E"/></process></definitions>`
	g, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	f := bpmn.Check(g, bpmn.CheckOptions{ResearchConvention: true})
	if len(f) != 1 || f[0].Code != "ACTIVITY_AFTER_JOIN" || f[0].Scope != "P" {
		t.Fatalf("scope counts or join rule incorrect: %#v", f)
	}
}

func TestCheckLocalGatewayWarningsAreDeterministic(t *testing.T) {
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><exclusiveGateway id="X" default="missing"/><parallelGateway id="J"/><task id="T"/><sequenceFlow id="a" sourceRef="X" targetRef="J"/><sequenceFlow id="b" sourceRef="X" targetRef="J"/><sequenceFlow id="c" sourceRef="J" targetRef="T"><conditionExpression>ready</conditionExpression></sequenceFlow></process></definitions>`
	g, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	f := bpmn.Check(g, bpmn.CheckOptions{})
	codes := map[string]int{}
	for _, item := range f {
		codes[item.Code]++
	}
	for _, code := range []string{"INVALID_GATEWAY_DEFAULT", "AND_CONDITION_IGNORED", "REVIEW_XOR_TO_AND_JOIN", "BRANCH_RULE_UNSPECIFIED"} {
		if codes[code] == 0 {
			t.Fatal("missing", code)
		}
	}
	a, _ := json.Marshal(f)
	for i := 0; i < 20; i++ {
		b, _ := json.Marshal(bpmn.Check(g, bpmn.CheckOptions{}))
		if string(a) != string(b) {
			t.Fatal("unstable findings")
		}
	}
}
