package planning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestNumberedMigrationIsEvidenceWithoutEditingHistory(t *testing.T) {
	file := "backend/migrations/000005_operations.up.sql"
	for _, action := range []domain.RequiredAction{domain.ActionModify, domain.ActionRemove} {
		err := validateMigrationEdit(FilePlan{File: file, Action: action})
		if !errors.Is(err, ErrAgentPlanIncomplete) || !strings.Contains(err.Error(), "new forward migration") {
			t.Fatalf("expected actionable repair for %s: %v", action, err)
		}
	}
	for _, plan := range []FilePlan{
		{File: file, Action: domain.ActionNoAction},
		{File: "backend/migrations/000006_next.up.sql", Action: domain.ActionAdd},
		{File: "backend/schema.sql", Action: domain.ActionModify},
	} {
		if err := validateMigrationEdit(plan); err != nil {
			t.Fatalf("rejected %s for %s: %v", plan.Action, plan.File, err)
		}
	}
}

func TestRoutingGuardRejectsExplicitBusinessContradictions(t *testing.T) {
	change := ChangeInput{BranchChanges: []domain.BranchChange{{NewCondition: "semua selisih nol"}}}
	for _, step := range []string{
		"Calculate the sum of deltas for all lines before choosing the zero path.",
		"If the calculated delta sum is zero, set terminal status.",
		"Return success without writing the new status.",
		"Set d.Status = \"NO_ADJUSTMENT\" and return immediately without executing the stock mutation logic.",
	} {
		plan := FilePlan{File: "store.go", Steps: []string{step}}
		if err := validateRoutingPlan(change, plan); !errors.Is(err, ErrAgentPlanIncomplete) {
			t.Errorf("accepted conflicting step %q: %v", step, err)
		}
	}
	for _, step := range []string{
		"Compare the snapshot and counted quantity for every line, then persist the selected status and commit.",
		"Return HTTP 409 without writing the new status when the version is stale.",
		"Set d.Status = \"NO_ADJUSTMENT\", persist it, and return immediately after commit.",
		"Use a count of nonzero differences to select the zero path.",
	} {
		plan := FilePlan{File: "store.go", ProposedChanges: []string{step}}
		if err := validateRoutingPlan(change, plan); err != nil {
			t.Errorf("rejected valid step %q: %v", step, err)
		}
	}
	aggregate := ChangeInput{BranchChanges: []domain.BranchChange{{NewCondition: "net balance == 0"}}}
	if err := validateRoutingPlan(aggregate, FilePlan{Steps: []string{"Calculate the sum of deltas."}}); err != nil {
		t.Fatal("a true aggregate requirement may use an aggregate", err)
	}
}

func TestMisplacedDiagramRecipeIsRejectedWithoutBanningDocumentation(t *testing.T) {
	bad := FilePlan{File: "docs/API-CONTRACT.md", Steps: []string{"Identify the current BPMN process.", "Add a new user task ReasonReview.", "Update the flow connections."}, ProposedChanges: []string{"Insert a new user task in the BPMN process."}}
	if err := validatePlanScope(bad); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("uploaded failed plan was accepted", err)
	}
	gateway := FilePlan{File: "docs/api.md", ElementID: "Join", Steps: []string{"Insert the new gateway element into the process definition."}, ProposedChanges: []string{"Add a new BPMN element of type exclusiveGateway."}}
	if !errors.Is(validatePlanScope(gateway), ErrAgentPlanIncomplete) {
		t.Fatal("XML gateway edits accepted for docs")
	}
	md := string(Markdown(ChangePlan{Status: StatusNeedsReview, Plans: []FilePlan{gateway}}))
	if !strings.Contains(md, "still needs supporting evidence") || strings.Contains(md, "Add a new BPMN element") || strings.Contains(md, "NEEDS_REVIEW") {
		t.Fatal(md)
	}
	for _, p := range []FilePlan{
		{File: "service.go", Steps: []string{"Add validation in Submit before state mutation to implement the new BPMN task."}},
		{File: "docs/API-CONTRACT.md", Steps: []string{"Document the reason request field and invalid-input response for submit."}},
		{File: "diagram.bpmn", Steps: bad.Steps},
	} {
		if err := validatePlanScope(p); err != nil {
			t.Fatal("valid proposal rejected", err)
		}
	}
}

func TestDocumentationDoesNotHideUnresolvedImplementationCoverage(t *testing.T) {
	b := agentBundle()
	p := FilePlan{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: "docs/api.md", Layer: "documentation"}
	issues := implementationCoverageIssues(b, []FilePlan{p})
	if len(issues) != 1 || !strings.Contains(issues[0], "implementation candidates remain unresolved") {
		t.Fatal(issues)
	}
	code := p
	code.File = "service.go"
	code.Layer = "application"
	if issues := implementationCoverageIssues(b, []FilePlan{p, code}); len(issues) != 0 {
		t.Fatal(issues)
	}
	if issues := implementationCoverageIssues(b, nil); len(issues) != 0 {
		t.Fatal("invented a documentation proposal", issues)
	}
}

func TestInvalidOutputFeedbackIsRoleSpecific(t *testing.T) {
	analyst := modelOutputFeedback(Analyst, errors.New("unknown target index"))
	if !strings.Contains(analyst, "unknown target index") || !strings.Contains(analyst, "verification.decisions") || strings.Contains(analyst, "current_behavior") {
		t.Fatal(analyst)
	}
	if !strings.Contains(modelOutputFeedback(Planner, errors.New("invalid structured action")), "plan.plans") {
		t.Fatal("planner format absent")
	}
	if !strings.Contains(modelOutputFeedback(Reviewer, errors.New("empty response")), "review action") {
		t.Fatal("review format absent")
	}
}

func TestAgentDoesNotExportPlanForWeakTarget(t *testing.T) {
	b := agentBundle()
	decision := CandidateDecision{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: "store.go", Symbol: "Submit", Verdict: VerdictWeak, EvidenceIDs: []string{"proof"}}
	turn := AgentTurn{Role: Planner, Bundle: b}
	reply, _ := goodReply(turn)
	s := AgentSession{Role: Planner, Bundle: b, Decisions: []CandidateDecision{decision}}
	if err := (AgentEngine{}).apply(context.Background(), &s, reply, b); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("uncertain plan exported", err)
	}
}
