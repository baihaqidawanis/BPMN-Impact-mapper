package agenttools_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type resumableModel struct {
	calls   int
	reviews int
}

func (m *resumableModel) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "scripted", Model: "integration-fixture", PromptVersion: "agent-loop-v1"}
}
func (m *resumableModel) Turn(_ context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	m.calls++
	if m.calls == 1 {
		return planning.AgentReply{Tool: &planning.ToolRequest{Name: "read_function", ChangeIndex: 0, File: "submit.go", Symbol: "Submit"}}, nil
	}
	if m.calls == 2 {
		return planning.AgentReply{}, errors.New("simulated lost connection")
	}
	switch turn.Role {
	case planning.Analyst:
		var decisions []planning.CandidateDecision
		for _, c := range turn.Bundle.Changes[0].Candidates {
			decisions = append(decisions, planning.CandidateDecision{ChangeType: domain.AddTask, ElementID: "validate", File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictRelevant, Rationale: "Inspect submit validation before mutation.", EvidenceIDs: []string{c.Evidence[0].ID}})
		}
		return planning.AgentReply{Verification: &planning.VerificationResponse{Decisions: decisions}}, nil
	case planning.Planner:
		var plans []planning.FilePlan
		for _, c := range turn.Bundle.Changes[0].Candidates {
			plans = append(plans, planning.FilePlan{ChangeType: domain.AddTask, ElementID: "validate", File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: domain.ActionVerify, Summary: "Inspect submit guard", ProposedChanges: []string{"Require the new validation before submission while preserving unrelated actions."}, AcceptanceChecks: []string{"Invalid input is rejected before state mutation; valid input preserves the existing transition."}, Steps: []string{"Check validation before status mutation and rejection behavior."}, EvidenceIDs: []string{c.Evidence[0].ID}})
		}
		return planning.AgentReply{Plan: &planning.LayerPlanningResponse{Plans: plans}}, nil
	case planning.Reviewer:
		m.reviews++
		if m.reviews == 1 {
			return planning.AgentReply{Review: &planning.AgentReview{Issues: []string{"Explain rejection behavior."}, RevisionTarget: planning.Planner}}, nil
		}
		return planning.AgentReply{Review: &planning.AgentReview{Approved: true}}, nil
	}
	return planning.AgentReply{}, errors.New("unexpected role")
}

func TestSourceDiscoveryDiskResumeRevisionAndPlanExport(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	file := filepath.Join(root, "submit.go")
	original := []byte("package fixture\nfunc Submit() { validate() }\nfunc validate() {}\n")
	if err := os.WriteFile(file, original, 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := agenttools.New(ctx, root, ingest.Options{})
	if err != nil {
		t.Fatal(err)
	}
	bundle := planning.EvidenceBundle{Version: planning.BundleVersion, AnalysisID: "analysis", InputFingerprint: "fingerprint", Changes: []planning.ChangeInput{{ChangeType: domain.AddTask, ElementID: "validate", Candidates: []planning.Candidate{}}}}
	bundle.ContextID, _ = planning.ComputeContextID(bundle)
	store := filestore.AgentSessionStore{Root: t.TempDir()}
	model := &resumableModel{}
	engine := planning.AgentEngine{Model: model, Tools: tools, Store: store}
	paused, err := engine.Run(ctx, "integration", bundle)
	if err == nil || paused.Status != "PAUSED" {
		t.Fatalf("expected interruption: %+v %v", paused, err)
	}
	checkpoint, err := store.Load(ctx, "integration")
	if err != nil || checkpoint.Tools != 1 || len(checkpoint.Bundle.Changes[0].Candidates) != 1 {
		t.Fatalf("discovery not saved: %v", err)
	}
	// Recreate both store and engine as a restarted process would, retaining the
	// provider identity and snapshot. The failed model attempt stays charged.
	restarted := planning.AgentEngine{Model: model, Tools: tools, Store: filestore.AgentSessionStore{Root: store.Root}}
	done, err := restarted.Run(ctx, "integration", bundle)
	if err != nil || done.Status != "COMPLETE" || done.Revisions != 1 || done.Tools != 1 || done.Turns != 7 {
		t.Fatalf("resume/revision failed: %+v %v", done, err)
	}
	plan, err := done.ChangePlan()
	if err != nil || plan.PlanID == "" || len(plan.Plans) != 1 {
		t.Fatalf("canonical export failed: %v", err)
	}
	if len(planning.Markdown(plan)) == 0 {
		t.Fatal("empty rendered plan")
	}
	actual, err := os.ReadFile(file)
	if err != nil || string(actual) != string(original) {
		t.Fatal("analyzed source changed")
	}
}
