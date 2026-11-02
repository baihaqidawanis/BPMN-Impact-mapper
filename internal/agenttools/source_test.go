package agenttools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestReadOnlySourceTools(t *testing.T) {
	root := t.TempDir()
	source := []byte("package fixture\nfunc Submit() { validate() }\nfunc validate() {}\nfunc Handle() { Submit() }\ntype Store struct{}\nfunc (s Store) Save() {}\n")
	if err := os.WriteFile(filepath.Join(root, "flow.go"), source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=do-not-read"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []planning.ToolRequest{{Name: "search_source", Query: "Submit"}, {Name: "read_function", File: "flow.go", Symbol: "Submit"}, {Name: "find_callers", File: "flow.go", Symbol: "Submit"}} {
		result, err := s.Execute(context.Background(), q)
		if err != nil || len(result.Candidates) == 0 {
			t.Fatalf("%s: %v %+v", q.Name, err, result)
		}
		for _, c := range result.Candidates {
			if c.File != "flow.go" || c.Origin != "agent-source-tool" || c.RetrievalScore != nil || len(c.Evidence[0].Excerpt) > 240 {
				t.Fatalf("unsafe result: %+v", c)
			}
		}
	}
	for _, file := range []string{"../flow.go", "C:/flow.go", ".env"} {
		if _, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: file, Symbol: "Submit"}); err == nil {
			t.Fatal("unsafe/unapproved source accepted")
		}
	}
	if _, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: "flow.go", Symbol: "fixture"}); err == nil {
		t.Fatal("package misrepresented as function")
	}
	if r, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: "flow.go", Symbol: "Store.Save"}); err != nil || len(r.Candidates) != 1 || r.Candidates[0].Symbol != "Store.Save" {
		t.Fatalf("Go method unavailable: %+v %v", r, err)
	}
	// Immutable snapshot: later changes require a new tool identity/session.
	if err := os.WriteFile(filepath.Join(root, "flow.go"), []byte("package changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_function", File: "flow.go", Symbol: "Submit"}); err != nil {
		t.Fatal("snapshot changed live")
	}
	changed, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil || s.Identity() == changed.Identity() {
		t.Fatal("source identity did not change")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Execute(ctx, planning.ToolRequest{Name: "search_source", Query: "submit"}); err == nil {
		t.Fatal("cancellation ignored")
	}
}
