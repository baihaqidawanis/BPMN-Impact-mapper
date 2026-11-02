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

func TestReadSourceApprovedWindowAndSnapshot(t *testing.T) {
	root := t.TempDir()
	body := "const handler = () => {};\n" + strings.Repeat("// other line\n", 20) + "const actionReason = action === 'submit' ? reason : '';\n" + strings.Repeat("// tail\n", 10)
	os.WriteFile(filepath.Join(root, "panel.tsx"), []byte(body), 0600)
	os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=secret"), 0600)
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "panel.tsx"), []byte("changed"), 0600)
	r, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_source", File: "panel.tsx", Query: "actionReason"})
	if err != nil || len(r.Candidates) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	found := false
	for _, e := range r.Candidates[0].Evidence {
		if strings.Contains(e.Excerpt, "actionReason") {
			found = true
			if e.Location != "panel.tsx:22" {
				t.Fatal(e.Location)
			}
		}
		if len([]rune(e.Excerpt)) > 240 {
			t.Fatal("unbounded excerpt")
		}
	}
	if !found || r.Candidates[0].Symbol != "" {
		t.Fatal("generic source window not returned")
	}
	for _, q := range []planning.ToolRequest{{Name: "read_source", File: "../panel.tsx"}, {Name: "read_source", File: ".env"}, {Name: "read_source", File: "panel.tsx", StartLine: 1000}, {Name: "read_source", File: "panel.tsx", StartLine: -1}, {Name: "read_source", File: "panel.tsx", Query: "absent"}} {
		if _, err := s.Execute(context.Background(), q); err == nil {
			t.Fatalf("unsafe window accepted: %+v", q)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Execute(ctx, planning.ToolRequest{Name: "read_source", File: "panel.tsx"}); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestReadSourceFindsTextInsideLongCompressedLine(t *testing.T) {
	root := t.TempDir()
	code := strings.Repeat("x", 700) + " requiredInput={reason} " + strings.Repeat("y", 700)
	if err := os.WriteFile(filepath.Join(root, "view.tsx"), []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Execute(context.Background(), planning.ToolRequest{Name: "read_source", File: "view.tsx", Query: "requiredInput"})
	if err != nil || len(r.Candidates) != 1 || !strings.Contains(r.Candidates[0].Evidence[0].Excerpt, "requiredInput={reason}") {
		t.Fatalf("matching span missing: %+v %v", r, err)
	}
	if len([]rune(r.Candidates[0].Evidence[0].Excerpt)) > 240 {
		t.Fatal("unbounded long line")
	}
}
