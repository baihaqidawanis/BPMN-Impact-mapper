package planning

import (
	"context"
	"errors"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func planningCandidateCount(t AgentTurn) int {
	n := 0
	for _, c := range t.Bundle.Changes {
		n += len(c.Candidates)
	}
	return n
}

func TestPartitionedPlannerResumesEveryTargetAndKeepsAllRequirements(t *testing.T) {
	b := batchReviewFixture()
	b.Changes = append(b.Changes, ChangeInput{ChangeType: domain.ModifySequenceFlow, ElementID: "adjacent-flow", ElementName: "Review before submit"})
	b.ContextID, _ = ComputeContextID(b)
	calls := map[string]int{}
	total := 0
	disconnect := true
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		total++
		if turn.Role == Planner {
			if planningCandidateCount(turn) > 1 {
				return AgentReply{}, ErrAgentContextLimit
			}
			if len(turn.Bundle.Changes) != 2 || turn.Bundle.Changes[1].ElementID != "adjacent-flow" {
				t.Fatal("requirements lost in partition")
			}
			file := turn.Decisions[0].File
			calls[file]++
			if file == "second.go" && disconnect {
				disconnect = false
				return AgentReply{}, errors.New("disconnected")
			}
		}
		return goodReply(turn)
	}}
	e := AgentEngine{Model: m, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	first, err := e.Run(context.Background(), "planning-parts", b)
	if err == nil || first.Status != "PAUSED" || len(first.Draft) != 0 {
		t.Fatal(first.Status, first.Draft, err)
	}
	done, err := e.Run(context.Background(), "planning-parts", b)
	if err != nil || done.Status != "COMPLETE" || len(done.Draft) != 2 || calls["store.go"] != 1 || calls["second.go"] != 2 || done.Turns != total {
		t.Fatal(done.Status, done.Feedback, calls, done.Turns, total, err)
	}
}

func TestPartitionedPlannerCannotOverrunBudgetOrSkipReview(t *testing.T) {
	for _, limit := range []int{3, 4, 5} {
		calls, reviews := 0, 0
		m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
			calls++
			if turn.Role == Planner && planningCandidateCount(turn) > 1 {
				return AgentReply{}, ErrAgentContextLimit
			}
			if turn.Role == Reviewer {
				reviews++
			}
			return goodReply(turn)
		}}
		e := AgentEngine{Model: m, Tools: fixtureTools{"source"}, Store: memorySessions{}, Budget: AgentBudget{MaxTurns: limit, MaxRevisions: 1}}
		s, _ := e.Run(context.Background(), "planning-budget", batchReviewFixture())
		if calls > limit || s.Turns > limit {
			t.Fatal("unaccounted model call", calls, s.Turns, limit)
		}
		if limit < 5 && (s.Status != "NEEDS_REVIEW" || reviews != 0) {
			t.Fatal("skipped review became approval", s.Status, reviews)
		}
		if limit == 5 && (s.Status != "COMPLETE" || reviews != 1) {
			t.Fatal(s.Status, reviews, s.Feedback)
		}
	}
}
