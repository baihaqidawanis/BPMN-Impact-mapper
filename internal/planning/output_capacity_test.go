package planning

import (
	"context"
	"testing"
)

func TestOutputCapacitySplitsAnalystAndPlannerWithoutLosingTargets(t *testing.T) {
	for _, role := range []AgentRole{Analyst, Planner} {
		t.Run(string(role), func(t *testing.T) {
			b := batchReviewFixture()
			overflows := 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				if turn.Role == role && planningCandidateCount(turn) > 1 {
					overflows++
					return AgentReply{}, ErrAgentOutputLimit
				}
				return goodReply(turn)
			}}
			s, err := (FocusedEngine{Model: model, Tools: fixtureTools{"output-bound"}, Store: memorySessions{}}).Run(context.Background(), "output-capacity", b)
			if err != nil || s.Status != "COMPLETE" || len(s.Draft) != 2 || overflows != 1 || s.FocusedReview.CandidatesProcessed != 2 {
				t.Fatal(s.Status, len(s.Draft), overflows, s.Feedback, err)
			}
		})
	}
}

func TestSingleTargetOutputLimitDoesNotRepeatFormatRepair(t *testing.T) {
	calls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) { calls++; return AgentReply{}, ErrAgentOutputLimit }}
	s, err := (AgentEngine{Model: model, Tools: fixtureTools{"output-bound"}, Store: memorySessions{}}).Run(context.Background(), "single-output", agentBundle())
	if err != ErrAgentOutputLimit || calls != 1 || s.Status != "PAUSED" || s.OutputRepairs != 0 {
		t.Fatal(s.Status, calls, s.OutputRepairs, err)
	}
}
