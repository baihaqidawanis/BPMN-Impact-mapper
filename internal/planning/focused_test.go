package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestUltraLightNewSplitUsesSpareLookupForCallers(t *testing.T) {
	bundle := agentBundle()
	change := &bundle.Changes[0]
	change.ChangeType = domain.AddGateway
	change.BranchChanges = []domain.BranchChange{{FlowID: "continue"}, {FlowID: "bypass"}}
	change.Candidates = []Candidate{
		{File: "backend/domain.go", Symbol: "Adjustment.Submit"},
		{File: "frontend/panel.tsx"},
		{File: "backend/operations_test.go"},
		{File: "backend/other.go", Symbol: "Other"},
	}
	decisions := []CandidateDecision{
		{File: "backend/domain.go", Symbol: "Adjustment.Submit", Verdict: VerdictRelevant},
		{File: "backend/other.go", Symbol: "Other", Verdict: VerdictWeak},
	}
	requests := weakSourceRequests(bundle, decisions, 4)
	if len(requests) != 4 {
		t.Fatalf("expected four lookups, got %+v", requests)
	}
	last := requests[3]
	if last.Name != "find_callers" || last.File != "backend/domain.go" || last.Symbol != "Adjustment.Submit" {
		t.Fatalf("expected the named Go candidate's caller lookup, got %+v", last)
	}
}

func TestWeakSourceInspectionUsesFlowGuardAndDestination(t *testing.T) {
	bundle := agentBundle()
	c := &bundle.Changes[0]
	c.ElementID = "f"
	c.ElementName = ""
	c.Candidates = []Candidate{{File: "backend/review.go", Module: "backend"}}
	g := &domain.Graph{
		ProcessNames: map[string]string{"P": "Claim approval"},
		Elements: map[string]*domain.Element{
			"x": {Name: "Choose route", ProcessID: "P"},
			"y": {Name: "Return to owner", ProcessID: "P"},
		},
		Flows: map[string]*domain.Flow{"f": {SourceRef: "x", TargetRef: "y", Condition: "amount exceeds limit"}},
	}
	for _, pc := range []*domain.ProcessContext{{After: g}, {Before: g}} {
		bundle.ProcessContext = pc
		requests := weakSourceRequests(bundle, nil, 1)
		if len(requests) != 1 {
			t.Fatal(requests)
		}
		for _, term := range []string{"amount exceeds limit", "Return to owner", "Claim approval"} {
			if !strings.Contains(requests[0].Query, term) {
				t.Fatalf("lost %q from %q", term, requests[0].Query)
			}
		}
	}
}

func TestUltraLightKeepsBackendTestAndForwardSchemaEvidence(t *testing.T) {
	bundle := agentBundle()
	change := &bundle.Changes[0]
	change.ElementName = "Submit stock adjustment"
	change.Candidates = []Candidate{
		{File: "frontend/first.tsx"}, {File: "frontend/second.tsx"}, {File: "frontend/third.tsx"},
		{File: "frontend/tests/stock.test.tsx"}, {File: "backend/adjustment/store.go"},
		{File: "backend/migrations/001_operations.down.sql"}, {File: "backend/test/integration/operations_test.go"},
		{File: "backend/migrations/001_operations.up.sql"},
	}
	engine := FocusedEngine{UltraLight: true}
	for _, rollbackOnly := range []bool{false, true} {
		if rollbackOnly {
			change.Candidates = change.Candidates[:len(change.Candidates)-1]
		}
		initial, _ := engine.seedCandidates(change.Candidates)
		input := bundle
		input.Changes = []ChangeInput{*change}
		input.Changes[0].Candidates = initial
		reads := weakSourceRequests(input, nil, 4)
		files := make(map[string]bool, len(reads))
		for _, read := range reads {
			files[read.File] = true
			if strings.HasSuffix(read.File, ".up.sql") && !strings.Contains(read.Query, "constraint check status") {
				t.Fatalf("schema inspection lost its constraint query: %+v", read)
			}
		}
		for _, want := range []string{"backend/adjustment/store.go", "backend/test/integration/operations_test.go", "backend/migrations/001_operations.up.sql"} {
			if !files[want] {
				t.Fatalf("missing %s in %+v (rollback only: %v)", want, reads, rollbackOnly)
			}
		}
		if files["backend/migrations/001_operations.down.sql"] || len(reads) > 4 {
			t.Fatalf("unexpected lookups: %+v", reads)
		}
	}
}

