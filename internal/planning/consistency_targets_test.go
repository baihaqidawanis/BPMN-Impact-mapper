package planning

import (
	"context"
	"strings"
	"testing"
)

func TestOversizedFileReviewChecksEveryTargetThenContracts(t *testing.T) {
	for _, reject := range []bool{false, true} {
		b := agentBundle()
		plans := []FilePlan{}
		for _, id := range []string{"task", "flow"} {
			plans = append(plans, FilePlan{ChangeType: b.Changes[0].ChangeType, ElementID: id, File: "store.go", Symbol: "Submit", CurrentBehavior: "current", Steps: []string{"complete step"}, AcceptanceChecks: []string{"complete check"}})
		}
		second := b.Changes[0]
		second.ElementID = "flow"
		b.Changes = append(b.Changes, second)
		targets, contracts := 0, 0
		model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
			instruction := turn.Feedback[0]
			if strings.HasPrefix(instruction, "FINAL TARGET SOURCE") {
				targets++
				if len(turn.Draft) != 1 || len(turn.Draft[0].Steps) != 1 || len(turn.Draft[0].AcceptanceChecks) != 1 || len(turn.Bundle.Changes) != 2 {
					t.Fatal("target lost plan or requirements")
				}
				if reject && turn.Draft[0].ElementID == "flow" {
					return AgentReply{Review: &AgentReview{Issues: []string{"wrong operation"}, RevisionTarget: Planner}}, nil
				}
				return AgentReply{Review: &AgentReview{Approved: true}}, nil
			}
			if strings.HasPrefix(instruction, "FINAL CONTRACT REVIEW") {
				contracts++
				if len(turn.Draft) != 2 {
					t.Fatal("lost cross-change proposals")
				}
				return AgentReply{Review: &AgentReview{Approved: true}}, nil
			}
			return AgentReply{}, ErrAgentContextLimit
		}}
		engine := FullRepositoryEngine{Model: model, Store: memorySessions{}}
		review, _, err := engine.reviewCombinedPlan(context.Background(), "target-fallback", AgentSession{Bundle: b, Draft: plans})
		if err != nil || targets != 2 || review.Approved == reject {
			t.Fatal(review, targets, contracts, err)
		}
		if (!reject && contracts != 1) || (reject && contracts != 0) {
			t.Fatal("contract approval bypass", contracts)
		}
		_, _, err = engine.reviewCombinedPlan(context.Background(), "target-fallback", AgentSession{Bundle: b, Draft: plans})
		if err != nil || targets != 2 {
			t.Fatal("completed targets replayed", targets, err)
		}
	}
}
