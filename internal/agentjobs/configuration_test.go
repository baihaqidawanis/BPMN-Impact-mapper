package agentjobs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type configuredModel struct {
	*model
	alias       string
	failPrepare bool
	prepared    int
}

func (m *configuredModel) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "fixture", Model: m.alias, PromptVersion: "configured-v1"}
}
func (m *configuredModel) Prepare(context.Context) error {
	m.prepared++
	if m.failPrepare {
		return errors.New("metadata unavailable")
	}
	return nil
}
func settingsFixture() agentjobs.Settings {
	r := agentjobs.RoleSettings{Model: "installed-fixture", Profile: "coding"}
	return agentjobs.Settings{Analyst: r, Planner: r, Reviewer: r, Context: 32768, Output: 6144}
}

func TestConfigurationValidationAndMetadataFailureAreAtomic(t *testing.T) {
	original := &model{}
	v := settingsFixture()
	builds := 0
	next := &configuredModel{model: &model{}, alias: "next"}
	s := service(t, original)
	s.Configuration = &agentjobs.Configuration{Current: v, Build: func(agentjobs.Settings) (agentjobs.Model, error) { builds++; return next, nil }}
	for _, mutate := range []func(*agentjobs.Settings){
		func(v *agentjobs.Settings) { v.Context = 0 }, func(v *agentjobs.Settings) { v.Output = 32768 },
		func(v *agentjobs.Settings) { v.Analyst.Model = "" }, func(v *agentjobs.Settings) { v.Reviewer.Profile = "unknown" },
	} {
		bad := v
		mutate(&bad)
		if err := s.Configure(context.Background(), bad); !errors.Is(err, agentjobs.ErrConfiguration) {
			t.Fatal(err)
		}
	}
	if builds != 0 || s.Model != original {
		t.Fatal("invalid settings reached model factory")
	}
	v.Reviewer.Thinking = true
	next.failPrepare = true
	if err := s.Configure(context.Background(), v); err == nil {
		t.Fatal("failed preparation accepted")
	}
	saved, _ := s.Settings()
	if saved.Reviewer.Thinking || s.Model != original {
		t.Fatal("failed apply changed active settings")
	}
	next.failPrepare = false
	if err := s.Configure(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	saved, _ = s.Settings()
	if !saved.Reviewer.Thinking || s.Model != next || next.calls.Load() != 0 {
		t.Fatal("settings lost or configuration ran inference")
	}
}

func TestRunningReviewLocksConfigurationAndChangedModelCannotResume(t *testing.T) {
	m := &model{gate: make(chan struct{})}
	s := service(t, m)
	builds := 0
	s.Configuration = &agentjobs.Configuration{Current: settingsFixture(), Build: func(agentjobs.Settings) (agentjobs.Model, error) {
		builds++
		return &configuredModel{model: &model{}, alias: "changed"}, nil
	}}
	v, err := s.Start(context.Background(), fixtureReport(), "repo", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Configure(context.Background(), settingsFixture()); !errors.Is(err, agentjobs.ErrBusy) || builds != 0 {
		t.Fatal("configuration changed during review", err)
	}
	close(m.gate)
	finish(t, s, v.SessionID)
	if err := s.Configure(context.Background(), settingsFixture()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), fixtureReport(), "repo", 0, v.SessionID); !errors.Is(err, agentjobs.ErrBinding) {
		t.Fatal("changed model resumed old checkpoint", err)
	}
	old, err := s.Get(context.Background(), v.SessionID)
	if err != nil || old.Provider.Model != "worker" {
		t.Fatal("old checkpoint altered", err)
	}
}
