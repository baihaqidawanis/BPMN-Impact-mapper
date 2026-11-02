package wiring

import (
	"context"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
)

type identityOnlyEmbedder struct{}

func (identityOnlyEmbedder) Identity() semantic.Identity {
	return semantic.Identity{Provider: "test", Model: "test"}
}

func (identityOnlyEmbedder) Embed(_ context.Context, _ []string) ([][]float64, error) {
	return nil, nil
}

func TestProductionHybridUsesContextBM25WithoutLegacyWeights(t *testing.T) {
	strategy, _, err := (strategyResolver{semanticEmbedder: identityOnlyEmbedder{}}).Resolve(domain.MethodHybrid, "")
	if err != nil {
		t.Fatal(err)
	}
	combined, ok := strategy.(fusion.Strategy)
	if !ok {
		t.Fatalf("unexpected strategy: %T", strategy)
	}
	if _, ok := combined.Left.(retrieval.ContextBM25); !ok || combined.PreserveLeftActions {
		t.Fatalf("production Hybrid uses legacy weighted ranking or action promotion: %#v", combined)
	}
	if !combined.Right.(semantic.Strategy).IncludeProcessContext {
		t.Fatal("production semantic branch lost explicit process context")
	}
}

func TestPublicMethodsUseFileLevelTopKWithoutChangingFusionBranches(t *testing.T) {
	resolver := strategyResolver{semanticEmbedder: identityOnlyEmbedder{}}
	bm25, _, err := resolver.Resolve(domain.MethodBM25, "")
	if err != nil || !bm25.(retrieval.BM25Only).DiversifyFiles {
		t.Fatalf("BM25 display does not select one candidate per file: %T %v", bm25, err)
	}
	semanticOnly, _, err := resolver.Resolve(domain.MethodSemantic, "")
	if err != nil || !semanticOnly.(semantic.Strategy).DiversifyFiles {
		t.Fatalf("Semantic display does not select one candidate per file: %T %v", semanticOnly, err)
	}
	for _, method := range []domain.AnalysisMethod{domain.MethodFusion, domain.MethodHybrid} {
		value, _, err := resolver.Resolve(method, "")
		if err != nil {
			t.Fatal(err)
		}
		combined := value.(fusion.Strategy)
		if !combined.DiversifyFiles || combined.Right.(semantic.Strategy).DiversifyFiles {
			t.Fatalf("%s must diversify only after artifact-level RRF: %#v", method, combined)
		}
		if method == domain.MethodFusion && combined.Left.(retrieval.BM25Only).DiversifyFiles {
			t.Fatal("Fusion BM25 branch lost artifact-level ranking")
		}
	}
}
