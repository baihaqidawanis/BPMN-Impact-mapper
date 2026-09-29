package llm

import (
	"context"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
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

func (NoLLMProvider) Name() string { return "none" }

func (NoLLMProvider) Verify(_ context.Context, request VerificationRequest) (VerificationResult, error) {
	return VerificationResult{Enabled: false, Explanation: "LLM verification is disabled", Candidates: append([]domain.Impact(nil), request.Candidates...)}, nil
}
