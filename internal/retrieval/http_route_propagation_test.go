package retrieval

import (
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
)

func TestHTTPRoutePropagationRequiresSameClientLineAndStaysVerify(t *testing.T) {
	client := domain.Artifact{ID: "ui/panel.tsx", File: "ui/panel.tsx", Module: "ui", Layer: "frontend",
		Relations: []domain.SourceRelation{{Kind: domain.RelationHTTPRoute, TargetID: "handler", Location: "ui/panel.tsx:2", Excerpt: `api("/requests", { method: "POST" })`, TargetLocation: "api/handler.go:3", TargetExcerpt: `r.Post("/requests", h.create)`}}}
	handler := domain.Artifact{ID: "handler", File: "api/handler.go", Module: "api", Layer: "transport", Symbol: domain.Symbol{Name: "create", Receiver: "Handler", Kind: "method"},
		Relations: []domain.SourceRelation{{Kind: domain.RelationCallsMethod, TargetID: "service", Location: "api/handler.go:8", Excerpt: "h.Service.CreateRequest()"}}}
	service := domain.Artifact{ID: "service", File: "app/service.go", Module: "app", Layer: "application", Symbol: domain.Symbol{Name: "CreateRequest", Receiver: "Service", Kind: "method"}}
	artifacts := []domain.Artifact{client, handler, service}
	seed := artifactImpact{impact: domain.Impact{File: client.File, Symbol: "", Confidence: 0.6, RequiredAction: domain.ActionVerify,
		Evidence: []domain.Evidence{{Type: "BM25", Location: "ui/panel.tsx:2", Excerpt: "request"}}}}
	got := propagateHTTPRoutes([]artifactImpact{seed}, artifacts, nil, 0.2)
	if len(got) != 3 || got[1].impact.Symbol != "Handler.create" || got[1].impact.Confidence != 0.6 || got[1].impact.RequiredAction != domain.ActionVerify || got[2].impact.Confidence != 0.3 {
		t.Fatalf("unexpected route propagation: %+v", got)
	}
	// An independently retrieved service keeps its own score and receives the
	// located call clue; the dependency score never overwrites direct evidence.
	existing := artifactImpact{impact: domain.Impact{File: service.File, Symbol: "Service.CreateRequest", Confidence: 0.28, RequiredAction: domain.ActionVerify}}
	got = propagateHTTPRoutes([]artifactImpact{seed, existing}, artifacts, nil, 0.2)
	if len(got) != 3 || got[1].impact.Confidence != 0.28 || !hasEvidence(got[1].impact, "HANDLER_CALLS") {
		t.Fatalf("service evidence changed score or disappeared: %+v", got)
	}
	seed.impact.Evidence[0].Location = "ui/panel.tsx:1"
	if got = propagateHTTPRoutes([]artifactImpact{seed}, artifacts, []string{"other", "item"}, 0.2); len(got) != 1 {
		t.Fatalf("unrelated line activated route: %+v", got)
	}
	if got = propagateHTTPRoutes([]artifactImpact{seed}, artifacts, []string{"request", "another"}, 0.2); len(got) != 1 {
		t.Fatalf("single generic query word activated route: %+v", got)
	}
	if got = propagateHTTPRoutes([]artifactImpact{seed}, artifacts, []string{"submit", "request", "requests"}, 0.2); len(got) != 1 {
		t.Fatalf("one repeated query concept activated route: %+v", got)
	}
}

func hasEvidence(impact domain.Impact, kind string) bool {
	for _, evidence := range impact.Evidence {
		if evidence.Type == kind {
			return true
		}
	}
	return false
}

func TestRouteQueryWordsRequireTwoDistinctConcepts(t *testing.T) {
	line := `await api("/purchase-requests", { method: "POST" })`
	if !routeMatchesQuery(line, []string{"purchase", "request", "upload"}) {
		t.Fatal("relevant route line was not recognized")
	}
	if routeMatchesQuery(line, []string{"request", "requests"}) || routeMatchesQuery(line, []string{"invoice", "upload"}) {
		t.Fatal("unrelated or duplicate terms activated route")
	}
}
