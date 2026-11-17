package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFocusedWithholdsReviewerRejectedDraft(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer && (len(turn.Feedback) == 0 || !strings.HasPrefix(turn.Feedback[0], "FINAL")) {
			return AgentReply{Review: &AgentReview{Issues: []string{"Unsupported schema change."}}}, nil
		}
		return goodReply(turn)
	}}
	store := memorySessions{}
	result, err := (FocusedEngine{Model: model, Tools: fixtureTools{"source"}, Store: store}).Run(context.Background(), "withhold", agentBundle())
	if err != nil || result.Status != "NEEDS_REVIEW" || len(result.Draft) != 0 {
		t.Fatal("rejected draft promoted", result.Status, result.Draft, err)
	}
	child, err := store.Load(context.Background(), focusedChildID("withhold", 0, agentBundle().Changes[0].Candidates))
	if err != nil || len(child.Draft) == 0 {
		t.Fatal("lost diagnostic draft", err)
	}
	if !result.FocusedReview.OutcomeCountsAvailable || result.FocusedReview.BatchesApproved != 0 || result.FocusedReview.BatchesNeedingReview != 1 {
		t.Fatal("processed batch mislabeled as approved", result.FocusedReview)
	}
	if !strings.Contains(strings.Join(result.Feedback, " "), "withheld") {
		t.Fatal("missing diagnostic", result.Feedback)
	}
}

func TestFocusedIncludesEveryDisplayedCandidateAndResumesCandidateBatches(t *testing.T) {
	b := agentBundle()
	seed := b.Changes[0].Candidates[0]
	b.Changes[0].Candidates = nil
	for i := 0; i < 25; i++ {
		c := seed
		c.File = fmt.Sprintf("file%d.go", i)
		c.Evidence = []EvidenceItem{{ID: fmt.Sprintf("proof%d", i), Type: "TEXT", Location: c.File + ":1", Excerpt: "submit"}}
		b.Changes[0].Candidates = append(b.Changes[0].Candidates, c)
	}
	b.ContextID, _ = ComputeContextID(b)
	seen := map[string]int{}
	failed := false
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Analyst {
			if len(turn.Bundle.Changes[0].Candidates) > 10 {
				t.Fatal("unbounded focused batch")
			}
			file := turn.Bundle.Changes[0].Candidates[0].File
			if file == "file10.go" && !failed {
				failed = true
				return AgentReply{}, errors.New("disconnect")
			}
			for _, c := range turn.Bundle.Changes[0].Candidates {
				seen[c.File]++
			}
		}
		return goodReply(turn)
	}}
	e := FocusedEngine{Model: m, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	if first, err := e.Run(context.Background(), "top-ten", b); err == nil || first.Status != "PAUSED" {
		t.Fatal("expected resumable candidate batch")
	}
	done, err := e.Run(context.Background(), "top-ten", b)
	if err != nil || done.Status != "COMPLETE" || len(done.Draft) != 25 || done.FocusedReview.CandidatesProcessed != 25 || done.FocusedReview.BatchesReviewed != 3 {
		t.Fatal("lost top-ten coverage", done.Status, len(done.Draft), err)
	}
	for file, n := range seen {
		if n != 1 {
			t.Fatal("completed seed replayed", file, n)
		}
	}
	if len(seen) != 25 {
		t.Fatal("missed seed files")
	}
	if _, err := done.ChangePlan(); err != nil {
		t.Fatal(err)
	}
}

func TestFocusedSeedPayloadBudgetPreservesLargeAndEmptyInputs(t *testing.T) {
	large := Candidate{File: "large.go", Evidence: []EvidenceItem{{Excerpt: strings.Repeat("x", 12500)}}}
	next := Candidate{File: "next.go"}
	batches := focusedSeedBatches([]Candidate{large, next})
	if len(batches) != 2 || len(batches[0]) != 1 || batches[1][0].File != "next.go" {
		t.Fatal("oversize seed discarded or combined", batches)
	}
	if empty := focusedSeedBatches(nil); len(empty) != 1 || len(empty[0]) != 0 {
		t.Fatal("unseeded change lost discovery batch", empty)
	}
}

