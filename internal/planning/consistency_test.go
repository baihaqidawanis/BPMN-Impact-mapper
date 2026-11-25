package planning

import (
	"bytes"
	"context"
	"errors"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strings"
	"testing"
)

func TestFinalConsistencyRejectsCrossBatchConflict(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer && len(turn.Feedback) > 0 && strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") {
			if len(turn.Draft) != 8 {
				t.Fatal("review did not receive all batches", len(turn.Draft))
			}
			return AgentReply{Review: &AgentReview{Issues: []string{"Batch one requires a reason but another proposal permits an empty reason."}}}, nil
		}
		return goodReply(turn)
	}}
	result, err := (FullRepositoryEngine{Model: model, Source: fullFixture(), Store: memorySessions{}}).Run(context.Background(), "conflict", agentBundle())
	if err != nil || result.Status != "NEEDS_REVIEW" || result.FullReview.ConsistencyReview != "needs_review" || len(result.Feedback) != 1 {
		t.Fatal(result.Status, result.Feedback, err)
	}
}
func TestFinalConsistencyResumeDoesNotRepeatScanOrPlanning(t *testing.T) {
	scan, planner, final := 0, 0, 0
	store := memorySessions{}
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.RepositoryChunk != nil {
			scan++
		}
		if turn.Role == Planner {
			planner++
		}
		if turn.Role == Reviewer && len(turn.Feedback) > 0 && strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") {
			final++
			if final == 1 {
				return AgentReply{}, errors.New("connection interrupted")
			}
		}
		return goodReply(turn)
	}}
	engine := FullRepositoryEngine{Model: model, Source: fullFixture(), Store: store}
	first, err := engine.Run(context.Background(), "resume-final", agentBundle())
	if err == nil || first.Status != "PAUSED" {
		t.Fatal("expected interrupted final review", err)
	}
	scanBefore, planBefore := scan, planner
	result, err := engine.Run(context.Background(), "resume-final", agentBundle())
	if err != nil || result.Status != "COMPLETE" || final != 2 || scan != scanBefore || planner != planBefore {
		t.Fatal("completed work repeated", result.Status, err, scan, planner, final)
	}
	if _, err = engine.Run(context.Background(), "resume-final", agentBundle()); err != nil || final != 2 {
		t.Fatal("final replay called model", err)
	}
}
func TestFinalConsistencyInvalidOutputHasBoundedRetries(t *testing.T) {
	count := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer && len(turn.Feedback) > 0 && strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") {
			count++
			return AgentReply{}, nil
		}
		return goodReply(turn)
	}}
	source := fullFixture()
	source.chunks = source.chunks[:1]
	engine := FullRepositoryEngine{Model: model, Source: source, Store: memorySessions{}}
	var result AgentSession
	for i := 0; i < 4; i++ {
		result, _ = engine.Run(context.Background(), "invalid-final", agentBundle())
	}
	if count != 3 || result.Status != "COMPLETE" {
		t.Fatal(count, result.Status, result.Feedback)
	}
}
func TestGroupedMarkdownRetainsChangesEvidenceAndUnresolvedCoverage(t *testing.T) {
	plan := ChangePlan{Status: StatusNeedsReview, Plans: []FilePlan{
		{File: "store.go", ElementID: "A", ChangeType: domain.AddTask, Summary: "Capture reason", CurrentBehavior: "Submit omits reason.", TargetBehavior: "Require reason before submit.", DesignRationale: "Reuse existing input.", Assumptions: []string{"Confirm error response contract."}, Steps: []string{"Validate before submit"}, EvidenceIDs: []string{"E1"}},
		{File: "other.go", ElementID: "B", ChangeType: domain.AddGateway, Summary: "Check active warehouse", EvidenceIDs: []string{"E2"}},
		{File: "store.go", ElementID: "C", ChangeType: domain.AddGateway, Summary: "Reject invalid reason", EvidenceIDs: []string{"E3"}},
	}, ChangeCoverage: []ChangeCoverage{{ElementID: "D", ChangeType: domain.ModifySequenceFlow, Status: "UNRESOLVED"}}}
	first := Markdown(plan)
	second := Markdown(plan)
	if !bytes.Equal(first, second) || strings.Count(string(first), ". `store.go`") != 1 {
		t.Fatal("file grouping is not stable")
	}
	for _, text := range []string{"Capture reason", "Reject invalid reason", "E1", "E3", "The affected code still needs to be identified.", "Validate before submit", "Submit omits reason.", "Require reason before submit.", "Confirm error response contract."} {
		if !strings.Contains(string(first), text) {
			t.Fatal("lost plan detail", text)
		}
	}
}

