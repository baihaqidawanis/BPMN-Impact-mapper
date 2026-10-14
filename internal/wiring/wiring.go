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
	return app.Analyzer{Resolver: strategyResolver{semanticEmbedder: semantic.NewPythonEmbedderFromEnvironment()}}
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
			MethodName: domain.MethodHybrid, Left: retrieval.Hybrid{}, Right: semanticStrategy,
			PreserveLeftActions: true,
		}, fusionIdentity(domain.MethodHybrid, identity), nil
	case domain.MethodBM25:
		return retrieval.BM25Only{}, nil, nil
	case domain.MethodSemantic:
		strategy, identity, err := resolver.semanticStrategy()
		if err != nil {
			return nil, nil, err
		}
		return strategy, identity, nil
	case domain.MethodFusion:
		semanticStrategy, identity, err := resolver.semanticStrategy()
		if err != nil {
			return nil, nil, err
		}
		return fusion.Strategy{
			MethodName: domain.MethodFusion, Left: retrieval.BM25Only{}, Right: semanticStrategy,
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
	identity, err := json.Marshal(resolver.semanticEmbedder.Identity())
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
