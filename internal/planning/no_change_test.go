package planning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func noChangeFixture() FilePlan {
	return FilePlan{ChangeType: domain.AddTask, ElementID: "task", File: "store.go", Symbol: "Submit", Module: "core", Layer: "application", Action: domain.ActionNoAction, Summary: "Existing input contract is adequate for this file.", CurrentBehavior: "This layer forwards the existing input unchanged.", TargetBehavior: "The input remains available to the action-specific validator.", DesignRationale: "The inspected forwarding contract already supports that responsibility; validation changes belong to another verified target.", EvidenceIDs: []string{"proof"}, AcceptanceChecks: []string{"The existing input is forwarded without dropping its value."}}
}

func TestRelevantTargetMayBeReviewedWithoutInventingEdits(t *testing.T) {
	b := agentBundle()
	d, _ := goodReply(AgentTurn{Role: Analyst, Bundle: b})
	p := noChangeFixture()
	s := AgentSession{Role: Planner, Bundle: b, Decisions: d.Verification.Decisions}
	if err := (AgentEngine{}).apply(context.Background(), &s, AgentReply{Plan: &LayerPlanningResponse{Plans: []FilePlan{p}}}, b); err != nil {
		t.Fatal("forced invented changes", err)
	}
	if s.Role != Reviewer || s.Status == "COMPLETE" {
		t.Fatal("no-change bypassed review", s)
	}
	for _, bad := range []FilePlan{
		func() FilePlan { v := p; v.ProposedChanges = []string{"Add validation"}; return v }(),
		func() FilePlan { v := p; v.Steps = []string{"Change signature"}; return v }(),
		func() FilePlan { v := p; v.DesignRationale = ""; return v }(),
		func() FilePlan { v := p; v.AcceptanceChecks = nil; return v }(),
	} {
		if err := validatePlans(b, d.Verification.Decisions, "", []FilePlan{bad}); !errors.Is(err, ErrAgentPlanIncomplete) {
			t.Fatal("unsupported no-change accepted", err)
		}
	}
	coverage := summarizeChangeCoverage(b, d.Verification.Decisions, []FilePlan{p})
	if coverage[0].Plans != 0 || coverage[0].Status != "NO_CHANGE_PROPOSED" {
		t.Fatal("no-change counted as an edit", coverage)
	}
	md := string(Markdown(ChangePlan{Plans: []FilePlan{p}}))
	if !strings.Contains(md, "No source edits are proposed") || strings.Contains(md, "#### Implementation steps") {
		t.Fatal("misleading no-change export", md)
	}
	if Summarize(ChangePlan{Plans: []FilePlan{p}}).Files != 0 {
		t.Fatal("no-change assessment counted as an edit")
	}
	edit := p
	edit.Action = domain.ActionVerify
	if len(ProposedFiles([]FilePlan{p, edit, edit})) != 1 {
		t.Fatal("proposed file set duplicated targets")
	}
}
