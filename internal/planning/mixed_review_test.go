package planning

import (
	"context"
	"strings"
	"testing"
)

func TestMixedReviewCorrectsOnlySourceBackedTarget(t *testing.T) {
	b := agentBundle()
	other := b.Changes[0].Candidates[0]
	other.File = "unknown.go"
	b.Changes[0].Candidates = append(b.Changes[0].Candidates, other)
	b.ContextID, _ = ComputeContextID(b)
	verification, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	proposal, _ := goodReply(AgentTurn{Role: Planner, Bundle: b, Decisions: verification.Verification.Decisions})
	s := AgentSession{Bundle: b, Decisions: verification.Verification.Decisions, Draft: proposal.Plan.Plans}
	findings := []FileReviewFinding{}
	for i, p := range s.Draft {
		checks := []ReviewCriterion{{Status: "REVISE", Finding: "correct explicit edit", Criterion: "scope"}}
		if i == 1 {
			checks = append(checks, ReviewCriterion{Status: "NEED_EVIDENCE", Finding: "missing source"})
		}
		findings = append(findings, FileReviewFinding{ChangeType: string(p.ChangeType), ElementID: p.ElementID, File: p.File, Symbol: p.Symbol, Checks: checks})
	}
	calls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if turn.Role != Planner || len(turn.Bundle.Changes[0].Candidates) != 1 || turn.Bundle.Changes[0].Candidates[0].File != "store.go" || len(turn.RelatedProposals) != 1 || strings.Contains(strings.Join(turn.Feedback, " "), "missing source") {
			t.Fatal("mixed source uncertainty contaminated correction", turn)
		}
		return goodReply(turn)
	}}
	changed, err := (FocusedEngine{Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}).reviseFocusedPlan(context.Background(), "mixed", &s, AgentReview{RevisionTarget: Analyst, Findings: findings})
	if err != nil || !changed || calls != 1 || len(s.Draft) != 2 {
		t.Fatal(changed, calls, err)
	}
}
