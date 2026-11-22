package agentjobs_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type model struct {
	gate  chan struct{}
	fail  atomic.Bool
	calls atomic.Int32
}

func (m *model) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "scripted", Model: "worker", PromptVersion: "v1"}
}
func (m *model) Prepare(context.Context) error { return nil }
func (m *model) Turn(ctx context.Context, t planning.AgentTurn) (planning.AgentReply, error) {
	m.calls.Add(1)
	if m.gate != nil {
		select {
		case <-m.gate:
		case <-ctx.Done():
			return planning.AgentReply{}, ctx.Err()
		}
	}
	if m.fail.Swap(false) {
		return planning.AgentReply{}, errors.New("temporary connection error")
	}
	switch t.Role {
	case planning.Analyst:
		var out planning.VerificationResponse
		for _, change := range t.Bundle.Changes {
			for _, c := range change.Candidates {
				out.Decisions = append(out.Decisions, planning.CandidateDecision{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictRelevant, Rationale: "Validate requirement against current evidence.", EvidenceIDs: []string{c.Evidence[0].ID}})
			}
		}
		return planning.AgentReply{Verification: &out}, nil
	case planning.Planner:
		var out planning.LayerPlanningResponse
		for _, change := range t.Bundle.Changes {
			for _, c := range change.Candidates {
				out.Plans = append(out.Plans, planning.FilePlan{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: domain.ActionVerify, Summary: "Conditional change proposal", ProposedChanges: []string{"Require the new validation before submission while preserving unrelated actions."}, AcceptanceChecks: []string{"Invalid input is rejected before state mutation; valid input preserves the existing transition."}, Steps: []string{"Verify the new guard before submission."}, EvidenceIDs: []string{c.Evidence[0].ID}})
			}
		}
		return planning.AgentReply{Plan: &out}, nil
	default:
		return planning.AgentReply{Review: &planning.AgentReview{Approved: true}}, nil
	}
}

type snapshot struct{}

func (snapshot) Identity() string                          { return "fixed-source" }
func (snapshot) CheckBundle(planning.EvidenceBundle) error { return nil }
func (snapshot) Execute(context.Context, planning.ToolRequest) (planning.ToolResult, error) {
	return planning.ToolResult{}, nil
}

