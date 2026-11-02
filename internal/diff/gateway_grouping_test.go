package diff_test

import (
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func gatewayFixture(t *testing.T, branches string) *domain.Graph {
	t.Helper()
	xml := `<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><task id="S" name="Submit claim"/><exclusiveGateway id="G" name="Eligible?"/><task id="A" name="Pay claim"/><task id="B" name="Reject claim"/><sequenceFlow id="IN" sourceRef="S" targetRef="G"/>` + branches + `</process></definitions>`
	g, err := bpmn.Parse([]byte(xml), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGatewayConditionGroupingPreservesIndependentChanges(t *testing.T) {
	old := `<sequenceFlow id="F" sourceRef="G" targetRef="A" name="yes"><conditionExpression>approved</conditionExpression></sequenceFlow><sequenceFlow id="R" sourceRef="G" targetRef="B" name="rejected"/>`
	for _, tc := range []struct {
		name, next string
		grouped    bool
	}{
		{"condition", strings.Replace(old, ">approved<", ">budget available<", 1), true},
		{"label only", strings.Replace(old, `name="yes"`, `name="Accepted claim"`, 1), false},
		{"rewire and condition", strings.Replace(strings.Replace(old, `targetRef="A"`, `targetRef="B"`, 1), ">approved<", ">budget available<", 1), false},
		{"remove", strings.Split(old, `<sequenceFlow id="R"`)[0], false},
		{"add", old + `<sequenceFlow id="X" sourceRef="G" targetRef="A" name="expedited"/>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes := diff.Compare(gatewayFixture(t, old), gatewayFixture(t, tc.next))
			seen := false
			for _, c := range changes {
				if c.ChangeType == domain.ModifySequenceFlow {
					seen = true
					if (c.PartOf == "G") != tc.grouped {
						t.Fatalf("unexpected grouping: %#v", c)
					}
				}
				if tc.name == "remove" && c.ChangeType == domain.ModifyGatewayCondition && !strings.Contains(strings.Join(c.Context.BranchTerms, " "), "rejected") {
					t.Fatal("removed branch condition lost")
				}
			}
			if !seen {
				t.Fatal("atomic flow diff missing")
			}
		})
	}
}

func TestGatewayLabelSurrogateGroupsButOtherBranchDisplayEditDoesNot(t *testing.T) {
	old := `<sequenceFlow id="F" sourceRef="G" targetRef="A" name="approved"/><sequenceFlow id="R" sourceRef="G" targetRef="B" name="no"><conditionExpression>rejected</conditionExpression></sequenceFlow>`
	next := strings.Replace(strings.Replace(old, `name="approved"`, `name="budget available"`, 1), `name="no"`, `name="Claim rejected"`, 1)
	for _, c := range diff.Compare(gatewayFixture(t, old), gatewayFixture(t, next)) {
		if c.ChangeType != domain.ModifySequenceFlow {
			continue
		}
		if (c.PartOf != "") != (c.NewElement.ID == "F") {
			t.Fatalf("display edit incorrectly swallowed: %#v", c)
		}
	}
}

func TestAddedAndRemovedGatewayKeepDirectBranchConditions(t *testing.T) {
	withGateway := gatewayFixture(t, `<sequenceFlow id="F" sourceRef="G" targetRef="A" name="yes"><conditionExpression>budget available</conditionExpression></sequenceFlow><sequenceFlow id="R" sourceRef="G" targetRef="B" name="claim rejected"/>`)
	withoutGateway, err := bpmn.Parse([]byte(`<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><task id="S" name="Submit claim"/><task id="A" name="Pay claim"/><task id="B" name="Reject claim"/><sequenceFlow id="F" sourceRef="S" targetRef="A"/><sequenceFlow id="R" sourceRef="S" targetRef="B"/></process></definitions>`), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		old, next *domain.Graph
		kind      domain.ChangeType
	}{
		{withoutGateway, withGateway, domain.AddGateway},
		{withGateway, withoutGateway, domain.RemoveGateway},
	} {
		found := false
		for _, c := range diff.Compare(tc.old, tc.next) {
			if c.ChangeType != tc.kind {
				continue
			}
			found = true
			if c.Context == nil || strings.Join(c.Context.BranchTerms, "|") != "budget available|claim rejected|yes" {
				t.Fatalf("%s lost direct conditions: %#v", tc.kind, c)
			}
		}
		if !found {
			t.Fatalf("missing %s", tc.kind)
		}
	}
}
