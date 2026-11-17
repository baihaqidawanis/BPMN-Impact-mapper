package planning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMisplacedDiagramRecipeIsRejectedWithoutBanningDocumentation(t *testing.T) {
	bad := FilePlan{File: "docs/API-CONTRACT.md", Steps: []string{"Identify the current BPMN process.", "Add a new user task ReasonReview.", "Update the flow connections."}, ProposedChanges: []string{"Insert a new user task in the BPMN process."}}
	if err := validatePlanScope(bad); !errors.Is(err, ErrAgentPlanIncomplete) {
		t.Fatal("uploaded failed plan was accepted", err)
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
