package agentjobs_test

import (
	"context"
	"errors"
	"path/filepath"
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
	if _, err := s.Start(ctx, r, "repo", 0, v.SessionID); err != nil {
		t.Fatal(err)
	}
	done := finish(t, s, v.SessionID)
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
