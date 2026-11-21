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
