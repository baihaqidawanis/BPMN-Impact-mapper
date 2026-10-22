// Package wiring is the composition root. It is the only internal package that
// knows both application use cases and concrete mapping adapters.
package wiring

import (
	"encoding/json"
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/configmapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/semantic"
)

func NewAnalyzer() app.Analyzer {
	return NewAnalyzerWithSemanticEmbedder(semantic.NewPythonEmbedderFromEnvironment())
}

// NewAnalyzerWithSemanticEmbedder keeps runtime construction in the composition
// root while allowing the server to share one configured embedder with its
// capability check.
func NewAnalyzerWithSemanticEmbedder(embedder semantic.Embedder) app.Analyzer {
	return app.Analyzer{Resolver: strategyResolver{semanticEmbedder: embedder}}
}

type strategyResolver struct{ semanticEmbedder semantic.Embedder }

func (resolver strategyResolver) Resolve(method domain.AnalysisMethod, configPath string) (mapping.Strategy, []byte, error) {
	switch method {
	case domain.MethodHybrid:
		semanticStrategy, identity, err := resolver.semanticStrategy()
		if err != nil {
			return nil, nil, err
		}
		return fusion.Strategy{
			MethodName: domain.MethodHybrid, Left: retrieval.ContextBM25{}, Right: semanticStrategy,
			PreserveLeftActions: false,
			DiversifyFiles:      true,
		}, fusionIdentity(domain.MethodHybrid, identity), nil
	case domain.MethodBM25:
		return retrieval.BM25Only{DiversifyFiles: true}, nil, nil
	case domain.MethodSemantic:
		strategy, identity, err := resolver.semanticStrategy()
		if err != nil {
			return nil, nil, err
		}
		strategy.DiversifyFiles = true
		return strategy, identity, nil
	case domain.MethodFusion:
		semanticStrategy, identity, err := resolver.semanticStrategy()
		if err != nil {
			return nil, nil, err
		}
		return fusion.Strategy{
			MethodName: domain.MethodFusion, Left: retrieval.BM25Only{}, Right: semanticStrategy,
			DiversifyFiles: true,
		}, fusionIdentity(domain.MethodFusion, identity), nil
	case domain.MethodConfiguration:
		mapper, err := configmapping.Load(configPath)
		if err != nil {
			return nil, nil, err
		}
		canonical, err := json.Marshal(mapper.Config)
		if err != nil {
			return nil, nil, err
		}
		return mapper, canonical, nil
	default:
		return nil, nil, fmt.Errorf("unsupported analysis method %q", method)
	}
}

func (resolver strategyResolver) semanticStrategy() (semantic.Strategy, []byte, error) {
	if resolver.semanticEmbedder == nil {
		return semantic.Strategy{}, nil, fmt.Errorf("semantic embedder is not configured")
	}
	identityValue := resolver.semanticEmbedder.Identity()
	// Resolver identity must remain stable before and after the first runtime
	// call. Dynamic package/model metadata is added to the final analysis
	// identity by app.Analyzer after retrieval completes.
	identityValue.PythonVersion = ""
	identityValue.SentenceTransformersVersion = ""
	identityValue.ModelArtifactSHA256 = ""
	identityValue.EmbeddingDimension = 0
	identity, err := json.Marshal(identityValue)
	if err != nil {
		return semantic.Strategy{}, nil, err
	}
	return semantic.Strategy{Embedder: resolver.semanticEmbedder}, identity, nil
}

func fusionIdentity(method domain.AnalysisMethod, semanticIdentity []byte) []byte {
	identity, _ := json.Marshal(struct {
		Method   domain.AnalysisMethod `json:"method"`
		Fusion   string                `json:"fusion"`
		RRFK     int                   `json:"rrf_k"`
		Semantic json.RawMessage       `json:"semantic"`
	}{method, "reciprocal-rank-fusion-v2-full-pool", fusion.DefaultRRFK, semanticIdentity})
	return identity
}
