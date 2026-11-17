package agenttools_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestInspectedGoCallReportsLiteralPositionsWithoutEvaluatingExpressions(t *testing.T) {
	root := t.TempDir()
	code := `package sample
func run() {
 client.Send(ctx, id, "publish", actor, "", version, request)
 client.Send(ctx, id, "publish", actor, "" + reason, version, request)
}
`
	file := filepath.Join(root, "flow.go")
	if err := os.WriteFile(file, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := agenttools.New(context.Background(), root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	q := planning.ToolRequest{Name: "inspect_source", File: "flow.go", Query: "publish"}
	r, err := s.Execute(context.Background(), q)
	if err != nil || len(r.Candidates) != 1 {
		t.Fatal(r, err)
	}
	var fact string
	raw := false
	for _, e := range r.Candidates[0].Evidence {
		if len([]rune(e.Excerpt)) > 240 {
			t.Fatal("unbounded evidence")
		}
		if e.Type == "AST_CALL_LITERALS" {
			fact = e.Excerpt
			if e.Location != "flow.go:3:2" {
				t.Fatal("wrong call location", e.Location)
			}
		} else if strings.Contains(e.Excerpt, "client.Send") {
			raw = true
		}
	}
	if !strings.Contains(fact, `3="publish", 5=""`) || !strings.Contains(fact, "1-based") || strings.Contains(fact, "reason=") || !raw {
		t.Fatal("missing/overstated syntactic fact", fact)
	}
	merged := planning.MergeAgentEvidence([]planning.EvidenceItem{{ID: "decl", Type: "SOURCE_DECLARATION", Excerpt: "function declaration"}, {ID: "hit", Type: "BM25", Excerpt: "search hit"}}, r.Candidates[0].Evidence, true)
	retained := false
	for _, e := range merged {
		if e.Type == "AST_CALL_LITERALS" {
			retained = true
		}
	}
	if !retained || len(merged) > 8 {
		t.Fatal("fact lost before model input or evidence bound exceeded")
	}
	again, err := s.Execute(context.Background(), q)
	if err != nil || !reflect.DeepEqual(r, again) {
		t.Fatal("non-deterministic observations", err)
	}
	after, _ := os.ReadFile(file)
	if string(after) != code {
		t.Fatal("source modified")
	}
	// Snapshot reads must not follow later filesystem changes.
	os.WriteFile(file, []byte("changed"), 0600)
	frozen, err := s.Execute(context.Background(), q)
	if err != nil || !reflect.DeepEqual(r, frozen) {
		t.Fatal("facts not bound to approved snapshot", err)
	}
}

func TestCallFactsAbstainOnMalformedGoAndOtherLanguages(t *testing.T) {
	for _, tc := range []struct{ name, code string }{
		{"bad.go", `package p; func broken( { send("publish", "") }`},
		{"view.ts", `function run() { send("publish", ""); }`},
		{"computed.go", `package p; func run() { send("publish" + suffix, reason) }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			os.WriteFile(filepath.Join(root, tc.name), []byte(tc.code), 0600)
			s, err := agenttools.New(context.Background(), root, ingest.Options{})
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.Execute(context.Background(), planning.ToolRequest{Name: "inspect_source", File: tc.name, Query: "publish"})
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range r.Candidates {
				for _, e := range c.Evidence {
					if e.Type == "AST_CALL_LITERALS" {
						t.Fatal("unsupported syntax promoted to fact", e)
					}
				}
			}
		})
	}
}
