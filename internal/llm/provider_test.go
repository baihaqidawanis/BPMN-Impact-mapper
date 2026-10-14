package llm_test

import (
	"context"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestNoLLMProviderIsOfflineAndExplicit(t *testing.T) {
	provider := llm.NoLLMProvider{}
	result, err := provider.Verify(context.Background(), llm.VerificationRequest{})
	if err != nil || result.Enabled || provider.Name() != "none" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	plan, err := (planning.Orchestrator{Provider: provider}).Run(context.Background(), domain.Report{AnalysisID: "analysis", InputFingerprint: "fingerprint"})
	if err != nil || plan.Status != planning.StatusDisabled || plan.Provider.Provider != "none" || len(plan.ContextID) != 32 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
}
