package planning

import (
	"context"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type namedDiscoveryTool struct{}

func (namedDiscoveryTool) Identity() string { return "named-source" }
func (namedDiscoveryTool) Execute(context.Context, ToolRequest) (ToolResult, error) {
	return ToolResult{Candidates: []Candidate{{File: "store.go", Symbol: "Store.Submit", Module: "core", Layer: "application", Origin: "agent-source-tool", SuggestedAction: domain.ActionVerify, Evidence: []EvidenceItem{{ID: "named-proof", Type: "SOURCE_TOOL", Location: "store.go:3", Excerpt: "func (s Store) Submit(reason string) error"}}}}}, nil
}

func TestNamedDiscoveryEnrichesExistingWholeFileWithoutDuplicateTarget(t *testing.T) {
	b := agentBundle()
	b.Changes[0].Candidates[0].Symbol = ""
	b.ContextID, _ = ComputeContextID(b)
	s := AgentSession{Role: Analyst, Bundle: b, Budget: DefaultAgentBudget()}
	e := AgentEngine{Tools: namedDiscoveryTool{}, Store: memorySessions{}}
	request := ToolRequest{Name: "search_source", ChangeIndex: 0, Query: "submit reason"}
	if err := e.apply(context.Background(), &s, AgentReply{Tool: &request}, b); err != nil {
		t.Fatal(err)
	}
	candidates := s.Bundle.Changes[0].Candidates
	if len(candidates) != 1 || candidates[0].Symbol != "" {
		t.Fatal("whole-file seed renamed or duplicated", candidates)
	}
	found := false
	for _, v := range candidates[0].Evidence {
		if v.ID == "named-proof" {
			found = true
		}
	}
	if !found {
		t.Fatal("named source evidence lost")
	}
	// Distinct named functions still remain distinct targets when no whole-file
	// seed covers them; this is not path-only deduplication for all candidates.
	b = agentBundle()
	s = AgentSession{Role: Analyst, Bundle: b, Budget: DefaultAgentBudget()}
	if err := e.apply(context.Background(), &s, AgentReply{Tool: &request}, b); err != nil {
		t.Fatal(err)
	}
	if len(s.Bundle.Changes[0].Candidates) != 2 {
		t.Fatal("distinct functions merged")
	}
}
