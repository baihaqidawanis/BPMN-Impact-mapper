package planning

import (
	"context"
	"strings"
	"testing"
)

func TestReviewFindingsSurviveExportAndAreInvalidatedByNewDraft(t *testing.T) {
	b := agentBundle()
	verified, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	s := AgentSession{Bundle: b, Role: Planner, Decisions: verified.Verification.Decisions, LastReview: &AgentReview{Approved: true, Findings: []FileReviewFinding{{File: "old.go"}}}}
	planned, _ := goodReply(AgentTurn{Role: Planner, Bundle: b})
	e := AgentEngine{}
	if err := e.apply(context.Background(), &s, planned, b); err != nil {
		t.Fatal(err)
	}
	if s.LastReview != nil {
		t.Fatal("new draft kept stale review findings")
	}
	review := &AgentReview{Approved: true, Findings: []FileReviewFinding{{File: "store.go", ElementID: "task", Basis: "source_and_proposal", EvidenceIDs: []string{"proof"}, Checks: []ReviewCriterion{{Criterion: "contracts", Finding: "Existing signature is retained.", Status: "PASS"}}}}}
	if err := e.apply(context.Background(), &s, AgentReply{Review: review}, b); err != nil {
		t.Fatal(err)
	}
	p, err := s.ChangePlan()
	if err != nil || len(p.ReviewFindings) != 1 || p.ReviewFindings[0].File != "store.go" {
		t.Fatal(p, err)
	}
	md := string(Markdown(p))
	if strings.Contains(md, "Existing signature is retained.") || !strings.Contains(md, "JSON export") {
		t.Fatal("review conclusions missing from Markdown", md)
	}
	p.ReviewFindings[0].SourceExcerpt = "return `literal` <input>"
	p.ReviewFindings[0].ImplementationExcerpt = "Preserve `literal` and change only dispatch."
	archive, err := JSON(p)
	if err != nil || !strings.Contains(string(archive), "Existing signature is retained.") || !strings.Contains(string(archive), "Preserve `literal`") {
		t.Fatal("full review receipts missing from JSON")
	}
	p.ReviewFindings[0].Checks = append(p.ReviewFindings[0].Checks, ReviewCriterion{Criterion: "scope", Status: "REVISE", Finding: "Unrelated file."})
	if !strings.Contains(string(Markdown(p)), "Unrelated file.") {
		t.Fatal("open finding omitted")
	}
	s.Role = Analyst
	if err := e.apply(context.Background(), &s, verified, b); err != nil || s.LastReview != nil {
		t.Fatal("new decisions kept stale findings", err)
	}
}

func TestMarkdownRetainsAllFailingReviewChecksAndInlineFileConflicts(t *testing.T) {
	plan := ChangePlan{
		Status: StatusNeedsReview,
		Plans: []FilePlan{{
			ChangeType:       "ADD_GATEWAY",
			ElementID:        "ZeroDeltaDecision",
			File:             "backend/store.go",
			Symbol:           "Store.TransitionAdjustment",
			Action:           "MODIFY",
			Summary:          "Persist zero-delta status.",
			ProposedChanges:  []string{"Set NO_ADJUSTMENT when all lines have zero delta."},
			AcceptanceChecks: []string{"Zero delta sets NO_ADJUSTMENT."},
		}},
		ReviewFindings: []FileReviewFinding{{
			File: "backend/store.go",
			Checks: []ReviewCriterion{
				{Criterion: "requirement", Status: "REVISE", Finding: "Missing per-line loop."},
				{Criterion: "contracts", Status: "REVISE", Finding: "CHECK constraint blocks new status."},
				{Criterion: "scope", Status: "PASS", Finding: "Correct file."},
				{Criterion: "failure_tests", Status: "REVISE", Finding: "Missing opposite nonzero test."},
			},
		}},
		ReviewIssues: []string{
			"Plan 0: requirement: Missing per-line loop.; contracts: CHECK constraint blocks new status.; failure_tests: Missing opposite nonzero test.",
			"Cross-proposal conflict on backend/store.go: different BPMN change proposals assign conflicting zero-difference outcomes: NO_ADJUSTMENT (CheckZeroDelta) vs SUBMITTED (SkipOutcomeJoin).",
			"Schema prerequisite missing: backend/migrations/000005_operations.up.sql restricts status via CHECK(...) without \"NO_ADJUSTMENT\", yet backend/store.go proposes using \"NO_ADJUSTMENT\" without specifying a new forward migration to update the constraint.",
		},
	}
	md := string(Markdown(plan))
	for _, want := range []string{
		"Missing per-line loop.",
		"CHECK constraint blocks new status.",
		"Missing opposite nonzero test.",
		"> **Review warning:** Cross-proposal conflict on backend/store.go",
		"> **Review warning:** Schema prerequisite missing:",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in rendered Markdown:\n%s", want, md)
		}
	}
}
