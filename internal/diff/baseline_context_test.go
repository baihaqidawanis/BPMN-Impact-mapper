package diff_test

import (
	"encoding/json"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strings"
	"testing"
)

func contextGraph(t *testing.T, nodes, flows string) *domain.Graph {
	t.Helper()
	g, err := bpmn.Parse([]byte(`<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="P"><userTask id="Before" name="Submit claim"/><serviceTask id="After" name="Validate claim DRAFT version"/>`+nodes+flows+`</process></definitions>`), bpmn.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return g
}
func TestBaselineExpansionPreservesNeighboursAndStopsCycles(t *testing.T) {
	old := contextGraph(t, "", `<sequenceFlow id="Bridge" sourceRef="Before" targetRef="After"/>`)
	next := contextGraph(t, `<serviceTask id="N1" name="New check"/><parallelGateway id="N2" name="New join"/>`, `<sequenceFlow id="F1" sourceRef="Before" targetRef="N1"/><sequenceFlow id="F2" sourceRef="N1" targetRef="N2"/><sequenceFlow id="F3" sourceRef="N2" targetRef="After"/><sequenceFlow id="Back" sourceRef="N2" targetRef="N1"/>`)
	changes := diff.Compare(old, next)
	var c *domain.Change
	for i := range changes {
		if changes[i].NewElement != nil && changes[i].NewElement.ID == "N1" {
			c = &changes[i]
			break
		}
	}
	if c == nil {
		t.Fatal("missing test change")
	}
	before, _ := json.Marshal(c.Context)
	originalContext := c.Context
	diff.ExpandBaselineContext(old, next, changes, 4)
	if len(c.Context.BaselineAnchors) != 2 {
		t.Fatal(c.Context)
	}
	original, _ := json.Marshal(originalContext)
	if string(before) != string(original) {
		t.Fatal("mutated original context")
	}
	for _, a := range c.Context.BaselineAnchors {
		if a.Hops != len(a.FlowPath) || a.Hops > 4 {
			t.Fatal(a)
		}
		if a.ID == "After" && (a.Hops != 2 || a.Direction != "successor") {
			t.Fatal(a)
		}
	}
	a, _ := json.Marshal(changes)
	again := diff.Compare(old, next)
	diff.ExpandBaselineContext(old, next, again, 4)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Fatal("expansion nondeterministic")
	}
}
func TestBaselineExpansionHopsAndScopeBoundaries(t *testing.T) {
	old := contextGraph(t, `<subProcess id="Sub"><serviceTask id="Inner" name="Private validation"/></subProcess>`, "")
	nodes := `<subProcess id="Sub"><serviceTask id="Inner" name="Private validation"/></subProcess>`
	flows := `<sequenceFlow id="Start" sourceRef="Before" targetRef="N1"/><sequenceFlow id="Cross" sourceRef="N1" targetRef="Inner"/>`
	for i := 1; i <= 5; i++ {
		nodes += fmt.Sprintf(`<serviceTask id="N%d" name="new check %d"/>`, i, i)
		if i < 5 {
			flows += fmt.Sprintf(`<sequenceFlow id="F%d" sourceRef="N%d" targetRef="N%d"/>`, i, i, i+1)
		}
	}
	flows += `<sequenceFlow id="Finish" sourceRef="N5" targetRef="After"/>`
	next := contextGraph(t, nodes, flows)
	changes := diff.Compare(old, next)
	diff.ExpandBaselineContext(old, next, changes, 4)
	for _, c := range changes {
		if c.NewElement != nil && c.NewElement.ID == "N1" {
			if c.Context == nil {
				t.Fatal("missing context")
			}
			for _, a := range c.Context.BaselineAnchors {
				if a.ID == "Inner" || a.ID == "After" {
					t.Fatal("escaped scope or hop limit", a)
				}
			}
		}
	}
}
func TestSingleInsertionDoesNotExpandAndCapsWideGraphs(t *testing.T) {
	old := contextGraph(t, "", "")
	next := contextGraph(t, `<task id="N" name="new"/>`, `<sequenceFlow id="A" sourceRef="Before" targetRef="N"/><sequenceFlow id="B" sourceRef="N" targetRef="After"/>`)
	changes := diff.Compare(old, next)
	diff.ExpandBaselineContext(old, next, changes, 4)
	for _, c := range changes {
		if c.Context != nil && len(c.Context.BaselineAnchors) > 0 {
			t.Fatal("single insertion changed")
		}
	}
	// A wide fan-out must remain bounded and return at most two anchors/direction.
	var nodes, flows strings.Builder
	nodes.WriteString(`<parallelGateway id="N"/><task id="New"/>`)
	flows.WriteString(`<sequenceFlow id="Cluster" sourceRef="New" targetRef="N"/>`)
	for i := 0; i < 200; i++ {
		id := fmt.Sprintf("Old%03d", i)
		nodes.WriteString(fmt.Sprintf(`<task id="%s" name="existing %d"/>`, id, i))
		flows.WriteString(fmt.Sprintf(`<sequenceFlow id="Wide%03d" sourceRef="N" targetRef="%s"/>`, i, id))
	}
	oldNodes := strings.ReplaceAll(nodes.String(), `<parallelGateway id="N"/><task id="New"/>`, "")
	old = contextGraph(t, oldNodes, "")
	next = contextGraph(t, nodes.String(), flows.String())
	changes = diff.Compare(old, next)
	diff.ExpandBaselineContext(old, next, changes, 4)
	for _, c := range changes {
		if c.Context != nil && len(c.Context.BaselineAnchors) > 4 {
			t.Fatal("unbounded anchor output")
		}
	}
}
