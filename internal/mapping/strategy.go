// Package mapping defines the application-facing port for interchangeable
// impact-mapping methods. Implementations live in outward adapter packages.
package mapping

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

type Options struct {
	TopK               int     `json:"top_k,omitempty"`
	CandidateThreshold float64 `json:"candidate_threshold,omitempty"`
	HighThreshold      float64 `json:"high_threshold,omitempty"`
	TieDelta           float64 `json:"tie_delta,omitempty"`
}

type Strategy interface {
	Method() domain.AnalysisMethod
	Attach(changes []domain.Change, artifacts []domain.Artifact, options Options) ([]domain.Change, []domain.Warning)
}

type Resolver interface {
	Resolve(method domain.AnalysisMethod, configurationPath string) (Strategy, []byte, error)
}
