package relationprobe_test

import (
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/relationprobe"
)

func TestProbeRequiresCitedRouteAndFollowsOnlyOneServiceCall(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "client", File: "ui/request.tsx", Relations: []domain.SourceRelation{{Kind: domain.RelationHTTPRoute, TargetID: "handler", Location: "ui/request.tsx:3", Excerpt: `api("/requests", { method: "POST" })`, TargetLocation: "api/handler.go:2", TargetExcerpt: `r.Post("/requests", h.create)`}}},
		{ID: "handler", File: "api/handler.go", Symbol: domain.Symbol{Name: "create", Receiver: "Handler"}, Relations: []domain.SourceRelation{{Kind: domain.RelationCallsMethod, TargetID: "service", Location: "api/handler.go:9", Excerpt: "h.Service.CreateRequest()"}, {Kind: domain.RelationCalls, TargetID: "other", Location: "api/handler.go:10", Excerpt: "WriteJSON()"}}},
		{ID: "service", File: "app/service.go", Symbol: domain.Symbol{Name: "CreateRequest", Receiver: "Service"}},
		{ID: "other", File: "api/json.go", Symbol: domain.Symbol{Name: "WriteJSON"}},
	}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK-NEW"}, Impacts: []domain.Impact{
		{File: "ui/request.tsx", Evidence: []domain.Evidence{{Type: "BM25", Location: "ui/request.tsx:3", Excerpt: "request form"}}},
		{File: "api/handler.go", Symbol: "Handler.Register"},
		{File: "app/service.go", Symbol: "Service.CreateRequest"},
	}}
	report := domain.Report{Changes: []domain.Change{change}}
	first := relationprobe.Probe(report, artifacts, 1)
	second := relationprobe.Probe(report, artifacts, 1)
	if !reflect.DeepEqual(first, second) || len(first) != 2 {
		t.Fatalf("unexpected/non-deterministic links: %#v", first)
	}
	if first[0].TargetFile != "api/handler.go" || first[0].TargetSymbol != "Handler.create" || first[0].ExistingRank != 2 || len(first[0].Steps) != 2 {
		t.Fatalf("handler link=%#v", first[0])
	}
	if first[1].TargetFile != "app/service.go" || first[1].TargetSymbol != "Service.CreateRequest" || first[1].ExistingRank != 3 || len(first[1].Steps) != 3 {
		t.Fatalf("service link=%#v", first[1])
	}
	proposal := relationprobe.Propose(report, first)
	if len(proposal) != 1 || len(proposal[0].Moves) != 2 || proposal[0].Moves[0].ProposedRank != 2 || proposal[0].Moves[1].ProposedRank != 3 {
		t.Fatalf("relation-order proposal=%#v", proposal)
	}
	report.Changes[0].Impacts[0].Evidence[0].Location = "ui/request.tsx:4"
	if findings := relationprobe.Probe(report, artifacts, 1); len(findings) != 0 {
		t.Fatalf("unrelated source line incorrectly seeded a route: %#v", findings)
	}
}

func TestProposalNeverIntroducesUnrankedFiles(t *testing.T) {
	report := domain.Report{Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW"}, Impacts: []domain.Impact{{File: "ui.go"}, {File: "other.go"}}}}}
	links := []relationprobe.Finding{{ChangeType: domain.AddTask, ElementID: "NEW", SeedFile: "ui.go", TargetFile: "not-ranked.go", Steps: []relationprobe.Step{{Kind: domain.RelationHTTPRoute}}}}
	if proposals := relationprobe.Propose(report, links); len(proposals) != 0 {
		t.Fatalf("unranked file entered proposal: %#v", proposals)
	}
}
