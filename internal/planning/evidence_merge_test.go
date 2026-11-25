package planning

import (
	"context"
	"fmt"
	"testing"
)

func TestSourceWindowRetainsActionBoundaryThroughMergeAndPlanning(t *testing.T) {
	b := agentBundle()
	b.Changes[0].Candidates[0].Evidence = []EvidenceItem{{ID: "signature", Type: "SYMBOL", Excerpt: "Dispatch(action, reason)"}, {ID: "hit", Type: "BM25", Excerpt: "state query"}}
	b.ContextID, _ = ComputeContextID(b)
	window := []EvidenceItem{}
	for i, text := range []string{`case "publish":`, `if state != "DRAFT" {`, `return conflict`, `}`, `state = "PUBLISHED"`, `case "archive":`, `auditArchive()`, `return nil`} {
		window = append(window, EvidenceItem{ID: fmt.Sprint("line", i), Type: "SOURCE_TOOL", Excerpt: text})
	}
	s := AgentSession{Role: Analyst, Bundle: b, Budget: DefaultAgentBudget()}
	tools := evidenceWindowTools{window}
	e := AgentEngine{Tools: tools, Store: memorySessions{}}
	if err := e.apply(context.Background(), &s, AgentReply{Tool: &ToolRequest{Name: "inspect_source", File: "store.go", Query: "publish"}}, b); err != nil {
		t.Fatal(err)
	}
	proof := s.Bundle.Changes[0].Candidates[0].Evidence
	if len(proof) != 8 || proof[2].Excerpt != `case "publish":` || proof[3].Excerpt != `if state != "DRAFT" {` || proof[6].Excerpt != `state = "PUBLISHED"` {
		t.Fatal("action proof was evicted", proof)
	}
	next := MergeAgentEvidence(proof, []EvidenceItem{{ID: "later-search", Type: "SOURCE_TOOL", Excerpt: "unrelated keyword"}}, false)
	if next[2].ID != "line0" || next[6].ID != "line4" {
		t.Fatal("search evicted inspected action", next)
	}
}

type evidenceWindowTools struct{ items []EvidenceItem }

func TestDiscoveredDeclarationSurvivesLaterActionWindow(t *testing.T) {
	existing := []EvidenceItem{{ID: "keyword", Type: "SOURCE_TOOL", Excerpt: `case "publish":`}, {ID: "signature", Type: "SOURCE_DECLARATION", Excerpt: "func Dispatch(action, reason string) error {"}}
	window := []EvidenceItem{}
	for i, text := range []string{`case "publish":`, `if state != "DRAFT" {`, `return conflict`, `}`, `state = "PUBLISHED"`, `case "archive":`, `auditArchive()`, `return nil`} {
		window = append(window, EvidenceItem{ID: fmt.Sprint("window", i), Type: "SOURCE_TOOL", Excerpt: text})
	}
	merged := MergeAgentEvidence(existing, window, true)
	if len(merged) != 8 || merged[0].ID != "signature" || merged[1].ID != "window0" || merged[5].Excerpt != `state = "PUBLISHED"` {
		t.Fatal("source contract or mutation lost", merged)
	}
}

func TestInspectedCallEvidencePrecedesLexicalHits(t *testing.T) {
	existing := []EvidenceItem{{ID: "sales-hit", Type: "BM25", Excerpt: "unrelated sales test"}}
	incoming := []EvidenceItem{{ID: "inventory-call", Type: "AST_CALL_LITERALS"}, {ID: "inventory-source", Type: "SOURCE_TOOL"}}
	merged := MergeAgentEvidence(existing, incoming, true)
	if len(merged) != 3 || merged[0].ID != "inventory-call" || merged[1].ID != "inventory-source" {
		t.Fatal("retrieval hit was prioritized over inspected call", merged)
	}
}

func (evidenceWindowTools) Identity() string { return "window-fixture" }
func (f evidenceWindowTools) Execute(context.Context, ToolRequest) (ToolResult, error) {
	return ToolResult{Candidates: []Candidate{{File: "store.go", Symbol: "Submit", Origin: "agent-source-tool", SuggestedAction: "VERIFY", Evidence: f.items}}}, nil
}