func TestFocusedRetainsCombinedReviewerRejectedDraftAsUnapproved(t *testing.T) {
	reviews := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer {
			reviews++
			if len(turn.Feedback) == 0 || !strings.HasPrefix(turn.Feedback[0], "FINAL") {
				t.Fatal("unexpected per-batch review")
			}
			return AgentReply{Review: &AgentReview{Issues: []string{"Unsupported schema change."}}}, nil
		}
		return goodReply(turn)
	}}
	store := memorySessions{}
	result, err := (FocusedEngine{Model: model, Tools: fixtureTools{"source"}, Store: store}).Run(context.Background(), "withhold", agentBundle())
	if err != nil || result.Status != "NEEDS_REVIEW" || len(result.Draft) != 1 || reviews != 1 {
		t.Fatal("rejected draft approved or lost", result.Status, result.Draft, err)
	}
	child, err := store.Load(context.Background(), focusedChildID("withhold", 0, agentBundle().Changes[0].Candidates))
	if err != nil || len(child.Draft) == 0 {
		t.Fatal("lost diagnostic draft", err)
	}
	if !result.FocusedReview.OutcomeCountsAvailable || result.FocusedReview.BatchesApproved != 0 || result.FocusedReview.BatchesPrepared != 1 || !child.AwaitingCombinedReview {
		t.Fatal("processed batch mislabeled as approved", result.FocusedReview)
	}
	if !strings.Contains(strings.Join(result.Feedback, " "), "Unsupported schema change") {
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

func TestLightDoesNotCountVerifyAsAnEdit(t *testing.T) {
	candidates := []Candidate{{File: "backend/store.go"}, {File: "backend/store_test.go"}}
	plans := []FilePlan{{File: candidates[0].File, Action: domain.ActionModify}, {File: candidates[1].File, Action: domain.ActionVerify}}
	if !lightNeedsMoreSource(candidates, plans) {
		t.Fatal("a test marked VERIFY was counted as a concrete edit")
	}
}

func TestUltraLightChecksOnlyWeakTestsWithSourceEvidence(t *testing.T) {
	const testFile = "backend/test/integration/operations_test.go"
	decisions := []CandidateDecision{
		{File: "backend/store.go", Verdict: VerdictRelevant},
		{File: testFile, Verdict: VerdictWeak},
	}
	observations := []ToolObservation{{
		Request: ToolRequest{Name: "inspect_source", File: testFile},
		Result:  ToolResult{Candidates: []Candidate{{File: testFile, Evidence: []EvidenceItem{{ID: "submit-call"}}}}},
	}}
	if !hasInspectedWeakTest(decisions, observations) {
		t.Fatal("inspected test with an affected implementation should get a second assessment")
	}
	observations[0].Result.Candidates[0].File = "backend/test/other_test.go"
	if hasInspectedWeakTest(decisions, observations) {
		t.Fatal("evidence from a different file triggered the test assessment")
	}
}

func TestUltraLightReassessesWeakImplementationAndTestTogether(t *testing.T) {
	bundle := agentBundle()
	base := bundle.Changes[0].Candidates[0]
	for _, file := range []string{"backend/helper.go", "backend/test/operations_test.go"} {
		c := base
		c.File = file
		bundle.Changes[0].Candidates = append(bundle.Changes[0].Candidates, c)
	}
	bundle.ContextID, _ = ComputeContextID(bundle)
	analystTurns := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		reply, err := goodReply(turn)
		if turn.Role == Analyst {
			analystTurns++
			if analystTurns == 1 {
				for i := range reply.Verification.Decisions {
					reply.Verification.Decisions[i].Verdict = VerdictWeak
				}
			} else if len(turn.Observations) != 2 || turn.Observations[0].Request.File != base.File || turn.Observations[1].Request.File != "backend/test/operations_test.go" {
				t.Fatalf("expected both targeted inspections before reassessment: %+v", turn.Observations)
			}
		}
		return reply, err
	}}
	engine := AgentEngine{InspectWeak: true, Model: model, Tools: fixtureTools{"source"}, Store: memorySessions{}, DeferReview: true, Budget: AgentBudget{MaxTools: 2, MaxTurns: 8, MaxDiscoveries: 2}}
	session, err := engine.Run(context.Background(), "weak-recheck", bundle)
	if err != nil || analystTurns != 2 || session.Tools != 2 {
		t.Fatalf("unexpected reassessment: turns=%d tools=%d err=%v", analystTurns, session.Tools, err)
	}
}

