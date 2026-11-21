package planning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInvalidReviewerAtTurnLimitIsNotOfferedEndlessResume(t *testing.T) {
	calls := 0
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if turn.Role == Reviewer {
			return AgentReply{}, ErrAgentModelOutput
		}
		return goodReply(turn)
	}}
	e := AgentEngine{Model: m, Tools: fixtureTools{"fixed"}, Store: memorySessions{}, Budget: AgentBudget{MaxTurns: 4, MaxTools: 1, MaxRevisions: 1}}
	s, err := e.Run(context.Background(), "invalid-final-turn", agentBundle())
	if !errors.Is(err, ErrAgentModelOutput) || s.Status != "NEEDS_REVIEW" || s.Turns != 4 || !strings.Contains(strings.Join(s.Feedback, " "), "Turn budget exhausted") {
		t.Fatal(s.Status, s.Turns, s.Feedback, err)
	}
	before := calls
	s, err = e.Run(context.Background(), "invalid-final-turn", agentBundle())
	if err != nil || calls != before || s.Status != "NEEDS_REVIEW" {
		t.Fatal("resume reset exhausted budget", s.Status, calls, err)
	}
}

func TestFocusedFinalReviewStopsAfterBoundedInvalidOutputAndExposesIssue(t *testing.T) {
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer {
			return AgentReply{}, errors.Join(ErrAgentModelOutput, errors.New("proposal-only evidence must be empty"))
		}
		return goodReply(turn)
	}}
	s, err := (FocusedEngine{Model: m, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}).Run(context.Background(), "reviewer-feedback", agentBundle())
	if err != nil || s.Status != "NEEDS_REVIEW" || s.Role != Reviewer || len(s.Draft) == 0 || !strings.Contains(strings.Join(s.Feedback, " "), "proposal-only evidence must be empty") {
		t.Fatal(s.Role, s.Status, s.Feedback, err)
	}
}
