package planning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFinalReviewRepairsPersistSpecificFeedbackAndKeepAttemptBudget(t *testing.T) {
	store := memorySessions{}
	calls := 0
	s := AgentSession{Binding: "bound", Identity: ProviderIdentity{Model: "fixture"}, Bundle: agentBundle(), Draft: []FilePlan{{File: "store.go"}}}
	e := FullRepositoryEngine{Store: store, Model: scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if calls == 1 {
			return AgentReply{}, errors.Join(ErrAgentModelOutput, errors.New("duplicate review JSON field"))
		}
		if !strings.Contains(strings.Join(turn.Feedback, " "), "duplicate review JSON field") {
			t.Fatal("retry lost exact validation reason")
		}
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}}
	r, n, err := e.reviewCombinedPlan(context.Background(), "repair-final-feedback", s)
	if err != nil || !r.Approved || n != 2 || calls != 2 {
		t.Fatal(r, n, calls, err)
	}
	if _, _, err = e.reviewCombinedPlan(context.Background(), "repair-final-feedback", s); err != nil || calls != 2 {
		t.Fatal("completed receipt repeated inference", calls, err)
	}
}

func TestPartialPlanIsUnapprovedAndPreservesUnresolvedTargets(t *testing.T) {
	b := agentBundle()
	reply, _ := goodReply(AgentTurn{Role: Planner, Bundle: b})
	verified, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	b.Changes[0].Candidates = append(b.Changes[0].Candidates, Candidate{File: "other.go", Symbol: "Other", Module: "core", Layer: "application"})
	d := verified.Verification.Decisions[0]
	d.File = "other.go"
	d.Symbol = "Other"
	s := AgentSession{Status: "PAUSED", Bundle: b, Draft: reply.Plan.Plans, Decisions: append(verified.Verification.Decisions, d)}
	if _, err := s.ChangePlan(); err == nil {
		t.Fatal("unfinished checkpoint exported as final")
	}
	p, err := s.PartialPlan()
	if err != nil || p.Status != StatusNeedsReview || len(p.Plans) != 1 || p.ChangeCoverage[0].Relevant != 2 || s.Status != "PAUSED" || len(s.Feedback) != 0 {
		t.Fatal("preview hid coverage or changed checkpoint", p, err)
	}
	if !strings.Contains(string(Markdown(p)), "PARTIAL CHECKPOINT EXPORT") {
		t.Fatal("missing partial warning")
	}
	s.Draft[0].EvidenceIDs = []string{"invented"}
	if _, err := s.PartialPlan(); err == nil {
		t.Fatal("preview bypassed evidence validation")
	}
}
