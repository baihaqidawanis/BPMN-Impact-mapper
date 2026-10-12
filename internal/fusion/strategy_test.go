package fusion_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type fakeStrategy struct {
	method   domain.AnalysisMethod
	changes  []domain.Change
	warnings []domain.Warning
	err      error
}

func (fake fakeStrategy) Method() domain.AnalysisMethod { return fake.method }

func (fake fakeStrategy) Attach(context.Context, []domain.Change, []domain.Artifact, mapping.Options) ([]domain.Change, []domain.Warning, error) {
	return fake.changes, fake.warnings, fake.err
}

func TestRRFRewardsAgreementAndMergesEvidence(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "T", Type: "task"}}}
	left := fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("a.go", domain.ActionVerify, "BM25"), impact("b.go", domain.ActionVerify, "BM25"),
	}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("b.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"), impact("c.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"),
	}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodFusion, Left: left, Right: right}

	first, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("rank fusion is not deterministic")
	}
	if first[0].UnresolvedImpact == nil {
		t.Fatal("resolved fusion result must encode unresolved_impacts as an empty array")
	}
	if got := first[0].Impacts[0]; got.File != "b.go" || got.ScoreBreakdown.RRF != 0.9919 || len(got.Evidence) != 3 {
		t.Fatalf("unexpected fused winner: %#v", got)
	}
	if first[0].Impacts[1].File != "a.go" || first[0].Impacts[2].File != "c.go" {
		t.Fatalf("tie order is not deterministic: %#v", first[0].Impacts)
	}
}

func TestGraphAwareHybridPreservesOnlyLeftActions(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.ModifyTaskLabel}}
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{impact("direct.go", domain.ActionModify, "TRACE_ID")}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("semantic.go", domain.ActionModify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, PreserveLeftActions: true}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]domain.RequiredAction{}
	for _, candidate := range changes[0].Impacts {
		actions[candidate.File] = candidate.RequiredAction
	}
	if actions["direct.go"] != domain.ActionModify || actions["semantic.go"] != domain.ActionVerify {
		t.Fatalf("unsafe fused actions: %#v", actions)
	}
}

func TestGraphAwareHybridPrefersGraphCandidateOnEqualRRFScore(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask}}
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{impact("z-graph.go", domain.ActionVerify, "TRACE_ID")}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("a-semantic.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, PreserveLeftActions: true}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if changes[0].Impacts[0].File != "z-graph.go" {
		t.Fatalf("semantic-only tie displaced graph evidence: %#v", changes[0].Impacts)
	}
}

func TestFusionPropagatesFailureAndAbstention(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask}}
	strategy := fusion.Strategy{MethodName: domain.MethodFusion, Left: fakeStrategy{method: domain.MethodBM25, err: errors.New("failed")}, Right: fakeStrategy{method: domain.MethodSemantic}}
	if _, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{}); err == nil {
		t.Fatal("retriever failure was hidden")
	}
	strategy.Left = fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{UnresolvedImpact: []string{"no lexical candidate"}}}}
	strategy.Right = fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{UnresolvedImpact: []string{"no semantic candidate"}}}}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) != 2 {
		t.Fatalf("missing abstention reasons: %#v", changes[0])
	}
}

func impact(file string, action domain.RequiredAction, evidenceType string) domain.Impact {
	return domain.Impact{
		File: file, RequiredAction: action, Evidence: []domain.Evidence{{Type: evidenceType, Location: file, Excerpt: evidenceType}},
	}
}