func TestFocusedContextSplittingPreservesAllSeedsAndResume(t *testing.T) {
	b := agentBundle()
	seed := b.Changes[0].Candidates[0]
	b.Changes[0].Candidates = nil
	for i := 0; i < 6; i++ {
		c := seed
		c.File = fmt.Sprintf("part%d.go", i)
		b.Changes[0].Candidates = append(b.Changes[0].Candidates, c)
	}
	b.ContextID, _ = ComputeContextID(b)
	seen := map[string]int{}
	oversizeCalls, interrupted := 0, false
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Analyst {
			candidates := turn.Bundle.Changes[0].Candidates
			if len(candidates) > 3 {
				oversizeCalls++
				return AgentReply{}, ErrAgentContextLimit
			}
			if candidates[0].File == "part3.go" && !interrupted {
				interrupted = true
				return AgentReply{}, errors.New("disconnect")
			}
			for _, c := range candidates {
				seen[c.File]++
			}
		}
		return goodReply(turn)
	}}
	e := FocusedEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	if first, err := e.Run(context.Background(), "context-split", b); err == nil || first.Status != "PAUSED" {
		t.Fatal(first.Status, err)
	}
	done, err := e.Run(context.Background(), "context-split", b)
	if err != nil || done.Status != "COMPLETE" || len(done.Draft) != 6 || done.FocusedReview.CandidatesProcessed != 6 || done.FocusedReview.BatchesReviewed != 2 || oversizeCalls != 1 {
		t.Fatal(done.Status, len(done.Draft), done.FocusedReview, oversizeCalls, err)
	}
	for file, n := range seen {
		if n != 1 {
			t.Fatal("seed repeated", file, n)
		}
	}
	if len(seen) != 6 {
		t.Fatal("seed omitted", seen)
	}
}

func TestFocusedSplitsPartitionBudgetWithoutRaisingChildBudget(t *testing.T) {
	b := agentBundle()
	second := b.Changes[0].Candidates[0]
	second.File = "another.go"
	b.Changes[0].Candidates = append(b.Changes[0].Candidates, second)
	b.ContextID, _ = ComputeContextID(b)
	failed := 0
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Analyst && len(turn.Bundle.Changes[0].Candidates) > 1 {
			failed++
			return AgentReply{}, errBatchReviewBudget
		}
		return goodReply(turn)
	}}
	store := memorySessions{}
	result, err := (FocusedEngine{Model: m, Tools: fixtureTools{"source"}, Store: store}).Run(context.Background(), "budget-split", b)
	if err != nil || result.Status != "COMPLETE" || len(result.Draft) != 2 || failed != 1 || result.FocusedReview.CandidatesProcessed != 2 {
		t.Fatal(result.Status, failed, err)
	}
	parent, err := store.Load(context.Background(), focusedChildID("budget-split", 0, b.Changes[0].Candidates))
	if err != nil || parent.Budget != DefaultAgentBudget() || !capacityLimitedChild(parent, nil) {
		t.Fatal("lost bounded failure checkpoint", parent, err)
	}
	if result.Turns <= parent.Turns {
		t.Fatal("failed work not charged", result.Turns, parent.Turns)
	}
}

func TestFocusedBatchesPreserveAllChangesAndResume(t *testing.T) {
	bundle := agentBundle()
	change := bundle.Changes[0]
	change.ElementID = "second"
	bundle.Changes = append(bundle.Changes, change)
	bundle.ContextID, _ = ComputeContextID(bundle)
	firstCalls := 0
	failed := false
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Analyst {
			if len(turn.Bundle.Changes) != 1 {
				t.Fatal("focused seed input was not split")
			}
			if turn.Bundle.Changes[0].ElementID == "second" && !failed {
				failed = true
				return AgentReply{}, errors.New("connection interrupted")
			}
			if turn.Bundle.Changes[0].ElementID != "second" {
				firstCalls++
			}
		}
		return goodReply(turn)
	}}
	engine := FocusedEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}}
	first, err := engine.Run(context.Background(), "focused-multi", bundle)
	if err == nil || first.Status != "PAUSED" {
		t.Fatal("interruption lost", err, first.Status)
	}
	before := firstCalls
	done, err := engine.Run(context.Background(), "focused-multi", bundle)
	if err != nil || done.Status != "COMPLETE" || len(done.Draft) != 2 || firstCalls != before {
		t.Fatal("focused resume lost coverage or replayed work", err, done.Status, len(done.Draft))
	}
	if _, err := done.ChangePlan(); err != nil {
		t.Fatal(err)
	}
}

func TestFocusedInvalidBatchDoesNotDiscardOtherChanges(t *testing.T) {
	bundle := agentBundle()
	second := bundle.Changes[0]
	second.ElementID = "second"
	bundle.Changes = append(bundle.Changes, second)
	bundle.ContextID, _ = ComputeContextID(bundle)
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Planner && turn.Bundle.Changes[0].ElementID != "second" {
			return AgentReply{Plan: &LayerPlanningResponse{}}, nil
		}
		return goodReply(turn)
	}}
	result, err := (FocusedEngine{Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}}).Run(context.Background(), "partial-focused", bundle)
	if err != nil || result.Status != "NEEDS_REVIEW" || len(result.Draft) != 1 || result.Draft[0].ElementID != "second" {
		t.Fatal("independent valid plan lost", result.Status, err, len(result.Draft))
	}
	plan, err := result.ChangePlan()
	if err != nil || len(plan.ChangeCoverage) != 2 || plan.ChangeCoverage[0].Status != "UNRESOLVED" || plan.ChangeCoverage[1].Status != "PLANNED" {
		t.Fatal("partial coverage misreported", plan.ChangeCoverage, err)
	}
}
