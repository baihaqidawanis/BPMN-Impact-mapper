package planning_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type fakeProvider struct {
	mode   string
	layers []string
}

func (provider *fakeProvider) Identity() planning.ProviderIdentity {
	if provider.mode == "empty-identity" {
		return planning.ProviderIdentity{}
	}
	return planning.ProviderIdentity{Provider: "fake-local", Model: "fixture-model", PromptVersion: "planning-v1"}
}
func (*fakeProvider) Enabled() bool { return true }

func (provider *fakeProvider) VerifyCandidates(ctx context.Context, request planning.VerificationRequest) (planning.VerificationResponse, error) {
	if provider.mode == "timeout" {
		<-ctx.Done()
		return planning.VerificationResponse{}, ctx.Err()
	}
	decisions := make([]planning.CandidateDecision, 0, len(request.Change.Candidates))
	for _, candidate := range request.Change.Candidates {
		evidenceID := candidate.Evidence[0].ID
		if provider.mode == "invent-evidence" {
			evidenceID = "MADE-UP"
		}
		decisions = append(decisions, planning.CandidateDecision{
			ChangeType: request.Change.ChangeType, ElementID: request.Change.ElementID,
			File: candidate.File, Symbol: candidate.Symbol, Verdict: planning.VerdictRelevant,
			Rationale: "The bounded evidence links this candidate to the changed task.", EvidenceIDs: []string{evidenceID},
		})
	}
	return planning.VerificationResponse{Decisions: decisions}, nil
}

func (provider *fakeProvider) PlanLayer(_ context.Context, request planning.LayerPlanningRequest) (planning.LayerPlanningResponse, error) {
	provider.layers = append(provider.layers, request.Layer)
	plans := make([]planning.FilePlan, 0, len(request.Decisions))
	for _, decision := range request.Decisions {
		if provider.mode == "omit-plan" {
			continue
		}
		candidate := findCandidate(request.Bundle, decision)
		file := candidate.File
		if provider.mode == "invent-file" {
			file = "invented/file.go"
		}
		action := candidate.SuggestedAction
		steps := []string{"Inspect the cited symbol and preserve its public contract."}
		if provider.mode == "escalate-action" {
			action = domain.ActionRemove
		}
		if provider.mode == "code-step" {
			steps = []string{"```go\nfunc invented() {}\n```"}
		}
		plans = append(plans, planning.FilePlan{
			ChangeType: decision.ChangeType, ElementID: decision.ElementID, File: file, Symbol: candidate.Symbol,
			Module: candidate.Module, Layer: candidate.Layer, Action: action,
			Summary: "Update the existing behavior without generating source code.",
			Steps:   steps, EvidenceIDs: append([]string(nil), decision.EvidenceIDs...),
		})
	}
	return planning.LayerPlanningResponse{Plans: plans}, nil
}

func (provider *fakeProvider) ReviewPlan(_ context.Context, request planning.ReviewRequest) (planning.ReviewResponse, error) {
	issues := []string{}
	if provider.mode == "approved-with-issues" {
		issues = []string{"A reviewer concern remains."}
	}
	return planning.ReviewResponse{Approved: true, Issues: issues, Revised: append([]planning.FilePlan(nil), request.Draft...)}, nil
}

func TestOrchestratorProducesDeterministicEvidenceGroundedPlan(t *testing.T) {
	provider := &fakeProvider{}
	orchestrator := planning.Orchestrator{Provider: provider, Limits: planning.DefaultLimits()}
	first, err := orchestrator.Run(context.Background(), fixtureReport())
	if err != nil {
		t.Fatal(err)
	}
	firstLayers := append([]string(nil), provider.layers...)
	provider.layers = nil
	second, err := orchestrator.Run(context.Background(), fixtureReport())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(firstLayers, []string{"application", "frontend"}) {
		t.Fatalf("non-deterministic plan or layer order: first=%#v second=%#v layers=%#v", first, second, firstLayers)
	}
	if first.Status != planning.StatusComplete || len(first.Plans) != 2 || len(first.PlanID) != 32 || len(first.ContextID) != 32 {
		t.Fatalf("plan=%#v", first)
	}
	jsonBytes, err := planning.JSON(first)
	if err != nil || !strings.Contains(string(jsonBytes), `"context_id"`) || !strings.Contains(string(jsonBytes), `"prompt_version"`) {
		t.Fatalf("json=%s err=%v", jsonBytes, err)
	}
	markdown := string(planning.Markdown(first))
	if !strings.Contains(markdown, "# BPMN change plan") || !strings.Contains(markdown, "internal/request.go") {
		t.Fatalf("markdown=%s", markdown)
	}
}

func TestOrchestratorRejectsInventedOrIncompleteOutput(t *testing.T) {
	for _, mode := range []string{"invent-file", "invent-evidence", "omit-plan", "escalate-action", "code-step"} {
		t.Run(mode, func(t *testing.T) {
			_, err := (planning.Orchestrator{Provider: &fakeProvider{mode: mode}}).Run(context.Background(), fixtureReport())
			if err == nil {
				t.Fatal("expected hallucination guard to reject provider output")
			}
		})
	}
}

