package agenttools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestSourceDiscoveryDoesNotStarveImplementationWithDocuments(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"docs/a.md":   "submit validate reason workflow change policyexclusive",
		"docs/b.md":   "submit validate reason workflow change",
		"docs/c.md":   "submit validate reason workflow change",
		"commands.go": "package fixture\nfunc Submit(reason string) {}\n",
		"panel.ts":    "export function submit(reason: string) { return reason; }\n",
	}
	for name, text := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	q := planning.ToolRequest{Name: "search_source", Query: "submit validate reason workflow change"}
	first, err := s.Execute(context.Background(), q)
	if err != nil || len(first.Candidates) != 3 || first.NextOffset != 3 || !first.Truncated {
		t.Fatal(first, err)
	}
	if first.Candidates[0].File != "commands.go" || first.Candidates[1].File != "panel.ts" {
		t.Fatal("documentation starved code", first.Candidates)
	}
	seen := map[string]bool{}
	for _, c := range first.Candidates {
		seen[c.File] = true
	}
	q.Offset = first.NextOffset
	second, err := s.Execute(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range second.Candidates {
		if seen[c.File] {
			t.Fatal("repeated page", c.File)
		}
		seen[c.File] = true
	}
	if len(seen) != len(files) || second.Truncated || second.NextOffset != 0 {
		t.Fatal("documents disappeared from pagination", seen, second)
	}
	q.Query, q.Offset = "policyexclusive", 0
	doc, err := s.Execute(context.Background(), q)
	if err != nil || len(doc.Candidates) != 1 || !strings.HasSuffix(doc.Candidates[0].File, "a.md") {
		t.Fatal("document-only search lost", doc, err)
	}
}
