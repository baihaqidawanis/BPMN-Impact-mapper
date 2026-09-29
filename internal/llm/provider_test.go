package llm_test

import (
	"context"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
)

func TestNoLLMProviderIsOfflineAndExplicit(t *testing.T) {
	provider := llm.NoLLMProvider{}
	result, err := provider.Verify(context.Background(), llm.VerificationRequest{})
	if err != nil || result.Enabled || provider.Name() != "none" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
