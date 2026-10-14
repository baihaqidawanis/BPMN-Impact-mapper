package llm

import (
	"context"
	"errors"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

type VerificationRequest struct {
	Change     domain.Change   `json:"change"`
	Candidates []domain.Impact `json:"candidates"`
}

type VerificationResult struct {
	Enabled     bool            `json:"enabled"`
	Explanation string          `json:"explanation"`
	Candidates  []domain.Impact `json:"candidates"`
}

type Provider interface {
	Name() string
	Verify(context.Context, VerificationRequest) (VerificationResult, error)
}

type NoLLMProvider struct{}

func (NoLLMProvider) Name() string  { return "none" }
func (NoLLMProvider) Enabled() bool { return false }

func (NoLLMProvider) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "none", Model: "none", PromptVersion: "none"}
}

func (NoLLMProvider) Verify(_ context.Context, request VerificationRequest) (VerificationResult, error) {
	return VerificationResult{Enabled: false, Explanation: "LLM verification is disabled", Candidates: append([]domain.Impact(nil), request.Candidates...)}, nil
}

func (NoLLMProvider) VerifyCandidates(context.Context, planning.VerificationRequest) (planning.VerificationResponse, error) {
	return planning.VerificationResponse{}, errors.New("LLM planning is disabled")
}

func (NoLLMProvider) PlanLayer(context.Context, planning.LayerPlanningRequest) (planning.LayerPlanningResponse, error) {
	return planning.LayerPlanningResponse{}, errors.New("LLM planning is disabled")
}

func (NoLLMProvider) ReviewPlan(context.Context, planning.ReviewRequest) (planning.ReviewResponse, error) {
	return planning.ReviewResponse{}, errors.New("LLM planning is disabled")
}

var _ planning.Provider = NoLLMProvider{}
