package retrieval_test

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/retrieval"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/textutil"
	"testing"
)

func TestContextBM25AnchorLabelsOnlyAffectOptInRanking(t *testing.T) {
	change := domain.Change{ChangeType: domain.AddGateway, NewElement: &domain.Element{ID: "NEW", Name: "Decision", Type: "exclusiveGateway"}, Context: &domain.ChangeContext{BaselineAnchors: []domain.BaselineAnchor{{ElementContext: domain.ElementContext{ID: "OLD-ID", Name: "invoice approval"}}}}}
	artifacts := []domain.Artifact{{ID: "generic", File: "generic.txt", Content: "decision", Tokens: textutil.Tokenize("decision")}, {ID: "invoice", File: "invoice.txt", Content: "invoice approval", Tokens: textutil.Tokenize("invoice approval")}}
	for _, enabled := range []bool{false, true} {
		changes, _, err := (retrieval.ContextBM25{UseBaselineAnchors: enabled}).Attach(context.Background(), []domain.Change{change}, artifacts, retrieval.Config{TopK: 2})
		if err != nil {
			t.Fatal(err)
		}
		want := "generic.txt"
		if enabled {
			want = "invoice.txt"
		}
		if len(changes[0].Impacts) == 0 || changes[0].Impacts[0].File != want {
			t.Fatal(enabled, changes)
		}
	}
}
