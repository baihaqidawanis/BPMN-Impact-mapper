package retrieval

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"testing"
)

func TestPropagationIsOneHopAndVerifyOnly(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "seed", File: "seed.go", Symbol: domain.Symbol{Name: "Seed", Kind: "function"}, Relations: []domain.SourceRelation{{Kind: domain.RelationCalls, TargetID: "child", Location: "seed.go:4", Excerpt: "Child()"}}},
		{ID: "child", File: "child.go", Symbol: domain.Symbol{Name: "Child", Kind: "function"}, Relations: []domain.SourceRelation{{Kind: domain.RelationCalls, TargetID: "grandchild", Location: "child.go:2", Excerpt: "Grandchild()"}}},
		{ID: "grandchild", File: "grand.go", Symbol: domain.Symbol{Name: "Grandchild", Kind: "function"}},
		{ID: "caller", File: "caller.go", Symbol: domain.Symbol{Name: "Caller", Kind: "function"}, Relations: []domain.SourceRelation{{Kind: domain.RelationCalls, TargetID: "seed", Location: "caller.go:3", Excerpt: "Seed()"}}},
	}
	seed := artifactImpact{symbolKind: "function", impact: domain.Impact{File: "seed.go", Symbol: "Seed", Confidence: 0.8, ScoreBreakdown: domain.ScoreBreakdown{TraceID: 1, Total: 0.8}}}
	result := propagate([]artifactImpact{seed}, artifacts, 0.2)
	if len(result) != 3 {
		t.Fatalf("expected seed and two one-hop candidates: %+v", result)
	}
	for _, r := range result[1:] {
		if r.impact.RequiredAction != domain.ActionVerify || r.impact.Confidence != 0.4 || len(r.impact.Evidence) != 2 || r.impact.File == "grand.go" {
			t.Fatalf("invalid propagation %+v", r)
		}
	}
	seed.impact.ScoreBreakdown.TraceID = 0
	if len(propagate([]artifactImpact{seed}, artifacts, 0.2)) != 1 {
		t.Fatal("lexical seed propagated")
	}
}
