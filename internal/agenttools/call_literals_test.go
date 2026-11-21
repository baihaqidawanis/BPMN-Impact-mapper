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

func TestInspectedTestCallKeepsEvidenceInMatchingOperation(t *testing.T) {
	root := t.TempDir()
	const file = "operations_test.go"
	code := `package sample
func TestSalesOrder(t *testing.T) {
 if err := service.Submit(ctx, order.ID, "submit", actor, "", order.Version, "sales-submit"); err != nil {
  t.Fatal(err)
 }
}
func TestStockAdjustment(t *testing.T) {
 if err := service.Transition(ctx, adjustment.ID, "submit", actor, "", adjustment.Version, "inventory-submit"); err != nil {
  t.Fatal(err)
 }
 if adjustment.Status != "SUBMITTED" { t.Fatal(adjustment.Status) }
}
`
	if err := os.WriteFile(filepath.Join(root, file), []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	snapshot, err := agenttools.New(ctx, root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := snapshot.Execute(ctx, planning.ToolRequest{Name: "inspect_source", File: file, Query: "submit stock adjustment"})
	if err != nil || len(result.Candidates) != 1 {
		t.Fatal(result, err)
	}
	var call, outcome bool
	for _, evidence := range result.Candidates[0].Evidence {
		if strings.Contains(evidence.Excerpt, "sales-submit") || strings.Contains(evidence.Excerpt, "TestSalesOrder") {
			t.Fatal("evidence from unrelated operation", evidence)
		}
		call = call || (evidence.Type == "AST_CALL_LITERALS" && strings.Contains(evidence.Excerpt, `5="", 7="inventory-submit"`))
		outcome = outcome || strings.Contains(evidence.Excerpt, `adjustment.Status != "SUBMITTED"`)
	}
	if !call || !outcome {
		t.Fatal("missing call or expected outcome", result)
	}
}
