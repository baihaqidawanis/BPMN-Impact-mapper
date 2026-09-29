// Package wiring is the composition root. It is the only internal package that
// knows both application use cases and concrete mapping adapters.
package wiring

import (
	"encoding/json"
	"fmt"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/configmapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
)

func NewAnalyzer() app.Analyzer { return app.Analyzer{Resolver: strategyResolver{}} }

type strategyResolver struct{}

func (strategyResolver) Resolve(method domain.AnalysisMethod, configPath string) (mapping.Strategy, []byte, error) {
	switch method {
	case domain.MethodHybrid:
		return retrieval.Hybrid{}, nil, nil
	case domain.MethodBM25:
		return retrieval.BM25Only{}, nil, nil
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
