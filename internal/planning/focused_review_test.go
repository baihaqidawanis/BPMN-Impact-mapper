package planning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
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

type schemaFixtureTools struct {
	fixtureTools
	facts []SchemaFact
	heads []string
}

func (t schemaFixtureTools) SchemaFacts() []SchemaFact { return t.facts }
func (t schemaFixtureTools) MigrationHeads() []string  { return t.heads }

func TestPostBatchPolishFixesLedgerTableAndFallsBackSafely(t *testing.T) {
	pc := &domain.ProcessContext{
		Before: &domain.Graph{
			ProcessNames: map[string]string{"p1": "Submit stock adjustment"},
			Elements:     map[string]*domain.Element{"sub": {Name: "Submit as SUBMITTED"}},
		},
		After: &domain.Graph{
			ProcessNames: map[string]string{"p1": "Submit stock adjustment"},
			Elements: map[string]*domain.Element{
				"Zero":  {ID: "Zero", Type: "exclusiveGateway", Name: "Semua selisih nol?"},
				"Close": {ID: "Close", Type: "serviceTask", Name: "Tutup NO_ADJUSTMENT", Documentation: "Lewati antrean approval; posting stok tetap aksi terpisah."},
				"Sub":   {ID: "Sub", Type: "serviceTask", Name: "Submit SUBMITTED"},
			},
			Flows: map[string]*domain.Flow{
				"f1": {SourceRef: "Zero", TargetRef: "Close", Condition: "semua selisih nol"},
				"f2": {SourceRef: "Zero", TargetRef: "Sub", Condition: "ada baris dengan selisih nonnol"},
			},
		},
	}
	storeCand := Candidate{
		File:            "backend/internal/inventory/adapter/postgres/store.go",
		Module:          "backend",
		Layer:           "adapter",
		SuggestedAction: domain.ActionVerify,
		Evidence: []EvidenceItem{{
			ID:       "ev-store-1",
			Type:     "SOURCE_TOOL",
			Location: "backend/internal/inventory/adapter/postgres/store.go:120",
			Excerpt:  "func (s *Store) TransitionAdjustment(ctx context.Context) error",
		}},
	}
	bundle := EvidenceBundle{
		Version:          BundleVersion,
		AnalysisID:       "analysis-1",
		InputFingerprint: "fp-1",
		ProcessContext:   pc,
		Changes: []ChangeInput{{
			ChangeType: domain.AddGateway,
			ElementID:  "ZeroDeltaDecision",
			After:      &ElementInput{ID: "ZeroDeltaDecision", Type: "exclusiveGateway", Name: "Semua selisih nol?"},
			Candidates: []Candidate{storeCand},
		}},
	}
	bundle.ContextID, _ = ComputeContextID(bundle)
	dec := CandidateDecision{
		ChangeType:  domain.AddGateway,
		ElementID:   "ZeroDeltaDecision",
		File:        storeCand.File,
		Verdict:     VerdictRelevant,
		Rationale:   "Owns adjustment submit transition.",
		EvidenceIDs: []string{"ev-store-1"},
	}
	unpolished := FilePlan{
		ChangeType:       domain.AddGateway,
		ElementID:        "ZeroDeltaDecision",
		File:             storeCand.File,
		Module:           "backend",
		Layer:            "adapter",
		Action:           domain.ActionModify,
		Summary:          "Extend TransitionAdjustment to route zero-delta adjustments to NO_ADJUSTMENT.",
		CurrentBehavior:  "TransitionAdjustment validates DRAFT status and queries stock movements.",
		TargetBehavior:   "Compare counted_quantity vs snapshot_quantity per line; set status to NO_ADJUSTMENT when all deltas are zero.",
		DesignRationale:  "Need zero-delta check and schema migration.",
		ProposedChanges:  []string{"Query stock movements and set status = 'NO_ADJUSTMENT' if all deltas are zero."},
		Steps:            []string{"Create a new forward migration 000013_inventory_adjustment_status.up.sql altering inventory_adjustments status CHECK constraint to allow NO_ADJUSTMENT, and provide a rollback script."},
		AcceptanceChecks: []string{"All zero lines -> NO_ADJUSTMENT; opposing +2/-2 -> SUBMITTED; stale version -> 409."},
		EvidenceIDs:      []string{"ev-store-1"},
	}
	tools := schemaFixtureTools{
		fixtureTools: fixtureTools{"schema-polish"},
		heads:        []string{"backend/migrations/000012_sales_invoice_review.up.sql"},
		facts: []SchemaFact{
			{File: "backend/migrations/000005_operations.up.sql", Excerpt: "CREATE TABLE inventory_adjustments (id uuid PRIMARY KEY, status text NOT NULL CHECK(status IN('DRAFT','SUBMITTED','APPROVED','REJECTED')))"},
			{File: "backend/migrations/000005_operations.up.sql", Excerpt: "CREATE TABLE inventory_adjustment_lines (id uuid PRIMARY KEY, snapshot_quantity numeric NOT NULL, counted_quantity numeric NOT NULL)"},
			{File: "backend/migrations/000005_operations.up.sql", Excerpt: "CREATE TABLE stock_movements (id uuid PRIMARY KEY, quantity numeric NOT NULL CHECK(quantity<>0))"},
		},
	}

	// 1. Fallback safety: if model errors or flips MODIFY -> NO_ACTION, original 75/100 draft MUST remain intact.
	for name, badModel := range map[string]AgentModel{
		"timeout": scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
			return AgentReply{}, errors.New("network timeout")
		}},
		"action-flip": scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
			flipped := unpolished
			flipped.Action = domain.ActionNoAction
			flipped.ProposedChanges = nil
			flipped.Steps = nil
			return AgentReply{Plan: &LayerPlanningResponse{Plans: []FilePlan{flipped}}}, nil
		}},
	} {
		t.Run("fallback-"+name, func(t *testing.T) {
			s := AgentSession{Bundle: bundle, Decisions: []CandidateDecision{dec}, Draft: []FilePlan{unpolished}}
			e := FocusedEngine{UltraLight: true, Model: badModel, Tools: tools, Store: memorySessions{}}
			e.polishDeterministicPlanIssues(context.Background(), "polish-"+name, &s)
			if s.Draft[0].Action != domain.ActionModify || s.Draft[0].ProposedChanges[0] != unpolished.ProposedChanges[0] || s.Revisions != 0 {
				t.Fatalf("fallback failed to preserve original draft on %s: %+v", name, s.Draft[0])
			}
		})
	}

	// 2. Success path: model fixes stock_movements -> inventory_adjustment_lines and adds existing-row rollback strategy.
	goodPolishModel := scriptedAgent{func(turn AgentTurn) (AgentReply, error) {
		joinedFB := strings.Join(turn.Feedback, "\n")
		if turn.Role != Planner || !strings.Contains(joinedFB, "inventory_adjustment_lines") || !strings.Contains(joinedFB, "Rollback strategy incomplete") {
			t.Fatalf("expected Planner turn with line-table and rollback feedback, got role=%s fb=%s", turn.Role, joinedFB)
		}
		clean := unpolished
		clean.CurrentBehavior = "TransitionAdjustment validates DRAFT status and version, then unconditionally sets status to SUBMITTED."
		clean.ProposedChanges = []string{"Query inventory_adjustment_lines to compare counted_quantity against snapshot_quantity per line, setting status = 'NO_ADJUSTMENT' when all deltas are zero."}
		clean.Steps = []string{"Create forward migration 000013_inventory_adjustment_status.up.sql altering inventory_adjustments status CHECK constraint to allow NO_ADJUSTMENT, with 000013_inventory_adjustment_status.down.sql blocking or remapping existing NO_ADJUSTMENT rows before restoring the prior CHECK constraint."}
		return AgentReply{Plan: &LayerPlanningResponse{Plans: []FilePlan{clean}}}, nil
	}}
	s := AgentSession{Bundle: bundle, Decisions: []CandidateDecision{dec}, Draft: []FilePlan{unpolished}}
	e := FocusedEngine{UltraLight: true, Model: goodPolishModel, Tools: tools, Store: memorySessions{}}
	e.polishDeterministicPlanIssues(context.Background(), "polish-ok", &s)
	if s.Revisions != 1 || !strings.Contains(s.Draft[0].ProposedChanges[0], "inventory_adjustment_lines") || strings.Contains(s.Draft[0].CurrentBehavior, "stock movements") {
		t.Fatalf("expected polished draft with inventory_adjustment_lines, got revisions=%d draft=%+v", s.Revisions, s.Draft[0])
	}
}
