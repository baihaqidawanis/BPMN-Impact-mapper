package diff_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmn"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/diff"
)

func TestWebScenarioCatalogChanges(t *testing.T) {
	root := filepath.Join("..", "..", "web", "public", "demo")
	raw, err := os.ReadFile(filepath.Join(root, "scenarios.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID       string   `json:"id"`
		Old      string   `json:"old"`
		New      string   `json:"new"`
		Required []string `json:"required_changes"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 8 {
		t.Fatal("missing scenarios")
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			old, err := os.ReadFile(filepath.Join(root, c.Old))
			if err != nil {
				t.Fatal(err)
			}
			next, err := os.ReadFile(filepath.Join(root, c.New))
			if err != nil {
				t.Fatal(err)
			}
			a, err := bpmn.Parse(old, bpmn.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			b, err := bpmn.Parse(next, bpmn.Limits{})
			if err != nil {
				t.Fatal(err)
			}
			changes := diff.Compare(a, b)
			if len(c.Required) == 0 && len(changes) != 0 {
				t.Fatalf("no-change control produced %d changes", len(changes))
			}
			found := map[string]bool{}
			for _, change := range changes {
				found[string(change.ChangeType)] = true
			}
			for _, required := range c.Required {
				if !found[required] {
					t.Errorf("missing %s; got %v", required, found)
				}
			}
		})
	}
}
