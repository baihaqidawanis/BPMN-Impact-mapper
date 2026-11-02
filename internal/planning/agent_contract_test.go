package planning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAgentRevisionRetainsSourceObservations(t *testing.T) {
	calls, reviews := 0, 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if calls == 1 {
			return AgentReply{Tool: &ToolRequest{Name: "search_source", Query: "submit"}}, nil
		}
		if turn.Role == Reviewer && reviews == 0 {
			reviews++
			return AgentReply{Review: &AgentReview{Issues: []string{"Inspect the next source window."}, RevisionTarget: Analyst}}, nil
		}
		if reviews > 0 && turn.Role == Analyst && len(turn.Observations) != 1 {
			t.Fatal("source history discarded on revision")
		}
		return goodReply(turn)
	}}
	s, err := (AgentEngine{Model: model, Tools: fixtureTools{"s"}, Store: memorySessions{}}).Run(context.Background(), "retain-source", agentBundle())
	if err != nil || s.Status != "COMPLETE" || s.Tools != 1 {
		t.Fatal(s.Status, s.Tools, err)
	}
}

func TestAgentInspectionOnlyPlanGetsOneRepair(t *testing.T) {
	for _, repair := range []bool{true, false} {
		t.Run(map[bool]string{true: "repaired", false: "bounded"}[repair], func(t *testing.T) {
			plannerCalls := 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				reply, err := goodReply(turn)
				if turn.Role == Planner {
					plannerCalls++
					if plannerCalls == 1 || !repair {
						reply.Plan.Plans[0].ProposedChanges = nil
						reply.Plan.Plans[0].AcceptanceChecks = nil
					}
					if plannerCalls == 2 && len(turn.Feedback) == 0 {
						t.Fatal("repair instructions missing")
					}
				}
				return reply, err
			}}
			s, err := (AgentEngine{Model: model, Tools: fixtureTools{"s"}, Store: memorySessions{}}).Run(context.Background(), "repair-output", agentBundle())
			repairs := 0
			for _, event := range s.Events {
				if event.Kind == "OUTPUT_REPAIR_REQUESTED" {
					repairs++
				}
			}
			if repairs != 1 {
				t.Fatal("repair event missing from checkpoint history")
			}
			if plannerCalls != 2 || s.OutputRepairs != 1 {
				t.Fatal("unbounded repair", plannerCalls, s.OutputRepairs)
			}
			if repair {
				if err != nil || s.Status != "COMPLETE" {
					t.Fatal(err, s.Status)
				}
				p, e := s.ChangePlan()
				if e != nil || !strings.Contains(string(Markdown(p)), "### Acceptance checks") {
					t.Fatal("missing structured plan", e)
				}
			} else if !errors.Is(err, ErrAgentPlanIncomplete) || s.Status != "NEEDS_REVIEW" {
				t.Fatal(err, s.Status)
			}
		})
	}
}