type emptyFocusedTools struct{}

func (emptyFocusedTools) Identity() string { return "empty focused source" }

func (emptyFocusedTools) Execute(context.Context, ToolRequest) (ToolResult, error) {
	return ToolResult{}, nil
}

func TestLightFocusedIncludesLowerRankedTestAndExpandsForMissingCoverage(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		weakTest, verifyOnly, ultra, expand bool
		wantProcessed                       int
	}{
		{name: "light-covered", wantProcessed: 4},
		{name: "light-expands", weakTest: true, expand: true, wantProcessed: 8},
		{name: "light-expands-verify-only", verifyOnly: true, expand: true, wantProcessed: 8},
		{name: "ultra-flags-verify-only", verifyOnly: true, ultra: true, expand: true, wantProcessed: 6},
		{name: "ultra-expands-only-missing-test", weakTest: true, ultra: true, expand: true, wantProcessed: 5},
		{name: "ultra-covered", ultra: true, wantProcessed: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bundle := agentBundle()
			files := []string{"frontend/panel.tsx", "backend/store.go", "docs/readme.md", "docs/api.md", "docs/dev.md", "scripts/util.py", "backend/operations_test.go", "backend/another_test.go"}
			seed := bundle.Changes[0].Candidates[0]
			if !tc.verifyOnly {
				seed.SuggestedAction = domain.ActionModify
			}
			bundle.Changes[0].Candidates = nil
			for _, file := range files {
				candidate := seed
				candidate.File = file
				candidate.Evidence = []EvidenceItem{{ID: file, Type: "TEXT", Location: file + ":1", Excerpt: "submit"}}
				bundle.Changes[0].Candidates = append(bundle.Changes[0].Candidates, candidate)
			}
			bundle.ContextID, _ = ComputeContextID(bundle)
			seen := map[string]bool{}
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				if tc.ultra && turn.Role == Analyst && turn.RemainingTools > 2 {
					t.Fatal("Ultra Light exceeded the lookup budget")
				}
				reply, err := goodReply(turn)
				if reply.Verification != nil {
					for i := range reply.Verification.Decisions {
						decision := &reply.Verification.Decisions[i]
						seen[decision.File] = true
						if tc.weakTest && decision.File == "backend/operations_test.go" {
							decision.Verdict = VerdictWeak
						}
					}
				}
				if reply.Plan != nil && !tc.verifyOnly {
					for i := range reply.Plan.Plans {
						reply.Plan.Plans[i].Action = domain.ActionModify
					}
				}
				if reply.Plan != nil && tc.weakTest {
					for i, plan := range reply.Plan.Plans {
						if plan.File == "backend/operations_test.go" {
							reply.Plan.Plans = append(reply.Plan.Plans[:i], reply.Plan.Plans[i+1:]...)
							break
						}
					}
				}
				return reply, err
			}}
			engine := FocusedEngine{Model: model, Tools: emptyFocusedTools{}, Store: memorySessions{}, Light: !tc.ultra, UltraLight: tc.ultra}
			result, err := engine.Run(context.Background(), "light-review", bundle)
			if err != nil || result.FocusedReview.UltraLight != tc.ultra || !seen["backend/operations_test.go"] {
				t.Fatalf("Light lost the test candidate: %v %+v", err, result.FocusedReview)
			}
			if seen["backend/another_test.go"] != tc.expand {
				t.Fatalf("Unexpected reserve coverage: %v", seen)
			}
			if tc.ultra && tc.verifyOnly && (!strings.Contains(strings.Join(result.Feedback, " "), "Ultra Light left source coverage unresolved") || result.Status != "NEEDS_REVIEW") {
				t.Fatalf("verify-only draft was cleared without warning: %+v", result)
			}
			if result.FocusedReview.CandidatesProcessed != tc.wantProcessed || result.FocusedReview.CandidatesTotal != len(files) {
				t.Fatalf("Invalid reported coverage: %+v", result.FocusedReview)
			}
		})
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
