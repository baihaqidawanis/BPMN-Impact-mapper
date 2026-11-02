package relationprobe

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"reflect"
	"testing"
)

func callerFixture() (domain.Change, []domain.Artifact) {
	c := domain.Change{ChangeType: domain.ModifyGatewayCondition, NewElement: &domain.Element{ID: "G", Name: "Copy invoice allowed?"}, Context: &domain.ChangeContext{BranchTerms: []string{"status rejected cancelled"}}, Impacts: []domain.Impact{{File: "store.go", Confidence: .8, RequiredAction: domain.ActionVerify}}}
	a := []domain.Artifact{
		{ID: "store", File: "store.go", Symbol: domain.Symbol{Name: "CopyInvoice", Kind: "method"}},
		{ID: "service", File: "service.go", Relations: []domain.SourceRelation{{Kind: domain.RelationConstructorCall, TargetID: "store", Location: "service.go:3", Excerpt: "s.store.CopyInvoice(id)", BindingLocation: "main.go:2", BindingExcerpt: "Build(Store{})", TargetLocation: "service.go:1", TargetExcerpt: "Service{store}"}}},
		{ID: "handler", File: "handler.go", Relations: []domain.SourceRelation{{Kind: domain.RelationCallsMethod, TargetID: "service", Location: "handler.go:9", Excerpt: "service.CopyInvoice(id)"}}},
		{ID: "client", File: "ui/panel.tsx", Module: "ui", Layer: "frontend", Content: "api(`/invoices/${id}/${action}`, {method:'POST'})\nif (status === 'REJECTED') copyInvoice();", Relations: []domain.SourceRelation{{Kind: domain.RelationHTTPPattern, TargetID: "handler", Location: "ui/panel.tsx:1", Excerpt: "api(`/invoices/${id}/${action}`, {method:'POST'})", TargetLocation: "handler.go:1", TargetExcerpt: "r.Post('/invoices/{id}/{action}',h.run)"}}},
	}
	return c, a
}

func TestCallerFilesFindThreeEdgeClientWithoutChangingRankedImpacts(t *testing.T) {
	c, a := callerFixture()
	before, _ := json.Marshal(c)
	got := CallerFiles(c, a)
	if len(got) != 1 || got[0].File != "ui/panel.tsx" || got[0].Hops != 3 || len(got[0].Evidence) != 8 {
		t.Fatalf("missing bounded client proof: %+v", got)
	}
	if got[0].Evidence[7].Location != "ui/panel.tsx:2" {
		t.Fatal("missing caller condition excerpt")
	}
	after, _ := json.Marshal(c)
	if string(before) != string(after) {
		t.Fatal("changed retrieval/action")
	}
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
	if !reflect.DeepEqual(got, CallerFiles(c, a)) {
		t.Fatal("artifact order changed caller result")
	}
}

func TestCallerFilesRejectAmbiguousSeedAndStopBeforeFourthEdge(t *testing.T) {
	c, a := callerFixture()
	a = append(a, domain.Artifact{ID: "other", File: "store.go", Symbol: domain.Symbol{Name: "CopyInvoiceDraft", Kind: "method"}})
	if len(CallerFiles(c, a)) != 0 {
		t.Fatal("forced an ambiguous method seed")
	}
	c, a = callerFixture()
	a[1].Relations[0].TargetID = "extra"
	a = append(a, domain.Artifact{ID: "extra", File: "extra.go", Relations: []domain.SourceRelation{{Kind: domain.RelationCallsMethod, TargetID: "store"}}})
	if len(CallerFiles(c, a)) != 0 {
		t.Fatal("exceeded three source edges")
	}
}
