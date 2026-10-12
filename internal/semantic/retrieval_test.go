package semantic_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
)

type fakeEmbedder struct {
	vectors [][]float64
	err     error
}

func (fake fakeEmbedder) Identity() semantic.Identity {
	return semantic.Identity{Provider: "fake-v1", Model: "fixture"}
}

func (fake fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	if len(fake.vectors) != len(texts) {
		return nil, errors.New("fixture vector count mismatch")
	}
	return fake.vectors, nil
}

func TestSemanticRanksCosineCandidatesWithEvidence(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-APPROVE", Name: "Approve purchase request", Type: "userTask"}}
	artifacts := []domain.Artifact{
		{ID: "unrelated", File: "ui/dashboard.tsx", Module: "ui", Layer: "frontend", Content: "RenderSalesDashboard", Symbol: domain.Symbol{Name: "RenderSalesDashboard", Kind: "function", StartLine: 3}},
		{ID: "relevant", File: "purchase/service.go", Module: "purchase", Layer: "application", Content: "func AuthorizeProcurementDemand()", Symbol: domain.Symbol{Name: "AuthorizeProcurementDemand", Kind: "function", StartLine: 7}},
	}
	strategy := semantic.Strategy{Embedder: fakeEmbedder{vectors: [][]float64{{1, 0}, {0, 1}, {0.8, 0.6}}}}
	first, warnings, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, mapping.Options{TopK: 2, CandidateThreshold: 0.2})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, mapping.Options{TopK: 2, CandidateThreshold: 0.2})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 || len(first[0].Impacts) != 1 {
		t.Fatalf("impacts=%#v warnings=%#v", first[0].Impacts, warnings)
	}
	impact := first[0].Impacts[0]
	if impact.File != "purchase/service.go" || impact.Symbol != "AuthorizeProcurementDemand" || impact.RequiredAction != domain.ActionVerify {
		t.Fatalf("impact=%#v", impact)
	}
	if impact.Confidence != 0.8 || impact.ScoreBreakdown.Semantic != 0.8 || impact.Evidence[0].Type != "SEMANTIC_SIMILARITY" {
		t.Fatalf("semantic evidence=%#v", impact)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("semantic ranking is not deterministic")
	}
}

func TestSemanticAbstainsBelowThresholdAndPropagatesRuntimeFailure(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK", Name: "Approve request", Type: "task"}}
	artifact := domain.Artifact{ID: "one", File: "one.go", Content: "unrelated"}
	strategy := semantic.Strategy{Embedder: fakeEmbedder{vectors: [][]float64{{1, 0}, {0, 1}}}}
	changes, _, err := strategy.Attach(context.Background(), []domain.Change{change}, []domain.Artifact{artifact}, mapping.Options{CandidateThreshold: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) == 0 {
		t.Fatalf("expected explicit abstention: %#v", changes[0])
	}
	strategy.Embedder = fakeEmbedder{err: errors.New("model unavailable")}
	if _, _, err := strategy.Attach(context.Background(), []domain.Change{change}, []domain.Artifact{artifact}, mapping.Options{}); err == nil {
		t.Fatal("semantic runtime failure was hidden")
	}
}