func TestConsistencyKeepsOtherFileFindingsWhenOneReviewExhaustsRetries(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") || turn.Draft[0].File == "a.go" {
			return AgentReply{}, ErrAgentModelOutput
		}
		return AgentReply{Review: &AgentReview{
			Issues:   []string{"Incorrect request field in b.go."},
			Findings: []FileReviewFinding{{File: "b.go"}},
		}}, nil
	}}
	engine := FullRepositoryEngine{Model: model, Store: memorySessions{}}
	s := AgentSession{Bundle: agentBundle(), Draft: []FilePlan{{File: "a.go"}, {File: "b.go"}}}
	review, _, err := engine.reviewCombinedPlan(context.Background(), "partial-review", s)
	if err != nil || review.Approved || len(review.Issues) < 2 || len(review.Findings) != 1 || review.Findings[0].File != "b.go" {
		t.Fatalf("expected an unresolved a.go and a retained b.go finding, got %+v, %v", review, err)
	}
}

func TestPartitionedConsistencyResumesWithoutRepeatingCompletedParts(t *testing.T) {
	calls := map[string]int{}
	fail := true
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		instruction := turn.Feedback[0]
		switch {
		case strings.HasPrefix(instruction, "FINAL CROSS-BATCH"):
			calls["whole"]++
			return AgentReply{}, ErrAgentContextLimit
		case strings.HasPrefix(instruction, "FINAL FILE"):
			file := turn.Draft[0].File
			calls[file]++
			for _, p := range turn.Draft {
				if p.File != file {
					t.Fatal("mixed file group")
				}
			}
			if file == "b.go" && fail {
				fail = false
				return AgentReply{}, errors.New("disconnect")
			}
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		case strings.HasPrefix(instruction, "FINAL CONTRACT"):
			calls["contracts"]++
			if len(turn.Draft) != 3 {
				t.Fatal("lost contract proposal")
			}
			for _, p := range turn.Draft {
				if p.TargetBehavior == "" || len(p.ProposedChanges) == 0 || len(p.Assumptions) == 0 {
					t.Fatal("lost contract details")
				}
				if p.CurrentBehavior != "" || len(p.Steps) != 0 {
					t.Fatal("not a compact review")
				}
			}
			return AgentReply{Review: &AgentReview{Issues: []string{"Frontend and backend disagree about the submit payload."}}}, nil
		}
		t.Fatal("unexpected review stage")
		return AgentReply{}, nil
	}}
	s := AgentSession{Binding: "b", Bundle: agentBundle()}
	for _, file := range []string{"a.go", "a.go", "b.go"} {
		s.Draft = append(s.Draft, FilePlan{File: file, CurrentBehavior: "current", TargetBehavior: "required", ProposedChanges: []string{"reuse submit contract"}, Assumptions: []string{"check compatibility"}, Steps: []string{"detailed edit"}})
	}
	engine := FullRepositoryEngine{Model: model, Store: memorySessions{}}
	if _, _, err := engine.reviewCombinedPlan(context.Background(), "partition", s); err == nil {
		t.Fatal("expected interrupted file review")
	}
	r, _, err := engine.reviewCombinedPlan(context.Background(), "partition", s)
	if err != nil || r.Approved || len(r.Issues) != 1 {
		t.Fatal("lost cross-file conflict", r, err)
	}
	if calls["whole"] != 1 || calls["a.go"] != 1 || calls["b.go"] != 2 || calls["contracts"] != 1 {
		t.Fatal("replayed completed work", calls)
	}
	if s.Draft[0].CurrentBehavior != "current" {
		t.Fatal("mutated saved draft")
	}
}

