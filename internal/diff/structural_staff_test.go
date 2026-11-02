package diff_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestStructuralStaffPairsHaveExpectedChangesAndCompleteDiagrams(t *testing.T) {
	for _, item := range []struct {
		name string
		kind domain.ChangeType
		id   string
	}{
		{"remove-task", domain.RemoveTask, "role_staff_l3_adj_cancel_input"},
		{"add-xor", domain.AddGateway, "ResearchSplit"},
		{"remove-gateway-control", domain.RemoveGateway, "role_staff_l3_adj_submit_valid"},
		{"remove-gateway", domain.RemoveGateway, "role_staff_l3_adj_submit_valid"},
		{"move-role", domain.ModifyActorRole, "role_staff_l3_adj_submit_input"},
		{"add-and", domain.AddGateway, "ResearchSplit"},
		{"add-or", domain.AddGateway, "ResearchSplit"},
	} {
		t.Run(item.name, func(t *testing.T) {
			graphs := []*domain.Graph{}
			for _, file := range []string{"before.bpmn", "after.bpmn"} {
				data, err := os.ReadFile(filepath.Join("..", "..", "experiments", "simple-erp", "structural-staff", item.name, file))
				if err != nil {
					t.Fatal(err)
				}
				graph, err := bpmn.Parse(data, bpmn.Limits{})
				if err != nil {
					t.Fatal(err)
				}
				graphs = append(graphs, graph)
				var diagram struct {
					Diagrams []struct {
						Plane struct {
							Shapes []struct {
								Element string `xml:"bpmnElement,attr"`
							} `xml:"BPMNShape"`
							Edges []struct {
								Element string `xml:"bpmnElement,attr"`
								Points  []struct {
									X string `xml:"x,attr"`
									Y string `xml:"y,attr"`
								} `xml:"waypoint"`
							} `xml:"BPMNEdge"`
						} `xml:"BPMNPlane"`
					} `xml:"BPMNDiagram"`
				}
				if err := xml.Unmarshal(data, &diagram); err != nil {
					t.Fatal(err)
				}
				shapes, edges := map[string]bool{}, map[string]bool{}
				for _, d := range diagram.Diagrams {
					for _, s := range d.Plane.Shapes {
						shapes[s.Element] = true
					}
					for _, e := range d.Plane.Edges {
						if len(e.Points) < 2 {
							t.Fatal("missing flow waypoints", e.Element)
						}
						edges[e.Element] = true
					}
				}
				for id := range graph.Elements {
					if !shapes[id] {
						t.Fatal("missing visible shape", file, id)
					}
				}
				for id := range graph.Flows {
					if !edges[id] {
						t.Fatal("missing visible flow", file, id)
					}
				}
			}
			found := false
			for _, change := range diff.Compare(graphs[0], graphs[1]) {
				element := change.NewElement
				if element == nil {
					element = change.OldElement
				}
				if change.ChangeType == item.kind && element != nil && element.ID == item.id {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s for %s", item.kind, item.id)
			}
		})
	}
}
