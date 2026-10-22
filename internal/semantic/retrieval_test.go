package semantic_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func TestSemanticCanDisplayOneBestCandidatePerFile(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW", Name: "Review", Type: "task"}}
	artifacts := []domain.Artifact{
		{ID: "a", File: "same.go", Content: "first"},
		{ID: "b", File: "same.go", Content: "second"},
		{ID: "c", File: "other.go", Content: "third"},
	}
	embedder := fakeEmbedder{vectors: [][]float64{{1, 0}, {1, 0}, {0.9, 0.435889894}, {0.8, 0.6}}}
	diverse, _, err := (semantic.Strategy{Embedder: embedder, DiversifyFiles: true}).Attach(context.Background(), []domain.Change{change}, artifacts, mapping.Options{TopK: 2})
	if err != nil || len(diverse[0].Impacts) != 2 || diverse[0].Impacts[0].File != "same.go" || diverse[0].Impacts[1].File != "other.go" {
		t.Fatalf("file-level display=%#v err=%v", diverse, err)
	}
	artifactsOnly, _, err := (semantic.Strategy{Embedder: embedder}).Attach(context.Background(), []domain.Change{change}, artifacts, mapping.Options{TopK: 2})
	if err != nil || artifactsOnly[0].Impacts[1].File != "same.go" {
		t.Fatalf("fusion branch artifact ranking changed: %#v err=%v", artifactsOnly, err)
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

func TestSemanticLongArtifactUsesBestBoundedPassage(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW", Name: "Check request", Type: "userTask"}}
	content := "func Start() {\n" + strings.Repeat("x", 700) + "\nfunc CheckRequest() {}"
	artifact := domain.Artifact{File: "service.go", Module: "app", Layer: "application", Content: content, Symbol: domain.Symbol{Name: "CheckRequest", Kind: "function", StartLine: 10}}
	strategy := semantic.Strategy{Embedder: fakeEmbedder{vectors: [][]float64{{1, 0}, {0, 1}, {1, 0}}}}
	result, _, err := strategy.Attach(context.Background(), []domain.Change{change}, []domain.Artifact{artifact}, mapping.Options{TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result[0].Impacts) != 1 || result[0].Impacts[0].Confidence != 1 {
		t.Fatalf("tail passage was not ranked: %#v", result[0].Impacts)
	}
	evidence := result[0].Impacts[0].Evidence[0]
	if !strings.Contains(evidence.Excerpt, "CheckRequest") || evidence.Location != "service.go:12" {
		t.Fatalf("evidence does not identify matched passage: %#v", evidence)
	}
}

type keywordEmbedder struct{}

func (keywordEmbedder) Identity() semantic.Identity {
	return semantic.Identity{Provider: "keyword-test", Model: "fixture"}
}

func (keywordEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	vectors := make([][]float64, len(texts))
	for index, value := range texts {
		if strings.Contains(strings.ToLower(value), "critical attachment") {
			vectors[index] = []float64{1, 0}
		} else {
			vectors[index] = []float64{0, 1}
		}
	}
	return vectors, nil
}

func TestSemanticLongArtifactCanRetrieveMiddlePassage(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW", Name: "Critical attachment", Type: "userTask"}}
	content := strings.Repeat("padding\n", 100) + "critical attachment handling\n" + strings.Repeat("padding\n", 100)
	artifact := domain.Artifact{ID: "service", File: "service.go", Content: content}
	result, warnings, err := (semantic.Strategy{Embedder: keywordEmbedder{}}).Attach(context.Background(), []domain.Change{change}, []domain.Artifact{artifact}, mapping.Options{TopK: 1})
	if err != nil || len(warnings) != 0 || len(result[0].Impacts) != 1 || result[0].Impacts[0].Confidence != 1 {
		t.Fatalf("middle passage was not retrieved: result=%#v warnings=%#v err=%v", result, warnings, err)
	}
	evidence := result[0].Impacts[0].Evidence[0]
	if !strings.Contains(evidence.Excerpt, "critical attachment handling") || evidence.Location != "service.go:101" || len([]rune(evidence.Excerpt)) > 240 {
		t.Fatalf("middle passage evidence is not located and bounded: %#v", evidence)
	}
}