func TestNoActionComparisonBasisSurvivesContractCompaction(t *testing.T) {
	checked := false
	m := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") {
			return AgentReply{}, ErrAgentContextLimit
		}
		if strings.HasPrefix(turn.Feedback[0], "FINAL CONTRACT") {
			checked = true
			if turn.Draft[0].CurrentBehavior != "Existing responsibility is already supported." || len(turn.Draft[0].Steps) > 0 {
				t.Fatal("no-action comparison basis stripped", turn.Draft)
			}
		}
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}
	s := AgentSession{Binding: "noaction-source", Bundle: agentBundle(), Draft: []FilePlan{{File: "store.go", Action: domain.ActionNoAction, CurrentBehavior: "Existing responsibility is already supported.", TargetBehavior: "Preserve that responsibility."}}}
	r, _, err := (FullRepositoryEngine{Model: m, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "noaction-contract", s)
	if err != nil || !r.Approved || !checked {
		t.Fatal(r, err, checked)
	}
}

func TestPartitionedConsistencyKeepsSourceFindingsAlongsideFinalContractChecks(t *testing.T) {
	for _, rejectFile := range []bool{false, true} {
		t.Run(map[bool]string{false: "approved", true: "needs-review"}[rejectFile], func(t *testing.T) {
			contracts := 0
			model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
				instruction := turn.Feedback[0]
				if strings.HasPrefix(instruction, "FINAL CROSS-BATCH") {
					return AgentReply{}, ErrAgentContextLimit
				}
				if strings.HasPrefix(instruction, "FINAL FILE") {
					file := turn.Draft[0].File
					r := AgentReview{Approved: true, Findings: []FileReviewFinding{{File: file, Basis: "source_and_proposal", EvidenceIDs: []string{"proof-" + file}, SourceExcerpt: "actual source", ImplementationExcerpt: "actual edit", Checks: []ReviewCriterion{{Criterion: "contracts", Finding: "Compared the source and edit.", Status: "PASS"}}}}}
					if rejectFile && file == "b.go" {
						r.Approved, r.RevisionTarget, r.Issues = false, Planner, []string{"Payload edit missing."}
						r.Findings[0].Checks[0].Status = "REVISE"
					}
					return AgentReply{Review: &r}, nil
				}
				contracts++
				return AgentReply{Review: &AgentReview{Approved: true, Findings: []FileReviewFinding{{File: "a.go", Basis: "proposal_only", Checks: []ReviewCriterion{{Criterion: "contracts", Finding: "Cross-file contract consistent.", Status: "PASS"}}}}}}, nil
			}}
			s := AgentSession{Binding: "source-findings", Bundle: agentBundle(), Draft: []FilePlan{{File: "a.go"}, {File: "b.go"}}}
			r, _, err := (FullRepositoryEngine{Model: model, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "retain", s)
			if err != nil || r.Approved == rejectFile || len(r.Findings) < 2 || r.Findings[0].SourceExcerpt != "actual source" || r.Findings[1].EvidenceIDs[0] != "proof-b.go" || r.Findings[1].ImplementationExcerpt != "actual edit" {
				t.Fatal("detailed review evidence lost", r, err)
			}
			if rejectFile && (contracts != 0 || r.RevisionTarget != Planner || len(r.Findings) != 2) {
				t.Fatal("failed source review replaced by contract approval", r)
			}
			if !rejectFile && (contracts != 1 || len(r.Findings) != 3 || r.Findings[2].Basis != "proposal_only") {
				t.Fatal("final contract audit missing", r)
			}
		})
	}
}

func TestPartitionedConsistencyDoesNotApproveFailedFileReview(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if strings.HasPrefix(turn.Feedback[0], "FINAL CROSS-BATCH") {
			return AgentReply{}, ErrAgentContextLimit
		}
		if !strings.HasPrefix(turn.Feedback[0], "FINAL FILE") {
			t.Fatal("contracts must not approve rejected detail")
		}
		return AgentReply{Review: &AgentReview{Issues: []string{"Guard changes unrelated actions."}}}, nil
	}}
	s := AgentSession{Binding: "b", Bundle: agentBundle(), Draft: []FilePlan{{File: "a.go"}}}
	r, _, err := (FullRepositoryEngine{Model: model, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "reject-file", s)
	if err != nil || r.Approved || len(r.Issues) != 1 {
		t.Fatal(r, err)
	}
}

