package agenttools_test

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchWindowReportsActualFunctionOwner(t *testing.T) {
	root := t.TempDir()
	source := `package flow
type Store struct{}
func (s Store) Apply(action string) {
 switch action {
 case "publish":
  return
 }
}
func (s Store) Undo() {}
`
	file := filepath.Join(root, "flow.go")
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "inspect_source", File: "flow.go", Query: "publish"})
	if err != nil || !strings.Contains(result.Note, "Store.Apply lines 3-8") {
		t.Fatal("missing actual owner", result.Note, err)
	}
	// The read window can also include another function. Label it explicitly,
	// rather than associating every excerpt with the initial search hit.
	if !strings.Contains(result.Note, "Store.Undo lines 9-9") {
		t.Fatal("second boundary lost", result.Note)
	}
	if result.Candidates[0].Symbol != "" {
		t.Fatal("file target silently promoted")
	}
	hasScope := false
	for _, e := range result.Candidates[0].Evidence {
		if e.Type == "AST_SOURCE_SCOPE" {
			hasScope = true
		}
	}
	if !hasScope {
		t.Fatal("scope not retained as model evidence")
	}
	actual, _ := os.ReadFile(file)
	if string(actual) != source {
		t.Fatal("source changed")
	}
}

func TestDispatchPrefersOwnerNamedByProcess(t *testing.T) {
	root := t.TempDir()
	source := "package workflow\nfunc submitOrder(action string) {\nswitch action {\ncase \"submit\":\nreturn\n}\n}\n" + strings.Repeat("\n", 40) + "func submitAdjustment(action string) {\nswitch action {\ncase \"submit\":\nreturn\n}\n}\n"
	if err := os.WriteFile(filepath.Join(root, "workflow.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "inspect_source", File: "workflow.go", Query: "submit stock adjustment"})
	if err != nil || !strings.Contains(result.Note, "submitAdjustment") || strings.Contains(result.Note, "submitOrder") {
		t.Fatal(result.Note, err)
	}
}