func TestOrchestratorCannotCompleteWhileReviewIssuesRemain(t *testing.T) {
	plan, err := (planning.Orchestrator{Provider: &fakeProvider{mode: "approved-with-issues"}}).Run(context.Background(), fixtureReport())
	if err != nil || plan.Status != planning.StatusNeedsReview || len(plan.ReviewIssues) != 1 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
}

func TestEvidenceBundleAppliesHardLimits(t *testing.T) {
	report := fixtureReport()
	report.Changes[0].Impacts[1].Evidence = []domain.Evidence{
		{Type: "A", Location: "a:1", Excerpt: "123456789"},
		{Type: "B", Location: "a:2", Excerpt: "second"},
	}
	bundle := planning.BuildEvidenceBundle(report, planning.Limits{CandidatesPerChange: 1, EvidencePerCandidate: 1, ExcerptRunes: 5})
	if len(bundle.Changes[0].Candidates) != 1 || len(bundle.Changes[0].Candidates[0].Evidence) != 1 || bundle.Changes[0].Candidates[0].Evidence[0].Excerpt != "1234…" {
		t.Fatalf("bounded bundle=%#v", bundle)
	}
}

func TestAgentBundlePrefersDistinctSourceAndPreservesEvidenceIDs(t *testing.T) {
	report := fixtureReport()
	report.Changes[0].Impacts[1].Evidence = []domain.Evidence{
		{Type: "BM25", Location: "a:1", Excerpt: "first"},
		{Type: "LABEL_TOKEN", Location: "a:1", Excerpt: "first"},
		{Type: "PATH", Location: "a", Excerpt: "a"},
		{Type: "SEMANTIC_SIMILARITY", Location: "a:2", Excerpt: "second"},
	}
	limits := planning.Limits{CandidatesPerChange: 1, EvidencePerCandidate: 2}
	raw := planning.BuildEvidenceBundle(report, limits).Changes[0].Candidates[0].Evidence
	agent := planning.BuildAgentBundle(report, limits).Changes[0].Candidates[0].Evidence
	if len(raw) != 2 || len(agent) != 2 || raw[1].Excerpt != "first" || agent[1].Excerpt != "second" || agent[1].ID != "CH01-C01-E04" {
		t.Fatalf("raw=%+v agent=%+v", raw, agent)
	}
}

func TestAgentBundleGroupsOnlyUnconditionalJoinWiring(t *testing.T) {
	lead := func(excerpt string) []domain.Impact {
		return []domain.Impact{{File: "store.go", Evidence: []domain.Evidence{{Location: "store.go:10", Excerpt: excerpt}}}}
	}
	report := domain.Report{
		Changes: []domain.Change{
			{ChangeType: domain.AddGateway, NewElement: &domain.Element{ID: "join", Type: "exclusiveGateway"}, Impacts: lead("owner")},
			{ChangeType: domain.ModifySequenceFlow, NewElement: &domain.Element{ID: "wire"}, Impacts: lead("wire")},
			{ChangeType: domain.ModifySequenceFlow, NewElement: &domain.Element{ID: "conditional"}, Impacts: lead("condition")},
		},
		ProcessContext: &domain.ProcessContext{
			Before: &domain.Graph{},
			After: &domain.Graph{
				Incoming: map[string][]string{"join": {"wire", "conditional"}},
				Outgoing: map[string][]string{"join": {"next"}},
				Flows: map[string]*domain.Flow{
					"wire":        {SourceRef: "prepare", TargetRef: "join"},
					"conditional": {SourceRef: "check", TargetRef: "join", Condition: "valid"},
				},
			},
		},
	}
	bundle := planning.BuildAgentBundle(report, planning.Limits{GroupGatewayChanges: true})
	if len(bundle.Changes) != 2 || bundle.Changes[1].ElementID != "conditional" {
		t.Fatalf("unexpected units: %+v", bundle.Changes)
	}
	join := bundle.Changes[0]
	if len(join.SupportingFlowIDs) != 1 || join.SupportingFlowIDs[0] != "wire" || len(join.Candidates) != 1 || len(join.Candidates[0].Evidence) != 2 {
		t.Fatalf("wiring or evidence was lost: %+v", join)
	}
	if legacy := planning.BuildAgentBundle(report, planning.DefaultLimits()); len(legacy.Changes) != 3 {
		t.Fatalf("legacy checkpoint shape changed: %+v", legacy.Changes)
	}
}

