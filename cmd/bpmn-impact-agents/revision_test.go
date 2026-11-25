package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestLockedRevisionUsesPriorEditsForOnlySelectedElement(t *testing.T) {
	source := planning.AgentSession{Status: "NEEDS_REVIEW"}
	for _, element := range []string{"submit", "approve"} {
		candidate := planning.Candidate{File: element + ".go", SuggestedAction: domain.ActionModify, Evidence: []planning.EvidenceItem{{ID: "source"}}}
		source.Bundle.Changes = append(source.Bundle.Changes, planning.ChangeInput{ChangeType: domain.AddTask, ElementID: element, Candidates: []planning.Candidate{candidate}})
		source.Decisions = append(source.Decisions, planning.CandidateDecision{ChangeType: domain.AddTask, ElementID: element, File: candidate.File, Verdict: planning.VerdictRelevant, EvidenceIDs: []string{"source"}})
		source.Draft = append(source.Draft, planning.FilePlan{ChangeType: domain.AddTask, ElementID: element, File: candidate.File, Action: domain.ActionModify, Summary: "Validate argument.", Steps: []string{"Add a guard."}, EvidenceIDs: []string{"source"}})
	}
	graph := &domain.Graph{
		Elements: map[string]*domain.Element{"submit": {ID: "submit"}, "result": {ID: "result"}, "approve": {ID: "approve"}},
		Flows:    map[string]*domain.Flow{"send": {ID: "send", SourceRef: "submit", TargetRef: "result"}},
	}
	source.Bundle.ProcessContext = &domain.ProcessContext{After: graph}
	result, err := lockPlan(source, "submit")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.draft) != 1 || len(result.decisions) != 1 || len(result.actions) != 1 || len(result.bundle.Changes) != 1 || result.draft[0].File != "submit.go" {
		t.Fatalf("revision changed selection: %+v", result)
	}
	if local := result.bundle.ProcessContext.After; local.Flows["send"] == nil || local.Elements["result"] == nil || local.Elements["approve"] != nil || graph.Elements["approve"] == nil || len(local.Outgoing["submit"]) != 1 || len(local.Incoming["result"]) != 1 {
		t.Fatal("revision lost a local edge or changed the archived graph")
	}
}

func TestLockedPlannerRejectsMissingFixtureReferenceAndBindsEvidence(t *testing.T) {
	const file = "tests/lifecycle_test.go"
	sites := []agenttools.TestCallSite{
		{Reference: "lifecycle_test.go:12", Evidence: planning.EvidenceItem{ID: "first"}},
		{Reference: "lifecycle_test.go:28", Evidence: planning.EvidenceItem{ID: "second"}},
	}
	plan := planning.FilePlan{File: file, Action: domain.ActionModify, Steps: []string{"Update lifecycle_test.go:12 and lifecycle_test.go:280."}, EvidenceIDs: []string{"first", "second"}}
	turn := planning.AgentTurn{Role: planning.Planner, Decisions: []planning.CandidateDecision{{File: file}}}
	model := lockedPlanner{actions: map[editTarget]domain.RequiredAction{{file: file}: domain.ActionModify}, testFile: file, callSites: sites}
	model.AgentModel = revisionFixture{plans: []planning.FilePlan{plan}}
	if _, err := model.Turn(context.Background(), turn); !errors.Is(err, planning.ErrAgentModelOutput) || !strings.Contains(err.Error(), "lifecycle_test.go:28") {
		t.Fatalf("missing call site should be rejected with specific feedback: %v", err)
	}
	plan.Steps = []string{"Update lifecycle_test.go:12 and lifecycle_test.go:28."}
	plan.EvidenceIDs = []string{"first"}
	model.AgentModel = revisionFixture{plans: []planning.FilePlan{plan}}
	result, err := model.Turn(context.Background(), turn)
	if err != nil {
		t.Fatalf("supported coverage should be accepted: %v", err)
	}
	if evidence := result.Plan.Plans[0].EvidenceIDs; len(evidence) != 2 || evidence[1] != "second" {
		t.Fatalf("host did not attach evidence for the second verified reference: %v", evidence)
	}
}

type revisionFixture struct {
	planning.AgentModel
	plans []planning.FilePlan
}

func (m revisionFixture) Turn(_ context.Context, _ planning.AgentTurn) (planning.AgentReply, error) {
	return planning.AgentReply{Plan: &planning.LayerPlanningResponse{Plans: m.plans}}, nil
}

func TestLockedPlannerRejectsDroppedOrDowngradedEdit(t *testing.T) {
	turn := planning.AgentTurn{Role: planning.Planner, Decisions: []planning.CandidateDecision{{File: "submit.go"}}}
	actions := map[editTarget]domain.RequiredAction{{file: "submit.go"}: domain.ActionModify}
	for _, tc := range []struct {
		name  string
		plans []planning.FilePlan
		valid bool
	}{
		{"retained", []planning.FilePlan{{File: "submit.go", Action: domain.ActionModify}}, true},
		{"dropped", nil, false},
		{"downgraded", []planning.FilePlan{{File: "submit.go", Action: domain.ActionVerify}}, false},
		{"substituted", []planning.FilePlan{{File: "approve.go", Action: domain.ActionModify}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := lockedPlanner{AgentModel: revisionFixture{plans: tc.plans}, actions: actions}
			_, err := model.Turn(context.Background(), turn)
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, planning.ErrAgentModelOutput) {
				t.Fatalf("unexpected validation result: %v", err)
			}
		})
	}
}

func TestLockedSourceWindowRejectsOtherFilesBeforeReading(t *testing.T) {
	bundle := planning.EvidenceBundle{Changes: []planning.ChangeInput{{Candidates: []planning.Candidate{{File: "submit.go"}}}}}
	// A nil tool dependency proves the rejected window never reaches Execute.
	if _, err := addLockedSourceWindows(context.Background(), nil, bundle, []string{"other.go:8"}, nil); err == nil {
		t.Fatal("source outside locked targets was accepted")
	}
}

func TestFocusedLockedEvidenceKeepsExplicitSourceAndArchivedBundle(t *testing.T) {
	bundle := planning.EvidenceBundle{Changes: []planning.ChangeInput{{Candidates: []planning.Candidate{
		{File: "submit.go", Evidence: []planning.EvidenceItem{{ID: "rank", Type: "BM25"}, {ID: "source", Type: "SOURCE_TOOL"}, {ID: "verified", Type: "AST_TEST_CALL"}}},
		{File: "ui.ts", Evidence: []planning.EvidenceItem{{ID: "payload"}}},
	}}}}
	got := focusLockedPlannerEvidence(bundle, map[string][]string{"submit.go": {"verified"}})
	evidence := got.Changes[0].Candidates[0].Evidence
	if len(evidence) != 2 || evidence[0].ID != "verified" || evidence[1].ID != "source" {
		t.Fatalf("wrong evidence projection: %+v", evidence)
	}
	if len(bundle.Changes[0].Candidates[0].Evidence) != 3 || got.Changes[0].Candidates[1].Evidence[0].ID != "payload" {
		t.Fatal("scope projection modified the archived bundle or unrelated file")
	}
}
