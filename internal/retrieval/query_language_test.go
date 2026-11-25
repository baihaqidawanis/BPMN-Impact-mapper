package retrieval

import (
	"context"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

func TestMappingBM25PreservesConditionWordsAndExpandsBusinessTerms(t *testing.T) {
	query := mappingBM25Query([]string{"dan", "yang", "and", "the", "alasan", "versi", "tidak", "belum", "atau", "reason"})
	want := []string{"alasan", "atau", "belum", "reason", "tidak", "versi", "version"}
	if !reflect.DeepEqual(query, want) {
		t.Fatalf("query = %v, want %v", query, want)
	}
}

func TestMappingBM25BranchesUseBilingualQueries(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "a", File: "form.tsx", Content: "reason version", Tokens: []string{"reason", "version"}},
		{ID: "b", File: "unrelated.md", Content: "dan yang", Tokens: []string{"dan", "yang"}},
	}
	for _, strategy := range []mapping.Strategy{BM25Only{}, ContextBM25{}} {
		changes := []domain.Change{{
			ChangeType: domain.ModifyGatewayCondition,
			NewElement: &domain.Element{ID: "gateway", Type: "exclusiveGateway", Name: "alasan dan versi"},
		}}
		mapped, _, err := strategy.Attach(context.Background(), changes, artifacts, Config{TopK: 5})
		if err != nil || len(mapped[0].Impacts) != 1 || mapped[0].Impacts[0].File != "form.tsx" {
			t.Fatalf("%T matched %v, error %v", strategy, mapped, err)
		}
	}
}
