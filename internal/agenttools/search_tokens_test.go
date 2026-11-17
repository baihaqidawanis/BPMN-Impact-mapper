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

func TestSourceSearchUsesWordsRatherThanUnrelatedSubstrings(t *testing.T) {
	root := t.TempDir()
	for name, code := range map[string]string{
		"commands.go": "package fixture\nfunc Command(){}\n", // 'and' is only a substring
		"logic.go":    "package fixture\n// and\nfunc Handle(){}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Execute(context.Background(), planning.ToolRequest{Name: "search_source", Query: "and"})
	if err != nil || len(r.Candidates) != 1 || r.Candidates[0].File != "logic.go" {
		t.Fatal("unrelated substring became evidence", r, err)
	}
}
