package agenttools_test

import (
	"context"
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositorySearchPagesDistinctFilesBeyondInitialLeads(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		code := fmt.Sprintf("package fixture\nfunc Validate%d(){}\nfunc ValidateMore%d(){}\n", i, i)
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("file%d.go", i)), []byte(code), 0600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, offset := range []int{0, 3} {
		result, err := s.Execute(context.Background(), planning.ToolRequest{Name: "search_source", Query: "validate", Offset: offset})
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range result.Candidates {
			if seen[c.File] {
				t.Fatal("duplicate file starves repository discovery", c.File)
			}
			seen[c.File] = true
		}
	}
	if len(seen) != 5 {
		t.Fatal("later files unreachable", len(seen))
	}
	if _, err := s.Execute(context.Background(), planning.ToolRequest{Name: "search_source", Query: "validate", Offset: -1}); err == nil {
		t.Fatal("negative offset accepted")
	}
}
