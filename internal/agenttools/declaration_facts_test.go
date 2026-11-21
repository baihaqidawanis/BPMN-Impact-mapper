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

func TestDeclarationFactsPreserveGroupedArgumentOrderAndActualFields(t *testing.T) {
	root := t.TempDir()
	code := `package example
type Document struct { ID, Owner string; Status string; Version int }
func (d *Document) Submit() error { return nil }
func CreateEntry(id, warehouse, product string, quantity int, kind, source string) {}
`
	os.WriteFile(filepath.Join(root, "flow.go"), []byte(code), 0600)
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ symbol, want string }{{"Document.Submit", "parameters in order: none. Receiver declared fields: ID, Owner, Status, Version."}, {"CreateEntry", "parameters in order: id, warehouse, product, quantity, kind, source."}} {
		r, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: "flow.go", Symbol: tc.symbol})
		if err != nil || len(r.Candidates) != 1 {
			t.Fatal(r, err)
		}
		found := false
		raw := false
		for _, e := range r.Candidates[0].Evidence {
			if len([]rune(e.Excerpt)) > 240 {
				t.Fatal("unbounded fact")
			}
			if e.Type == "AST_DECLARATION_FACT" {
				found = strings.Contains(e.Excerpt, tc.want)
				if strings.Contains(e.Excerpt, "Reason") {
					t.Fatal("invented field")
				}
			} else {
				raw = true
			}
		}
		if !found || !raw {
			t.Fatal("missing declaration or raw source", r)
		}
	}
	// Tools remain bound to the ingestion snapshot even after local files change.
	os.WriteFile(filepath.Join(root, "flow.go"), []byte("package replaced"), 0600)
	r, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: "flow.go", Symbol: "CreateEntry"})
	if err != nil || !strings.Contains(r.Candidates[0].Evidence[0].Excerpt, "kind, source") {
		t.Fatal("snapshot drift", r, err)
	}
}