func TestFullRepositoryWithholdsRejectedBatchDrafts(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Reviewer && (len(turn.Feedback) == 0 || !strings.HasPrefix(turn.Feedback[0], "FINAL")) {
			return AgentReply{Review: &AgentReview{Issues: []string{"Missing evidence for this proposed migration."}}}, nil
		}
		return goodReply(turn)
	}}
	source := fullFixture()
	source.chunks = source.chunks[:1]
	result, err := (FullRepositoryEngine{Model: model, Source: source, Store: memorySessions{}}).Run(context.Background(), "withhold-full", agentBundle())
	if err != nil || result.Status != "NEEDS_REVIEW" || len(result.Draft) != 0 {
		t.Fatal(result.Status, len(result.Draft), err)
	}
	if !strings.Contains(strings.Join(result.Feedback, " "), "withheld") {
		t.Fatal("missing rejection diagnostic", result.Feedback)
	}
}

func TestEmptyCombinedPlanNeverCallsModel(t *testing.T) {
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		t.Fatal("empty plan sent to model")
		return AgentReply{}, nil
	}}
	review, turns, err := (FullRepositoryEngine{Model: model, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "empty-final", AgentSession{Bundle: agentBundle()})
	if err != nil || review.Approved || turns != 0 || len(review.Issues) != 1 {
		t.Fatal(review, turns, err)
	}
}

func TestScopedConsistencyBundleNarrowsTargetReviewToSingleChange(t *testing.T) {
	b := agentBundle()
	b.RepositoryOverview = &RepositoryOverview{File: "README.md", Summary: strings.Repeat("r", 1200)}
	second := b.Changes[0]
	second.ElementID = "OtherElement"
	b.Changes = append(b.Changes, second)
	plan := FilePlan{
		ChangeType: b.Changes[0].ChangeType,
		ElementID:  b.Changes[0].ElementID,
		File:       b.Changes[0].Candidates[0].File,
		Symbol:     b.Changes[0].Candidates[0].Symbol,
	}
	scoped := scopedConsistencyBundle(b, []FilePlan{plan})
	if scoped.RepositoryOverview != nil {
		t.Fatal("expected RepositoryOverview stripped from single-target review bundle")
	}
	if len(scoped.Changes) != 1 || scoped.Changes[0].ElementID != b.Changes[0].ElementID {
		t.Fatalf("expected 1 scoped change, got %d (%+v)", len(scoped.Changes), scoped.Changes)
	}
	if len(scoped.Changes[0].Candidates) != 1 {
		t.Fatalf("expected matching candidate evidence preserved, got %d", len(scoped.Changes[0].Candidates))
	}
}

func TestConsistencyFallsBackToScopedBundleOnTargetContextLimit(t *testing.T) {
	b := agentBundle()
	b.RepositoryOverview = &RepositoryOverview{File: "README.md", Summary: strings.Repeat("r", 1200)}
	second := b.Changes[0]
	second.ElementID = "OtherElement"
	b.Changes = append(b.Changes, second)
	plans := []FilePlan{
		{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: "store.go", Symbol: "Submit", Steps: []string{"step 1"}, AcceptanceChecks: []string{"check 1"}},
		{ChangeType: second.ChangeType, ElementID: second.ElementID, File: "store.go", Symbol: "Submit", Steps: []string{"step 2"}, AcceptanceChecks: []string{"check 2"}},
	}
	scopedCalls := 0
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		instruction := turn.Feedback[0]
		if strings.HasPrefix(instruction, "FINAL CROSS-BATCH") || strings.HasPrefix(instruction, "FINAL FILE") {
			return AgentReply{}, ErrAgentContextLimit
		}
		if strings.HasPrefix(instruction, "FINAL TARGET SOURCE") {
			if len(turn.Bundle.Changes) > 1 {
				return AgentReply{}, ErrAgentContextLimit
			}
			scopedCalls++
			if turn.Bundle.RepositoryOverview != nil {
				t.Fatal("scoped fallback kept RepositoryOverview")
			}
			return AgentReply{Review: &AgentReview{Approved: true}}, nil
		}
		return AgentReply{Review: &AgentReview{Approved: true}}, nil
	}}
	review, _, err := (FullRepositoryEngine{Model: model, Store: memorySessions{}}).reviewCombinedPlan(context.Background(), "scoped-fallback", AgentSession{Bundle: b, Draft: plans})
	if err != nil || !review.Approved || scopedCalls != 2 {
		t.Fatalf("expected 2 scoped target approvals, got approved=%v scopedCalls=%d err=%v issues=%v", review.Approved, scopedCalls, err, review.Issues)
	}
}

