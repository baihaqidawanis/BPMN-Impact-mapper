package planning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func batchReviewFixture() EvidenceBundle {
	b := agentBundle()
	c := b.Changes[0].Candidates[0]
	c.File = "second.go"
	c.Evidence = []EvidenceItem{{ID: "second-proof", Excerpt: "publish"}}
	b.Changes[0].Candidates = append(b.Changes[0].Candidates, c)
	b.ContextID, _ = ComputeContextID(b)
	return b
}

type fileScopedReviewFixture struct{ scriptedAgent }

func (fileScopedReviewFixture) PreferFileReview() bool { return true }

func TestPreferredFileReviewResumesAndRetainsFinalCrossFileCheck(t *testing.T) {
	calls := map[string]int{}
	disconnect := true
	model := fileScopedReviewFixture{scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role != Reviewer {
			return goodReply(turn)
		}
		instruction := strings.Join(turn.Feedback, " ")
		if strings.HasPrefix(instruction, "FINAL FILE") {
			if len(turn.Draft) != 1 || len(turn.Bundle.Changes[0].Candidates) != 1 || len(turn.Bundle.Changes[0].Candidates[0].Evidence) == 0 {
				t.Fatal("file review lost approved source", turn)
			}
			if len(turn.RelatedProposals) != 1 || turn.RelatedProposals[0].File == turn.Draft[0].File {
				t.Fatal("coordinated file responsibility lost", turn.RelatedProposals)
			}
			file := turn.Draft[0].File
			calls[file]++
			if file == "store.go" && disconnect {
				disconnect = false
				return AgentReply{}, errors.New("connection interrupted")
			}
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		if strings.HasPrefix(instruction, "FINAL CONTRACT") {
			if len(turn.Draft) != 2 || len(turn.Bundle.Changes[0].Candidates) != 0 {
				t.Fatal("contract review lost files or claims to have source")
			}
			calls["contracts"]++
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		t.Fatal("policy sent an unpartitioned model review")
		return AgentReply{}, nil
	}}}
	engine := AgentEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	first, err := engine.Run(context.Background(), "preferred", batchReviewFixture())
	if err == nil || first.Status != "PAUSED" {
		t.Fatal(first.Status, err)
	}
	done, err := engine.Run(context.Background(), "preferred", batchReviewFixture())
	if err != nil || done.Status != "COMPLETE" || calls["second.go"] != 1 || calls["store.go"] != 2 || calls["contracts"] != 1 || done.Turns != 8 {
		t.Fatal("completed review repeated or attempts escaped budget", done.Status, done.Turns, calls, err)
	}
	limited := AgentEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}, Budget: AgentBudget{MaxTurns: 5}}
	result, _ := limited.Run(context.Background(), "preferred-limited", batchReviewFixture())
	if result.Status != "NEEDS_REVIEW" || result.Turns != 5 {
		t.Fatal("policy bypassed the session budget", result.Status, result.Turns)
	}
}

func TestPartitionedReviewRepairForwardsFeedbackAndDoesNotReuseOldReceipt(t *testing.T) {
	broken := true
	seenRepair := false
	model := fileScopedReviewFixture{scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role != Reviewer {
			return goodReply(turn)
		}
		instruction := strings.Join(turn.Feedback, " ")
		if strings.HasPrefix(instruction, "FINAL FILE") {
			if broken {
				broken = false
				return AgentReply{}, ErrAgentModelOutput
			}
			if !strings.Contains(instruction, "Model response rejected for reviewer") {
				t.Fatal("partitioned output repair repeated the original prompt")
			}
			seenRepair = true
		}
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}}
	engine := AgentEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	result, err := engine.Run(context.Background(), "repair-partition", batchReviewFixture())
	if err != nil || result.Status != "COMPLETE" || result.OutputRepairs != 1 || !seenRepair {
		t.Fatal(result.Status, result.Feedback, seenRepair, err)
	}
	// A final consistency receipt is also invalidated by changed feedback.
	calls := 0
	plain := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		calls++
		if len(turn.Feedback) != 2 || turn.Feedback[1] == "" {
			t.Fatal("root feedback missing from final review")
		}
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}
	store := memorySessions{}
	full := FullRepositoryEngine{Model: plain, Store: store}
	s := AgentSession{Binding: "bound", Bundle: agentBundle(), Draft: []FilePlan{{File: "store.go"}}, Feedback: []string{"First repair"}}
	for _, message := range []string{"First repair", "First repair", "Different repair"} {
		s.Feedback = []string{message}
		if _, _, err := full.reviewCombinedPlan(context.Background(), "feedback-receipt", s); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatal("changed feedback reused stale approval or identical feedback repeated inference", calls)
	}
}
func TestOversizedChildReviewPartitionsAndResumesWithoutRepeatingApprovedFiles(t *testing.T) {
	calls := map[string]int{}
	disconnect := true
	total := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		total++
		if turn.Role != Reviewer {
			return goodReply(turn)
		}
		instruction := strings.Join(turn.Feedback, " ")
		if strings.HasPrefix(instruction, "FINAL FILE") {
			file := turn.Draft[0].File
			calls[file]++
			if file == "store.go" && disconnect {
				disconnect = false
				return AgentReply{}, errors.New("disconnected")
			}
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		if strings.HasPrefix(instruction, "FINAL CONTRACT") {
			if len(turn.Draft) != 2 {
				t.Fatal("missing cross-file proposal")
			}
			calls["contract"]++
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		return AgentReply{}, ErrAgentContextLimit
	}}
	store := memorySessions{}
	e := AgentEngine{Model: model, Tools: fixtureTools{"fixture"}, Store: store}
	first, err := e.Run(context.Background(), "partition-child", batchReviewFixture())
	if err == nil || first.Status != "PAUSED" {
		t.Fatal(first.Status, err)
	}
	before := calls["second.go"]
	done, err := e.Run(context.Background(), "partition-child", batchReviewFixture())
	if err != nil || done.Status != "COMPLETE" || calls["second.go"] != before || calls["contract"] != 1 {
		t.Fatal(done.Status, done.Feedback, calls, err)
	}
	if done.Turns != total {
		t.Fatal("partition calls escaped turn accounting", done.Turns, total)
	}
}
func TestPartitionedChildReviewCannotExceedTurnBudgetOrApproveRejectedFile(t *testing.T) {
	for _, budget := range []int{5, 18} {
		t.Run(string(rune('a'+budget)), func(t *testing.T) {
			total := 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				total++
				if turn.Role != Reviewer {
					return goodReply(turn)
				}
				instruction := strings.Join(turn.Feedback, " ")
				if strings.HasPrefix(instruction, "FINAL FILE") {
					if budget == 18 {
						return AgentReply{Review: &AgentReview{Issues: []string{"Proposed guard changes a different operation."}}}, nil
					}
					return AgentReply{Review: &AgentReview{Approved: true}}, nil
				}
				if strings.HasPrefix(instruction, "FINAL CONTRACT") {
					t.Fatal("unexpected approval stage")
				}
				return AgentReply{}, ErrAgentContextLimit
			}}
			e := AgentEngine{Model: model, Tools: fixtureTools{"fixture"}, Store: memorySessions{}, Budget: AgentBudget{MaxTurns: budget, MaxRevisions: 1}}
			result, _ := e.Run(context.Background(), "partition-budget", batchReviewFixture())
			if result.Status != "NEEDS_REVIEW" || result.Turns > budget || total > budget {
				t.Fatal(result.Status, result.Turns, total)
			}
		})
	}
}