func TestAgentBundleCarriesConditionBranchEvidenceIntoSplit(t *testing.T) {
	split := &domain.Element{ID: "split", Type: "exclusiveGateway"}
	report := domain.Report{
		Changes: []domain.Change{
			{ChangeType: domain.ModifyTaskLabel, NewElement: split},
			{ChangeType: domain.ModifyGatewayCondition, NewElement: split, Context: &domain.ChangeContext{
				BranchChanges: []domain.BranchChange{{FlowID: "branch", OldCondition: "valid", NewCondition: "valid and reason present"}},
			}},
			{ChangeType: domain.ModifySequenceFlow, NewElement: &domain.Element{ID: "branch"}, PartOf: "split",
				Impacts: []domain.Impact{{File: "panel.tsx", Evidence: []domain.Evidence{{Location: "panel.tsx:10", Excerpt: "send reason"}}}},
			},
			{ChangeType: domain.ModifySequenceFlow, NewElement: &domain.Element{ID: "unrelated"}},
		},
		ProcessContext: &domain.ProcessContext{
			Before: &domain.Graph{Flows: map[string]*domain.Flow{"branch": {SourceRef: "split", TargetRef: "old"}}},
			After:  &domain.Graph{Flows: map[string]*domain.Flow{"branch": {SourceRef: "split", TargetRef: "new"}}},
		},
	}
	bundle := planning.BuildAgentBundle(report, planning.Limits{GroupGatewayChanges: true, GroupGatewayBranches: true})
	if len(bundle.Changes) != 2 || bundle.Changes[1].ElementID != "unrelated" {
		t.Fatalf("unexpected grouping: %+v", bundle.Changes)
	}
	unit := bundle.Changes[0]
	if unit.ChangeType != domain.ModifyGatewayCondition || len(unit.BranchChanges) != 1 ||
		len(unit.SupportingFlowIDs) != 1 || unit.SupportingFlowIDs[0] != "branch" ||
		len(unit.Candidates) != 1 || unit.Candidates[0].File != "panel.tsx" ||
		len(unit.Candidates[0].Evidence) != 1 || unit.Candidates[0].Evidence[0].Excerpt != "send reason" {
		t.Fatalf("missing split condition or supporting evidence: %+v", unit)
	}
}

func TestEvidenceBundleContextIDIsStableAndSensitive(t *testing.T) {
	first := planning.BuildEvidenceBundle(fixtureReport(), planning.DefaultLimits())
	second := planning.BuildEvidenceBundle(fixtureReport(), planning.DefaultLimits())
	changed := fixtureReport()
	changed.Repository.Revision = "rev-2"
	third := planning.BuildEvidenceBundle(changed, planning.DefaultLimits())
	if len(first.ContextID) != 32 || first.ContextID != second.ContextID || first.ContextID == third.ContextID {
		t.Fatalf("context IDs first=%q second=%q changed=%q", first.ContextID, second.ContextID, third.ContextID)
	}
}

func TestOrchestratorEnforcesIdentityAndRoleTimeout(t *testing.T) {
	_, err := (planning.Orchestrator{Provider: &fakeProvider{mode: "empty-identity"}}).Run(context.Background(), fixtureReport())
	if err == nil || !strings.Contains(err.Error(), "requires provider") {
		t.Fatalf("identity error=%v", err)
	}
	_, err = (planning.Orchestrator{
		Provider: &fakeProvider{mode: "timeout"},
		Policy:   planning.Policy{VerifyTimeout: time.Millisecond},
	}).Run(context.Background(), fixtureReport())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error=%v", err)
	}
}

func fixtureReport() domain.Report {
	return domain.Report{
		AnalysisID: "analysis-1", InputFingerprint: "fingerprint-1", Method: domain.MethodHybrid,
		Repository: domain.Repository{Name: "fixture", Revision: "rev-1"},
		Changes: []domain.Change{{
			ChangeType: domain.AddTask,
			NewElement: &domain.Element{ID: "TASK-VALIDATE", Name: "Validate request", Type: "userTask"},
			Reason:     "task added",
			Impacts: []domain.Impact{
				{File: "web/request.tsx", Symbol: "RequestForm", Module: "web", Layer: "frontend", RequiredAction: domain.ActionModify, Confidence: 0.7, Evidence: []domain.Evidence{{Type: "LABEL_TOKEN", Location: "web/request.tsx:4", Excerpt: "request"}}},
				{File: "internal/request.go", Symbol: "Validate", Module: "internal", Layer: "application", RequiredAction: domain.ActionAdd, Confidence: 0.8, Evidence: []domain.Evidence{{Type: "TRACE_ID", Location: "internal/request.go:8", Excerpt: "TASK-VALIDATE"}}},
			},
		}},
	}
}

func findCandidate(bundle planning.EvidenceBundle, decision planning.CandidateDecision) planning.Candidate {
	for _, change := range bundle.Changes {
		if change.ChangeType != decision.ChangeType || change.ElementID != decision.ElementID {
			continue
		}
		for _, candidate := range change.Candidates {
			if candidate.File == decision.File && candidate.Symbol == decision.Symbol {
				return candidate
			}
		}
	}
	return planning.Candidate{}
}
