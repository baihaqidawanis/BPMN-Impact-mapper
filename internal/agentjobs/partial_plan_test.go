package agentjobs_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestExportOldPausedCheckpointWithoutInferenceOrMutation(t *testing.T) {
	m := &model{}
	s := service(t, m)
	ctx := context.Background()
	b := planning.BuildEvidenceBundle(fixtureReport(), planning.Limits{})
	c := b.Changes[0].Candidates[0]
	checkpoint := planning.AgentSession{Version: "1", ID: strings.Repeat("d", 32), Status: "PAUSED", Identity: m.Identity(), Bundle: b,
		Decisions: []planning.CandidateDecision{{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictRelevant, Rationale: "Stored relevant target", EvidenceIDs: []string{c.Evidence[0].ID}}},
		Draft:     []planning.FilePlan{{ChangeType: b.Changes[0].ChangeType, ElementID: b.Changes[0].ElementID, File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: domain.ActionVerify, Summary: "Stored proposal", Steps: []string{"Add the scoped guard before mutation."}, EvidenceIDs: []string{c.Evidence[0].ID}}}}
	rejected := c
	rejected.Symbol = "RejectedHandler"
	checkpoint.Bundle.Changes[0].Candidates = append(checkpoint.Bundle.Changes[0].Candidates, rejected)
	rejection := checkpoint.Decisions[0]
	rejection.Symbol = rejected.Symbol
	rejection.Verdict = planning.VerdictReject
	checkpoint.Decisions = append(checkpoint.Decisions, rejection)
	conflict := checkpoint.Draft[0]
	conflict.Symbol = rejected.Symbol
	checkpoint.Draft = append(checkpoint.Draft, conflict)
	if err := s.Sessions.Save(ctx, checkpoint); err != nil {
		t.Fatal(err)
	}
	before, err := s.Sessions.Load(ctx, checkpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.ExportDraft(ctx, checkpoint.ID)
	if err != nil || p.Status != planning.StatusNeedsReview || len(p.Plans) != 1 || p.Plans[0].Symbol != c.Symbol || m.calls.Load() != 0 {
		t.Fatal("export failed or inferred approval", p, err)
	}
	if !strings.Contains(strings.Join(p.ReviewIssues, "\n"), "Withheld invalid proposal") {
		t.Fatal("missing explanation for withheld proposal", p.ReviewIssues)
	}
	after, err := s.Sessions.Load(ctx, checkpoint.ID)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("checkpoint changed", err)
	}
	if _, err = s.Plans.Get(ctx, p.PlanID); err != nil {
		t.Fatal("export was not persisted", err)
	}
}