func TestFocusedPropagatesHistoricalMigrationNotesToLaterStorageBatch(t *testing.T) {
	b := agentBundle()
	migCandidate := b.Changes[0].Candidates[0]
	migCandidate.File = "backend/migrations/000005_operations.up.sql"
	migCandidate.Symbol = "stock_adjustments"
	migCandidate.Layer = "schema"
	migCandidate.Evidence = []EvidenceItem{{ID: "mig-1", Type: "SOURCE_TOOL", Location: migCandidate.File + ":36", Excerpt: strings.Repeat("x", 6500)}}
	storeCandidate := b.Changes[0].Candidates[0]
	storeCandidate.Evidence = []EvidenceItem{{ID: "store-1", Type: "SOURCE_TOOL", Location: storeCandidate.File + ":120", Excerpt: strings.Repeat("y", 6500)}}
	b.Changes[0].Candidates = []Candidate{migCandidate, storeCandidate}
	b.ContextID, _ = ComputeContextID(b)

	sawMigrationContract := false
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Planner && len(turn.Bundle.Changes) == 1 && len(turn.Bundle.Changes[0].Candidates) == 1 {
			c := turn.Bundle.Changes[0].Candidates[0]
			if c.File == migCandidate.File {
				return AgentReply{Plan: &LayerPlanningResponse{Plans: []FilePlan{{
					ChangeType:       b.Changes[0].ChangeType,
					ElementID:        b.Changes[0].ElementID,
					File:             c.File,
					Symbol:           c.Symbol,
					Module:           c.Module,
					Layer:            c.Layer,
					Action:           domain.ActionNoAction,
					Summary:          "Historical migration is read-only; add forward migration for new status.",
					CurrentBehavior:  "CHECK restricts status values.",
					TargetBehavior:   "Preserve historical migration.",
					DesignRationale:  "Do not modify applied migrations.",
					AcceptanceChecks: []string{"Historical migration remains unchanged."},
					EvidenceIDs:      []string{"mig-1"},
				}}}}, nil
			}
			if c.File == storeCandidate.File {
				for _, rel := range turn.RelatedProposals {
					if rel.File == migCandidate.File {
						sawMigrationContract = true
					}
				}
			}
		}
		return goodReply(turn)
	}}
	_, err := (FocusedEngine{Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}}).Run(context.Background(), "mig-prop", b)
	if err != nil {
		t.Fatal(err)
	}
	if !sawMigrationContract {
		t.Fatal("expected later storage batch to receive historical migration contract via related_proposals")
	}
}

func TestPartitionedPlannerRetainsCompanionRelatedProposals(t *testing.T) {
	b := agentBundle()
	verified, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	companion := []ProposalContract{{
		ChangeType: domain.AddGateway,
		ElementID:  "ZeroDeltaDecision",
		File:           "backend/migrations/000005_operations.up.sql",
		Action:         domain.ActionNoAction,
		TargetBehavior: "Historical migration restricts status via CHECK; add forward migration.",
	}}
	sawCompanion := false
	model := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		if turn.Role == Planner {
			for _, rel := range turn.RelatedProposals {
				if rel.File == companion[0].File && rel.ElementID == companion[0].ElementID {
					sawCompanion = true
				}
			}
		}
		return goodReply(turn)
	}}
	engine := AgentEngine{Model: model, Tools: fixtureTools{"fixed"}, Store: memorySessions{}, RelatedProposals: companion}
	s := AgentSession{Version: "1", ID: "pp-rel", Binding: "b", Bundle: b, Role: Planner, Decisions: verified.Verification.Decisions, Budget: DefaultAgentBudget()}
	reply, err := engine.planOversizedBatch(context.Background(), &s)
	if err != nil || reply.Plan == nil || len(reply.Plan.Plans) != 1 || !sawCompanion {
		t.Fatalf("expected partitioned planner to forward companion related_proposals, got err=%v sawCompanion=%v", err, sawCompanion)
	}
}
