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

func TestInspectSourceFindsDistantAndCompressedBehavior(t *testing.T) {
	root := t.TempDir()
	code := "const empty = true;\n" + strings.Repeat("// unrelated boilerplate\n", 40) + "function action() { return send({reason: reason}); }\n" + strings.Repeat("x", 650) + "<input value={reason} onChange={setReason} />\n"
	if err := os.WriteFile(filepath.Join(root, "view.tsx"), []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	query := planning.ToolRequest{Name: "inspect_source", File: "view.tsx", Query: "Enter and validate reason before submit"}
	result, err := s.Execute(context.Background(), query)
	if err != nil || len(result.Candidates) != 1 {
		t.Fatal(result, err)
	}
	text := ""
	for _, e := range result.Candidates[0].Evidence {
		text += e.Excerpt
		if len([]rune(e.Excerpt)) > 240 {
			t.Fatal("unbounded evidence")
		}
	}
	if !strings.Contains(text, "send({reason") || !strings.Contains(text, "value={reason}") {
		t.Fatal("relevant spans absent", text)
	}
	again, _ := s.Execute(context.Background(), query)
	if len(again.Candidates[0].Evidence) != len(result.Candidates[0].Evidence) {
		t.Fatal("non deterministic")
	}
	for i, e := range result.Candidates[0].Evidence {
		if e != again.Candidates[0].Evidence[i] {
			t.Fatal("unstable evidence")
		}
	}
	for _, file := range []string{"../view.tsx", ".env", "C:/view.tsx"} {
		query.File = file
		if _, err := s.Execute(context.Background(), query); err == nil {
			t.Fatal("unsafe source accepted")
		}
	}
	query.File = "view.tsx"
	query.Query = "unmatchedword"
	fallback, err := s.Execute(context.Background(), query)
	if err != nil || fallback.Error != "" || fallback.Note == "" || len(fallback.Candidates) == 0 {
		t.Fatal("fallback must be readable with an explicit limitation", fallback, err)
	}
}

func TestInspectSourcePrioritizesRequestedDispatchBlock(t *testing.T) {
	root := t.TempDir()
	code := `package flow
func command(action, reason string) {
 switch action {
 case "archive":
  if reason == "" { return }
 case "publish":
  state := "DRAFT"
  if state != "DRAFT" { return }
  state = "PUBLISHED"
 case "erase":
  if reason == "" { return }
 }
}
`
	if err := os.WriteFile(filepath.Join(root, "flow.go"), []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "inspect_source", File: "flow.go", Query: "validate reason before publish"})
	if err != nil || len(result.Candidates) != 1 || !strings.Contains(result.Note, "Action-label") {
		t.Fatal(result, err)
	}
	evidence := result.Candidates[0].Evidence
	if !strings.Contains(evidence[0].Excerpt, `case "publish"`) {
		t.Fatal("wrong operation anchor", evidence)
	}
	found := false
	for _, e := range evidence {
		if strings.Contains(e.Excerpt, `state = "PUBLISHED"`) {
			found = true
		}
	}
	if !found {
		t.Fatal("mutation context missing", evidence)
	}
	search, err := snapshot.Execute(context.Background(), planning.ToolRequest{Name: "search_source", Query: "publish"})
	if err != nil || len(search.Candidates) == 0 {
		t.Fatal(search, err)
	}
	contract := false
	for _, e := range search.Candidates[0].Evidence {
		if e.Type == "SOURCE_DECLARATION" && strings.Contains(e.Excerpt, "action, reason string") && e.Location == "flow.go:2" {
			contract = true
		}
	}
	if !contract {
		t.Fatal("discovery lost function parameter contract", search)
	}
}
