package planning

import (
	"context"
	"testing"
)

func TestFocusedPlannerSeesEarlierSameFileProposalWithoutSourcePromotion(t *testing.T) {
	b := agentBundle()
	next := b.Changes[0]
	next.ElementID = "next-change"
	b.Changes = append(b.Changes, next)
	b.ContextID, _ = ComputeContextID(b)
	planners := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Analyst && len(turn.RelatedProposals) != 0 {
			t.Fatal("planning claims leaked into source classification")
		}
		if turn.Role == Planner {
			planners++
			if planners == 1 && len(turn.RelatedProposals) != 0 {
				t.Fatal("unexpected prior plan")
			}
			if planners == 2 {
				if len(turn.RelatedProposals) != 1 || turn.RelatedProposals[0].ElementID != "task" || turn.RelatedProposals[0].File != "store.go" {
					t.Fatal("missing coordinated prior proposal", turn.RelatedProposals)
				}
			}
		}
		return goodReply(turn)
	}}
	e := FocusedEngine{Model: model, Tools: fixtureTools{"coordination"}, Store: memorySessions{}}
	s, err := e.Run(context.Background(), "coordinated", b)
	if err != nil || s.Status != "COMPLETE" || planners != 2 {
		t.Fatal(s.Status, planners, err)
	}
	_, err = e.Run(context.Background(), "coordinated", b)
	if err != nil || planners != 2 {
		t.Fatal("replayed completed work", err)
	}
}

func TestCompanionPlanChangeInvalidatesChildBinding(t *testing.T) {
	b := agentBundle()
	e := AgentEngine{Model: scriptedAgent{goodReply}, Tools: fixtureTools{"coordination"}, Store: memorySessions{}, DeferReview: true, RelatedProposals: []ProposalContract{{File: "other.go", TargetBehavior: "first"}}}
	if _, err := e.Run(context.Background(), "companion", b); err != nil {
		t.Fatal(err)
	}
	e.RelatedProposals[0].TargetBehavior = "different"
	if _, err := e.Run(context.Background(), "companion", b); err == nil {
		t.Fatal("changed companion reused old child")
	}
}
