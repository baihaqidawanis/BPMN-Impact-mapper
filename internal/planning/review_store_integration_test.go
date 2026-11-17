package planning_test

import (
	"context"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type storedReviewFixture struct{ failed bool }

func (*storedReviewFixture) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "fixture", Model: "fixture", PromptVersion: "store-review"}
}
func (*storedReviewFixture) PreferFileReview() bool { return true }
func (m *storedReviewFixture) Turn(_ context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	if turn.Role == planning.Planner {
		var plans []planning.FilePlan
		for _, c := range turn.Bundle.Changes[0].Candidates {
			plans = append(plans, planning.FilePlan{ChangeType: domain.AddTask, ElementID: "task", File: c.File, Module: "core", Layer: "application", Action: domain.ActionVerify, Summary: "Proposed scoped change", CurrentBehavior: "Existing input", TargetBehavior: "Validate input", DesignRationale: "New process requirement", ProposedChanges: []string{"Add a scoped guard"}, Steps: []string{"Validate before changing state"}, AcceptanceChecks: []string{"Invalid input preserves state"}, EvidenceIDs: []string{c.Evidence[0].ID}})
		}
		return planning.AgentReply{Plan: &planning.LayerPlanningResponse{Plans: plans}}, nil
	}
	if !m.failed {
		m.failed = true
		return planning.AgentReply{}, planning.ErrAgentModelOutput
	}
	return planning.AgentReply{Review: &planning.AgentReview{Approved: true}}, nil
}

type storedReviewTools struct{}

func (storedReviewTools) Identity() string { return "source-fixture" }
func (storedReviewTools) Execute(context.Context, planning.ToolRequest) (planning.ToolResult, error) {
	return planning.ToolResult{}, nil
}

func TestRelatedReviewRepairPersistsWithMaximumRootID(t *testing.T) {
	b := planning.EvidenceBundle{Version: planning.BundleVersion, AnalysisID: "analysis", InputFingerprint: "fingerprint", Changes: []planning.ChangeInput{{ChangeType: domain.AddTask, ElementID: "task"}}}
	var decisions []planning.CandidateDecision
	for _, file := range []string{"first.go", "second.go"} {
		b.Changes[0].Candidates = append(b.Changes[0].Candidates, planning.Candidate{File: file, Module: "core", Layer: "application", SuggestedAction: domain.ActionVerify, Evidence: []planning.EvidenceItem{{ID: file + "-proof", Excerpt: "input"}}})
		decisions = append(decisions, planning.CandidateDecision{ChangeType: domain.AddTask, ElementID: "task", File: file, Verdict: planning.VerdictRelevant, Rationale: "Affected input", EvidenceIDs: []string{file + "-proof"}})
	}
	b.ContextID, _ = planning.ComputeContextID(b)
	model := &storedReviewFixture{}
	engine := planning.AgentEngine{Model: model, Tools: storedReviewTools{}, Store: filestore.AgentSessionStore{Root: t.TempDir()}, InitialDecisions: decisions}
	id := strings.Repeat("a", 64)
	result, err := engine.Run(context.Background(), id, b)
	if err != nil || result.Status != "COMPLETE" || result.OutputRepairs != 1 {
		t.Fatal("compound related/feedback ID failed with real store", result.Status, result.Feedback, err)
	}
	if _, err := engine.Run(context.Background(), id, b); err != nil {
		t.Fatal("completed filesystem receipt could not be restored", err)
	}
}
