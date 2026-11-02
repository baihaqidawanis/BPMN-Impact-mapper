package diff_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestCombinedXORANDTasksPreservesEveryPrimaryChange(t *testing.T) {
	graphs := []*domain.Graph{}
	for _, name := range []string{"before.bpmn", "after.bpmn"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "experiments", "simple-erp", "staff-single-end", name))
		if err != nil {
			t.Fatal(err)
		}
		g, err := bpmn.Parse(data, bpmn.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if findings := bpmn.Check(g, bpmn.CheckOptions{ResearchConvention: true}); len(findings) != 0 {
			t.Fatal(name, findings)
		}
		graphs = append(graphs, g)
	}
	changes := diff.Compare(graphs[0], graphs[1])
	found := map[string]domain.Change{}
	for _, c := range changes {
		if c.ChangeType == domain.UnresolvedMatch {
			t.Fatal("unexpected ambiguous match", c)
		}
		if c.NewElement != nil {
			found[string(c.ChangeType)+":"+c.NewElement.ID] = c
		}
	}
	for id, kind := range map[string]domain.ChangeType{
		"ReasonDecision": domain.AddGateway, "MasterChecksSplit": domain.AddGateway, "MasterChecksJoin": domain.AddGateway, "MasterDecision": domain.AddGateway,
		"RejectMissingReason": domain.AddTask, "CheckWarehouse": domain.AddTask, "CheckProducts": domain.AddTask, "CombineValidation": domain.AddTask, "RejectInvalidMaster": domain.AddTask,
	} {
		c, ok := found[string(kind)+":"+id]
		if !ok {
			t.Fatalf("missing %s %s", kind, id)
		}
		if c.PartOf != "" {
			t.Fatal("primary change swallowed as a supporting flow", c)
		}
	}
	first, _ := json.Marshal(changes)
	for i := 0; i < 10; i++ {
		next, _ := json.Marshal(diff.Compare(graphs[0], graphs[1]))
		if string(first) != string(next) {
			t.Fatal("non deterministic compound diff")
		}
	}
	if c := diff.Compare(graphs[1], graphs[1]); len(c) != 0 {
		t.Fatal("unchanged compound graph generated changes")
	}
	g := graphs[1]
	if len(g.Outgoing["MasterChecksSplit"]) != 2 || len(g.Incoming["MasterChecksJoin"]) != 2 {
		t.Fatal("lost parallel branch")
	}
	for _, reject := range []string{"RejectMissingReason", "RejectInvalidMaster"} {
		if reachable(g, reject, "role_staff_l3_adj_submit_effect0") {
			t.Fatal("rejection reaches success-side state update", reject)
		}
		if !reachable(g, reject, "ReturnResponse") {
			t.Fatal("rejection has no response path", reject)
		}
	}
}

func reachable(g *domain.Graph, from, to string) bool {
	queue := []string{from}
	seen := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == to {
			return true
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		for _, fid := range g.Outgoing[id] {
			queue = append(queue, g.Flows[fid].TargetRef)
		}
	}
	return false
}