func fixtureReport() domain.Report {
	r := domain.Report{AnalysisID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", InputFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Method: domain.MethodHybrid}
	for _, id := range []string{"first", "second"} {
		r.Changes = append(r.Changes, domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: id, Name: id, Type: "userTask"}, Impacts: []domain.Impact{{File: "source.go", Symbol: "Submit", Module: "core", Layer: "application", RequiredAction: domain.ActionVerify, Evidence: []domain.Evidence{{Type: "SOURCE", Location: "source.go:1", Excerpt: "func Submit() {}"}}}}})
	}
	return r
}
func service(t *testing.T, m *model) *agentjobs.Service {
	root := t.TempDir()
	return &agentjobs.Service{Model: m, Sessions: filestore.AgentSessionStore{Root: filepath.Join(root, "sessions")}, Plans: filestore.PlanStore{Root: filepath.Join(root, "plans")}, Source: func(context.Context, string) (agentjobs.Snapshot, error) { return snapshot{}, nil }}
}
func finish(t *testing.T, s *agentjobs.Service, id string) agentjobs.View {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestWorkerSelectionBusyPersistenceAndNoReplay(t *testing.T) {
	m := &model{gate: make(chan struct{})}
	s := service(t, m)
	ctx := context.Background()
	r := fixtureReport()
	v, err := s.Start(ctx, r, "repo", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(ctx, r, "repo", 0, ""); !errors.Is(err, agentjobs.ErrBusy) {
		t.Fatal("concurrent worker accepted")
	}
	if err := s.Check(ctx); !errors.Is(err, agentjobs.ErrBusy) {
		t.Fatal("model metadata could be repinned during a running review", err)
	}
	close(m.gate)
	done := finish(t, s, v.SessionID)
	if done.Status != "COMPLETE" || done.Plan == nil || len(done.Context.Changes) != 1 || done.Context.Changes[0].ElementID != "second" {
		t.Fatalf("wrong scope/result: %+v", done)
	}
	restarted := &agentjobs.Service{Model: m, Sessions: s.Sessions, Plans: s.Plans, Source: s.Source}
	loaded, err := restarted.Get(ctx, v.SessionID)
	if err != nil || loaded.Plan == nil {
		t.Fatal("saved plan lost on restart", err)
	}
	before := m.calls.Load()
	if _, err := restarted.Start(ctx, r, "repo", 1, v.SessionID); err != nil {
		t.Fatal(err)
	}
	finish(t, restarted, v.SessionID)
	if m.calls.Load() != before {
		t.Fatal("completed session reran model")
	}
	if _, err := s.Start(ctx, r, "repo", 0, v.SessionID); !errors.Is(err, agentjobs.ErrBinding) {
		t.Fatal("foreign change resumed")
	}
}

func TestReopensArchivedPlanAfterValidationPolicyChanges(t *testing.T) {
	ctx := context.Background()
	s := service(t, &model{})
	report := fixtureReport()
	report.Changes = report.Changes[:1]
	report.Changes[0].Impacts[0].File = "backend/migrations/000001_stock.up.sql"
	report.Changes[0].Impacts[0].RequiredAction = domain.ActionModify
	bundle := planning.BuildEvidenceBundle(report, planning.DefaultLimits())
	change := bundle.Changes[0]
	candidate := change.Candidates[0]
	proposal := planning.FilePlan{
		ChangeType: change.ChangeType, ElementID: change.ElementID, File: candidate.File,
		Symbol: candidate.Symbol, Module: candidate.Module, Layer: candidate.Layer,
		Action: domain.ActionModify, Summary: "Extend allowed status values.",
		Steps: []string{"Update the existing status constraint."}, EvidenceIDs: []string{candidate.Evidence[0].ID},
	}
	checkpoint := planning.AgentSession{
		ID: "legacy-session", Version: "1", Status: "NEEDS_REVIEW", Bundle: bundle,
		Draft: []planning.FilePlan{proposal},
		Decisions: []planning.CandidateDecision{{
			ChangeType: change.ChangeType, ElementID: change.ElementID, File: candidate.File, Symbol: candidate.Symbol,
			Verdict: planning.VerdictRelevant, EvidenceIDs: proposal.EvidenceIDs,
		}},
	}
	if _, err := checkpoint.ChangePlan(); !errors.Is(err, planning.ErrAgentPlanIncomplete) {
		t.Fatalf("fresh validation accepted a legacy edit: %v", err)
	}
	if err := s.Sessions.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	if view, err := s.Get(ctx, checkpoint.ID); err != nil || view.Plan != nil {
		t.Fatalf("unarchived invalid proposal was exposed: %+v, %v", view.Plan, err)
	}
	archive := planning.ChangePlan{
		Version: planning.BundleVersion, AnalysisID: bundle.AnalysisID, InputFingerprint: bundle.InputFingerprint,
		ContextID: bundle.ContextID, Status: planning.StatusNeedsReview, Plans: checkpoint.Draft,
		ReviewIssues: []string{},
		ChangeCoverage: []planning.ChangeCoverage{{
			ChangeType: change.ChangeType, ElementID: change.ElementID, ElementName: change.ElementName,
			Status: "PLANNED", Relevant: 1, Plans: 1,
		}},
	}
	var err error
	archive.PlanID, err = planning.ComputePlanID(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Plans.Save(ctx, archive); err != nil {
		t.Fatal(err)
	}
	view, err := s.Get(ctx, checkpoint.ID)
	if err != nil || view.Plan == nil || view.Plan.PlanID != archive.PlanID || view.Plan.Status != planning.StatusNeedsReview {
		t.Fatalf("archive was not restored by its original ID: %+v, %v", view.Plan, err)
	}
}

func TestReviewGroupsOnlyExplicitSupportingFlowsWithoutSourceLeads(t *testing.T) {
	r := fixtureReport()
	task := r.Changes[0]
	for _, kind := range []domain.ChangeType{domain.AddTask, domain.ModifyGatewayCondition} {
		task.ChangeType = kind
		flow := domain.Change{ChangeType: domain.ModifySequenceFlow, PartOf: task.NewElement.ID, NewElement: &domain.Element{ID: "edge"}}
		independent := flow
		independent.NewElement = &domain.Element{ID: "coded-edge"}
		independent.Impacts = task.Impacts
		orphan := flow
		orphan.PartOf = "unknown"
		orphan.NewElement = &domain.Element{ID: "orphan"}
		r.Changes = []domain.Change{flow, task, independent, orphan}
		b := planning.BuildAgentBundle(r, planning.DefaultLimits())
		raw := planning.BuildEvidenceBundle(r, planning.DefaultLimits())
		if len(b.Changes) != 3 || len(b.Changes[0].SupportingFlowIDs) != 1 || b.Changes[0].SupportingFlowIDs[0] != "edge" ||
			b.Changes[1].ElementID != "coded-edge" || b.Changes[2].ElementID != "orphan" ||
			b.Changes[0].Candidates[0].Evidence[0].ID != raw.Changes[1].Candidates[0].Evidence[0].ID {
			t.Fatalf("incorrect planning units for %s: %+v", kind, b.Changes)
		}
	}
	// Exercise the web worker's normal all-changes entry point.
	task.ChangeType = domain.AddTask
	r.Changes = []domain.Change{r.Changes[0], task}
	s := service(t, &model{})
	v, err := s.Start(context.Background(), r, "repo", -1, "")
	if err != nil {
		t.Fatal(err)
	}
	done := finish(t, s, v.SessionID)
	if done.Status != "COMPLETE" || done.Plan == nil || len(done.Plan.ChangeCoverage) != 1 || done.FocusedReview.BatchesTotal != 1 {
		t.Fatalf("supporting flow received a separate session: %+v", done)
	}
}

func TestWorkerConnectionFailureResumeAndShutdown(t *testing.T) {
	m := &model{}
	m.fail.Store(true)
	s := service(t, m)
	ctx := context.Background()
	r := fixtureReport()
	v, err := s.Start(ctx, r, "repo", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	paused := finish(t, s, v.SessionID)
	if paused.Status != "PAUSED" {
		t.Fatal("not resumable", paused.Status)
	}
	if !strings.Contains(paused.Message, "checkpoint is saved") || strings.Contains(paused.Message, "Restore VPN/tunnel") {
		t.Fatal("generic model failure misdiagnosed as VPN failure", paused.Message)
	}
	if _, err := s.Start(ctx, r, "repo", 0, v.SessionID); err != nil {
		t.Fatal(err)
	}
	done := finish(t, s, v.SessionID)
	// Failed Analyst + successful Analyst/Planner + one combined Reviewer.
	if done.Status != "COMPLETE" || done.Turns != 4 {
		t.Fatal("resume lost charged turns")
	}
	base, cancel := context.WithCancel(context.Background())
	s.Base = base
	cancel()
	if _, err := s.Start(ctx, r, "repo", 0, ""); err == nil {
		t.Fatal("started after shutdown")
	}
}

func TestStopCancelsActiveReviewAndReleasesWorker(t *testing.T) {
	m := &model{gate: make(chan struct{})}
	s := service(t, m)
	v, err := s.Start(context.Background(), fixtureReport(), "repo", -1, "")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for m.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if err := s.Stop(v.SessionID); err != nil {
		t.Fatal(err)
	}
	result := finish(t, s, v.SessionID)
	if result.Status != "PAUSED" {
		t.Fatalf("stop did not preserve paused session: %s", result.Status)
	}
	if len(result.Context.Changes) != 2 {
		t.Fatal("all-change scope lost")
	}
	if err := s.Stop(v.SessionID); !errors.Is(err, agentjobs.ErrMissing) {
		t.Fatal("stopped worker still active", err)
	}
}

func TestFocusedServiceIncludesAllDisplayedCandidates(t *testing.T) {
	s := service(t, &model{})
	r := fixtureReport()
	r.Changes = r.Changes[:1]
	seed := r.Changes[0].Impacts[0]
	r.Changes[0].Impacts = nil
	for i := 0; i < 10; i++ {
		c := seed
		c.File = fmt.Sprintf("source%d.go", i)
		c.Evidence = []domain.Evidence{{Type: "SOURCE", Location: c.File + ":1", Excerpt: "submit"}}
		r.Changes[0].Impacts = append(r.Changes[0].Impacts, c)
	}
	v, err := s.Start(context.Background(), r, "repo", -1, "", "focused")
	if err != nil {
		t.Fatal(err)
	}
	done := finish(t, s, v.SessionID)
	if done.Status != "COMPLETE" || done.Plan == nil || len(done.Plan.Plans) != 10 || done.FocusedReview == nil || done.FocusedReview.CandidatesProcessed != 10 {
		t.Fatal("service dropped displayed seeds", done.Status, done.Message)
	}
}
