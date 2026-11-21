package planning

import (
	"context"
	"strings"
	"testing"
)

func TestFocusedAuditsOnceAndCorrectsOnlyFailedTargetWithoutAnalystReplay(t *testing.T) {
	for _, stillRejected := range []bool{false, true} {
		t.Run(map[bool]string{false: "corrected", true: "unresolved"}[stillRejected], func(t *testing.T) {
			b := agentBundle()
			other := b.Changes[0].Candidates[0]
			other.File = "other.go"
			b.Changes[0].Candidates = append(b.Changes[0].Candidates, other)
			b.ContextID, _ = ComputeContextID(b)
			analyst, planner, reviewer := 0, 0, 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				switch turn.Role {
				case Analyst:
					analyst++
				case Planner:
					planner++
					if planner == 2 {
						if len(turn.Bundle.Changes) != 1 || len(turn.Bundle.Changes[0].Candidates) != 1 || turn.Bundle.Changes[0].Candidates[0].File != "store.go" || len(turn.Draft) != 1 || turn.RemainingTools != 0 || turn.RemainingRevisions != 0 || !strings.Contains(strings.Join(turn.Feedback, " "), "TARGETED CORRECTION") {
							t.Fatal("correction lost scope/prior draft or can loop", turn)
						}
					}
				case Reviewer:
					reviewer++
					if len(turn.Draft) != 2 {
						t.Fatal("combined audit lost target", len(turn.Draft))
					}
					if reviewer == 1 || stillRejected {
						p := turn.Draft[0]
						if p.File != "store.go" {
							p = turn.Draft[1]
						}
						return AgentReply{Review: &AgentReview{Issues: []string{"Specify the missing failure check for store.go."}, RevisionTarget: Planner, Findings: []FileReviewFinding{{ChangeType: string(p.ChangeType), ElementID: p.ElementID, File: p.File, Symbol: p.Symbol, Checks: []ReviewCriterion{{Criterion: "failure_tests", Status: "REVISE", Finding: "Specify the missing failure check."}}}}}}, nil
					}
				}
				return goodReply(turn)
			}}
			e := FocusedEngine{Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}
			s, err := e.Run(context.Background(), "targeted-restore", b)
			want := "COMPLETE"
			if stillRejected {
				want = "NEEDS_REVIEW"
			}
			if err != nil || s.Status != want || analyst != 1 || planner != 2 || reviewer != 2 || len(s.Draft) != 2 || s.Revisions != 1 {
				t.Fatal("unexpected loop or lost draft", s.Status, analyst, planner, reviewer, s.Revisions, err)
			}
			_, err = e.Run(context.Background(), "targeted-restore", b)
			if err != nil || analyst != 1 || planner != 2 || reviewer != 2 {
				t.Fatal("resume replayed final work", err)
			}
		})
	}
}

func TestDeferredReviewCannotResumeUnderAutoReviewedPolicy(t *testing.T) {
	e := AgentEngine{Model: scriptedAgent{goodReply}, Tools: fixtureTools{"fixed"}, Store: memorySessions{}, DeferReview: true}
	s, err := e.Run(context.Background(), "deferred", agentBundle())
	if err != nil || s.Status != "NEEDS_REVIEW" || !s.AwaitingCombinedReview || s.LastReview != nil {
		t.Fatal("unreviewed child mislabeled", s.Status, err)
	}
	e.DeferReview = false
	if _, err = e.Run(context.Background(), "deferred", agentBundle()); err == nil {
		t.Fatal("policy change resumed checkpoint")
	}
}

func TestFocusedMissingSourceDoesNotSendPlannerIntoCorrectionLoop(t *testing.T) {
	modelCalls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) { modelCalls++; return goodReply(turn) }}
	b := agentBundle()
	s := AgentSession{Bundle: b}
	r, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	s.Decisions = r.Verification.Decisions
	p, _ := goodReply(AgentTurn{Role: Planner, Bundle: b, Decisions: s.Decisions})
	s.Draft = p.Plan.Plans
	d := s.Draft[0]
	review := AgentReview{RevisionTarget: Planner, Issues: []string{"Source is missing."}, Findings: []FileReviewFinding{{ChangeType: string(d.ChangeType), ElementID: d.ElementID, File: d.File, Symbol: d.Symbol, Checks: []ReviewCriterion{{Status: "REVISE"}, {Status: "NEED_EVIDENCE"}}}}}
	changed, err := (FocusedEngine{Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}).reviseFocusedPlan(context.Background(), "missing-source", &s, review)
	if err != nil || changed || modelCalls != 0 {
		t.Fatal("missing source prompted speculative correction", changed, modelCalls, err)
	}
}

func TestUltraLightCorrectsOnlyTwoSourceBoundTargets(t *testing.T) {
	b := agentBundle()
	for _, file := range []string{"second.go", "third.go", "fourth.go"} {
		candidate := b.Changes[0].Candidates[0]
		candidate.File = file
		b.Changes[0].Candidates = append(b.Changes[0].Candidates, candidate)
	}
	b.ContextID, _ = ComputeContextID(b)
	verification, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	planned, _ := goodReply(AgentTurn{Role: Planner, Bundle: b})
	s := AgentSession{Bundle: b, Decisions: verification.Verification.Decisions, Draft: planned.Plan.Plans}
	review := AgentReview{RevisionTarget: Planner}
	for _, p := range s.Draft {
		review.Findings = append(review.Findings, FileReviewFinding{ChangeType: string(p.ChangeType), ElementID: p.ElementID, File: p.File, Symbol: p.Symbol, Checks: []ReviewCriterion{{Criterion: "failure_tests", Status: "REVISE", Finding: "Specify the failure check."}}})
	}
	review.Findings[0].Checks = append(review.Findings[0].Checks, ReviewCriterion{Status: "NEED_EVIDENCE"})
	calls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		candidates := turn.Bundle.Changes[0].Candidates
		if turn.Role != Planner || turn.RemainingTools != 0 || len(candidates) != 2 || candidates[0].File != "second.go" || candidates[1].File != "third.go" {
			t.Fatalf("unexpected correction scope: %+v", turn)
		}
		return goodReply(turn)
	}}
	engine := FocusedEngine{UltraLight: true, Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}
	changed, err := engine.reviseFocusedPlan(context.Background(), "ultra-bounded-repair", &s, review)
	if err != nil || !changed || calls != 1 || s.Revisions != 1 || len(s.Draft) != 4 {
		t.Fatalf("expected one bounded correction preserving other proposals, got changed=%v calls=%d revisions=%d plans=%d err=%v", changed, calls, s.Revisions, len(s.Draft), err)
	}
}
