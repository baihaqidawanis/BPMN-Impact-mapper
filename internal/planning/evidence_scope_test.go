package planning

import (
	"context"
	"testing"
)

type scopedTools struct{ candidate Candidate }

func (t scopedTools) Identity() string { return "scope-fixture" }
func (t scopedTools) Execute(context.Context, ToolRequest) (ToolResult, error) {
	return ToolResult{Candidates: []Candidate{t.candidate}}, nil
}

func TestSourceLookupDoesNotAttachToDifferentFunction(t *testing.T) {
	for _, symbol := range []string{"", "Transition"} {
		t.Run("result-"+symbol, func(t *testing.T) {
			b := agentBundle()
			b.Changes[0].Candidates[0].Symbol = "Reverse"
			b.ContextID, _ = ComputeContextID(b)
			candidate := b.Changes[0].Candidates[0]
			candidate.Symbol = symbol
			candidate.Origin = "agent-source-tool"
			candidate.Evidence = []EvidenceItem{{ID: "submit-proof", Type: "SOURCE_TOOL", Location: "store.go:120", Excerpt: "submit branch"}}
			s := AgentSession{ID: "scope", Role: Analyst, Bundle: b, Budget: DefaultAgentBudget()}
			engine := AgentEngine{Tools: scopedTools{candidate}, Store: memorySessions{}}
			request := ToolRequest{Name: "read_source", File: "store.go"}
			if err := engine.apply(context.Background(), &s, AgentReply{Tool: &request}, b); err != nil {
				t.Fatal(err)
			}
			got := s.Bundle.Changes[0].Candidates
			if len(got) != 2 || got[0].Symbol != "Reverse" || len(got[0].Evidence) != 1 || got[0].Evidence[0].ID != "proof" || got[1].Symbol != symbol {
				t.Fatalf("cross-function evidence contamination: %+v", got)
			}
		})
	}
}

func TestFinalReviewExcludesUnplannedRetrievalButKeepsRequirements(t *testing.T) {
	b := agentBundle()
	extra := b.Changes[0].Candidates[0]
	extra.File = "unrelated.go"
	b.Changes[0].Candidates = append(b.Changes[0].Candidates, extra)
	missing := b.Changes[0]
	missing.ElementID = "unresolved-change"
	missing.Candidates = nil
	b.Changes = append(b.Changes, missing)
	b.ContextID, _ = ComputeContextID(b)
	p := FilePlan{ChangeType: b.Changes[0].ChangeType, ElementID: "task", File: "store.go", Symbol: "Submit"}
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if len(turn.Bundle.Changes) != 2 || len(turn.Bundle.Changes[0].Candidates) != 1 || turn.Bundle.Changes[0].Candidates[0].Evidence[0].ID != "proof" || turn.Bundle.Changes[1].ElementID != "unresolved-change" {
			t.Fatalf("lost requirement/evidence or retained unrelated inventory: %+v", turn.Bundle)
		}
		return AgentReply{Review: &AgentReview{Issues: []string{"unresolved change requires inspection"}}}, nil
	}}
	_, _, err := (FullRepositoryEngine{Model: model, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "compact", AgentSession{Bundle: b, Draft: []FilePlan{p}})
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Changes[0].Candidates) != 2 {
		t.Fatal("mutated checkpoint")
	}
}

func TestNamedSourcePrefersExactFunctionOverEarlierWholeFile(t *testing.T) {
	b := agentBundle()
	whole := b.Changes[0].Candidates[0]
	whole.Symbol = ""
	b.Changes[0].Candidates = append([]Candidate{whole}, b.Changes[0].Candidates...)
	b.ContextID, _ = ComputeContextID(b)
	candidate := b.Changes[0].Candidates[1]
	candidate.Origin = "agent-source-tool"
	candidate.Evidence = []EvidenceItem{{ID: "exact-proof", Type: "SOURCE_TOOL", Location: "store.go:2", Excerpt: "submit body"}}
	s := AgentSession{ID: "exact", Role: Analyst, Bundle: b, Budget: DefaultAgentBudget()}
	e := AgentEngine{Tools: scopedTools{candidate}, Store: memorySessions{}}
	q := ToolRequest{Name: "read_function", File: "store.go", Symbol: "Submit"}
	if err := e.apply(context.Background(), &s, AgentReply{Tool: &q}, b); err != nil {
		t.Fatal(err)
	}
	got := s.Bundle.Changes[0].Candidates
	if len(got) != 2 || len(got[0].Evidence) != 1 {
		t.Fatal("exact source attached to whole-file target")
	}
	found := false
	for _, proof := range got[1].Evidence {
		found = found || proof.ID == "exact-proof"
	}
	if !found {
		t.Fatal("exact function lost source")
	}
}
