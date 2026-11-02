package fusion_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/fusion"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type fakeStrategy struct {
	method   domain.AnalysisMethod
	changes  []domain.Change
	warnings []domain.Warning
	err      error
}

func (fake fakeStrategy) Method() domain.AnalysisMethod { return fake.method }

func (fake fakeStrategy) Attach(context.Context, []domain.Change, []domain.Artifact, mapping.Options) ([]domain.Change, []domain.Warning, error) {
	return fake.changes, fake.warnings, fake.err
}

func TestRRFRewardsAgreementAndMergesEvidence(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "T", Type: "task"}}}
	left := fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("a.go", domain.ActionVerify, "BM25"), impact("b.go", domain.ActionVerify, "BM25"),
	}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("b.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"), impact("c.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"),
	}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodFusion, Left: left, Right: right}

	first, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("rank fusion is not deterministic")
	}
	if first[0].UnresolvedImpact == nil {
		t.Fatal("resolved fusion result must encode unresolved_impacts as an empty array")
	}
	if got := first[0].Impacts[0]; got.File != "b.go" || got.ScoreBreakdown.RRF != 0.9919 || len(got.Evidence) != 3 {
		t.Fatalf("unexpected fused winner: %#v", got)
	}
	if first[0].Impacts[1].File != "a.go" || first[0].Impacts[2].File != "c.go" {
		t.Fatalf("tie order is not deterministic: %#v", first[0].Impacts)
	}
}

func TestProductionHybridAbstainsFromUnsupportedFunctionWithoutChangingFileRank(t *testing.T) {
	file := "billing/store.go"
	change := domain.Change{ChangeType: domain.ModifyGatewayCondition, NewElement: &domain.Element{ID: "GW-1", Name: "Eligible?", Type: "exclusiveGateway"}}
	selected := impact(file, domain.ActionVerify, "BM25")
	selected.Symbol = "Store.PostInvoice"
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
	artifacts := []domain.Artifact{
		{ID: "post", File: file, Symbol: domain.Symbol{Name: "PostInvoice", Receiver: "Store", Kind: "method"}, Content: "func (Store) PostInvoice() { /* invoice amount */ }"},
		{ID: "approve", File: file, Symbol: domain.Symbol{Name: "ApproveInvoice", Receiver: "Store", Kind: "method"}, Content: "func (Store) ApproveInvoice() { /* eligibility */ }"},
	}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, DiversifyFiles: true, ConservativeSymbols: true}
	result, _, err := strategy.Attach(context.Background(), []domain.Change{change}, artifacts, mapping.Options{TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	got := result[0].Impacts[0]
	if got.File != file || got.Symbol != "" || got.Confidence != 1 || got.RequiredAction != domain.ActionVerify {
		t.Fatalf("unsupported function was asserted or file rank changed: %#v", got)
	}
	if !evidenceType(got.Evidence, "SYMBOL_SCOPE_UNRESOLVED") {
		t.Fatalf("missing function-level abstention evidence: %#v", got.Evidence)
	}
	if strategy.MethodMetadata()["symbol_selection"] == "" {
		t.Fatal("symbol-selection rule not recorded in method metadata")
	}
}

func TestProductionHybridKeepsUniquelySupportedFunction(t *testing.T) {
	for _, test := range []struct {
		name            string
		label           string
		id              string
		selectedContent string
		otherContent    string
		selectedName    string
		otherName       string
		wantSymbol      string
	}{
		{"direct ID", "Eligible?", "GW-1", `func Approve() { trace := "GW-1" }`, `func Reject() {}`, "Approve", "Reject", "Approve"},
		{"two unique label words", "Approve expense claim", "GW-1", `func ApproveClaim() {}`, `func RejectClaim() {}`, "ApproveClaim", "RejectClaim", "ApproveClaim"},
		{"ambiguous label words", "Approve expense claim", "GW-1", `func ApproveClaim() {}`, `func ApproveClaimAgain() {}`, "ApproveClaim", "ApproveClaimAgain", ""},
		{"shared direct ID", "Eligible?", "GW-1", `func Approve() { trace := "GW-1" }`, `func Reject() { trace := "GW-1" }`, "Approve", "Reject", ""},
		{"ID prefix is not exact", "Eligible?", "GW-1", `func Approve() { trace := "GW-12" }`, `func Reject() {}`, "Approve", "Reject", ""},
		{"Unicode ID boundary is exact", "Eligible?", "GW-1", `func Approve() { trace := "ÄGW-1" }`, `func Reject() {}`, "Approve", "Reject", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := "claims.go"
			selected := impact(file, domain.ActionVerify, "BM25")
			selected.Symbol = test.selectedName
			left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
			right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
			artifacts := []domain.Artifact{
				{ID: "selected", File: file, Symbol: domain.Symbol{Name: test.selectedName, Kind: "function"}, Content: test.selectedContent},
				{ID: "other", File: file, Symbol: domain.Symbol{Name: test.otherName, Kind: "function"}, Content: test.otherContent},
			}
			strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, ConservativeSymbols: true}
			result, _, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.ModifyGatewayCondition, NewElement: &domain.Element{ID: test.id, Name: test.label}}}, artifacts, mapping.Options{TopK: 1})
			if err != nil {
				t.Fatal(err)
			}
			if got := result[0].Impacts[0].Symbol; got != test.wantSymbol {
				t.Fatalf("symbol = %q, want %q", got, test.wantSymbol)
			}
		})
	}
}

func TestProductionHybridDoesNotPresentGoPackageAsFunction(t *testing.T) {
	file := "claims/store.go"
	selected := impact(file, domain.ActionVerify, "BM25")
	selected.Symbol = "claims"
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{selected}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, ConservativeSymbols: true}
	artifacts := []domain.Artifact{{ID: file, File: file, Symbol: domain.Symbol{Name: "claims", Kind: "package"}, Content: "package claims"}}
	result, _, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "NEW", Name: "Check claim"}}}, artifacts, mapping.Options{TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := result[0].Impacts[0].Symbol; got != "" {
		t.Fatalf("package name displayed as affected function: %q", got)
	}
}

func evidenceType(items []domain.Evidence, kind string) bool {
	for _, item := range items {
		if item.Type == kind {
			return true
		}
	}
	return false
}

func TestFusionWarnsWhenSemanticSupportIsSparse(t *testing.T) {
	left := fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("a.go", domain.ActionVerify, "BM25"), impact("b.go", domain.ActionVerify, "BM25"), impact("c.go", domain.ActionVerify, "BM25"),
	}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("b.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodFusion, Left: left, Right: right}
	_, warnings, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.AddTask}}, nil, mapping.Options{TopK: 3})
	if err != nil {
		t.Fatal(err)
	}
	for _, warning := range warnings {
		if warning.Code == "FUSION_SPARSE_SUPPORT" {
			return
		}
	}
	t.Fatal("sparse semantic list was not surfaced")
}

func TestFusionKeepsCodeEvidenceScoreSeparateFromRankAgreement(t *testing.T) {
	evidenceScore := 0.2968
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{{
		File: "a.go", RequiredAction: domain.ActionVerify,
		ScoreBreakdown: domain.ScoreBreakdown{EvidenceScore: &evidenceScore, Total: evidenceScore},
		Evidence:       []domain.Evidence{{Type: "BM25", Location: "a.go:1", Excerpt: "request"}},
	}}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{{
		File: "a.go", ScoreBreakdown: domain.ScoreBreakdown{Semantic: 0.4, Total: 0.4},
		Evidence: []domain.Evidence{{Type: "SEMANTIC_SIMILARITY", Location: "a.go:1", Excerpt: "request"}},
	}}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right}
	changes, _, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.AddTask}}, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	got := changes[0].Impacts[0].ScoreBreakdown
	if got.EvidenceScore == nil || *got.EvidenceScore != evidenceScore || got.Semantic != 0.4 || got.RRF != 1 || got.Total != 1 {
		t.Fatalf("separate scores lost during fusion: %#v", got)
	}
}

func TestGraphAwareHybridPreservesOnlyLeftActions(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.ModifyTaskLabel}}
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{impact("direct.go", domain.ActionModify, "TRACE_ID")}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("semantic.go", domain.ActionModify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, PreserveLeftActions: true}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]domain.RequiredAction{}
	for _, candidate := range changes[0].Impacts {
		actions[candidate.File] = candidate.RequiredAction
	}
	if actions["direct.go"] != domain.ActionModify || actions["semantic.go"] != domain.ActionVerify {
		t.Fatalf("unsafe fused actions: %#v", actions)
	}
}

func TestGraphAwareHybridPrefersGraphCandidateOnEqualRRFScore(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask}}
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{impact("z-graph.go", domain.ActionVerify, "TRACE_ID")}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("a-semantic.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, PreserveLeftActions: true}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if changes[0].Impacts[0].File != "z-graph.go" {
		t.Fatalf("semantic-only tie displaced graph evidence: %#v", changes[0].Impacts)
	}
}

func TestFusionPropagatesFailureAndAbstention(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask}}
	strategy := fusion.Strategy{MethodName: domain.MethodFusion, Left: fakeStrategy{method: domain.MethodBM25, err: errors.New("failed")}, Right: fakeStrategy{method: domain.MethodSemantic}}
	if _, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{}); err == nil {
		t.Fatal("retriever failure was hidden")
	}
	strategy.Left = fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{UnresolvedImpact: []string{"no lexical candidate"}}}}
	strategy.Right = fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{UnresolvedImpact: []string{"no semantic candidate"}}}}
	changes, _, err := strategy.Attach(context.Background(), base, nil, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) != 2 {
		t.Fatalf("missing abstention reasons: %#v", changes[0])
	}
}

func impact(file string, action domain.RequiredAction, evidenceType string) domain.Impact {
	return domain.Impact{
		File: file, RequiredAction: action, Evidence: []domain.Evidence{{Type: evidenceType, Location: file, Excerpt: evidenceType}},
	}
}

type boundedStrategy struct{ fakeStrategy }

func (s boundedStrategy) Attach(_ context.Context, changes []domain.Change, _ []domain.Artifact, options mapping.Options) ([]domain.Change, []domain.Warning, error) {
	result := append([]domain.Change(nil), s.changes...)
	for i := range result {
		if len(result[i].Impacts) > options.TopK {
			result[i].Impacts = result[i].Impacts[:options.TopK]
		}
	}
	return result, nil, nil
}

func TestFusionUsesFullPoolBeforeDisplayTopK(t *testing.T) {
	left := boundedStrategy{fakeStrategy{method: domain.MethodBM25, changes: []domain.Change{{Impacts: []domain.Impact{impact("a.go", domain.ActionVerify, "BM25"), impact("shared.go", domain.ActionVerify, "BM25")}}}}}
	right := boundedStrategy{fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("b.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"), impact("shared.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}}}}}
	strategy := fusion.Strategy{Left: left, Right: right}
	artifacts := make([]domain.Artifact, 3)
	one, _, err := strategy.Attach(context.Background(), []domain.Change{{}}, artifacts, mapping.Options{TopK: 1})
	if err != nil || len(one[0].Impacts) != 1 || one[0].Impacts[0].File != "shared.go" {
		t.Fatalf("lost agreement: %+v %v", one, err)
	}
	many, _, err := strategy.Attach(context.Background(), []domain.Change{{}}, artifacts, mapping.Options{TopK: 3})
	if err != nil || !reflect.DeepEqual(one[0].Impacts[0], many[0].Impacts[0]) {
		t.Fatal("display top-k changed winner or score")
	}
}

func TestProductHybridKeepsBestFunctionPerFileBeforeTopK(t *testing.T) {
	first := impact("backend/store.go", domain.ActionVerify, "BM25")
	first.Symbol = "Store.Create"
	second := impact("backend/store.go", domain.ActionVerify, "BM25")
	second.Symbol = "Store.Copy"
	frontend := impact("frontend/request.tsx", domain.ActionVerify, "BM25")
	left := fakeStrategy{method: domain.MethodHybrid, changes: []domain.Change{{Impacts: []domain.Impact{first, second, frontend}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{first, second, frontend}}}}
	base := []domain.Change{{ChangeType: domain.AddTask}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right, DiversifyFiles: true}
	result, _, err := strategy.Attach(context.Background(), base, make([]domain.Artifact, 3), mapping.Options{TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	got := result[0].Impacts
	if len(got) != 2 || got[0].File != first.File || got[0].Symbol != first.Symbol || got[1].File != frontend.File {
		t.Fatalf("file-diverse selection did not retain the best symbol and next file: %#v", got)
	}
	if got[0].Confidence != 1 || got[1].Confidence >= got[0].Confidence {
		t.Fatalf("RRF scores were altered by display selection: %#v", got)
	}
	strategy.DiversifyFiles = false
	baseline, _, err := strategy.Attach(context.Background(), base, make([]domain.Artifact, 3), mapping.Options{TopK: 2})
	if err != nil {
		t.Fatal(err)
	}
	if baseline[0].Impacts[1].Symbol != second.Symbol {
		t.Fatalf("research fusion should preserve artifact-level ranking: %#v", baseline[0].Impacts)
	}
}

func TestFusionTieWarningUsesDisplayedRRFResults(t *testing.T) {
	base := []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "task"}}}
	left := fakeStrategy{method: domain.MethodHybrid, warnings: []domain.Warning{{Code: "CANDIDATE_TIE", Message: "left candidates tie"}}, changes: []domain.Change{{Impacts: []domain.Impact{
		impact("a.go", domain.ActionVerify, "BM25"), impact("b.go", domain.ActionVerify, "BM25"),
	}}}}
	right := fakeStrategy{method: domain.MethodSemantic, changes: []domain.Change{{Impacts: []domain.Impact{impact("a.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}}}}
	strategy := fusion.Strategy{MethodName: domain.MethodHybrid, Left: left, Right: right}
	_, warnings, err := strategy.Attach(context.Background(), base, make([]domain.Artifact, 2), mapping.Options{TopK: 2})
	if err != nil {
		t.Fatalf("upstream tie incorrectly reported as final tie: %v %#v", err, warnings)
	}
	for _, warning := range warnings {
		if warning.Code == "CANDIDATE_TIE" {
			t.Fatalf("upstream tie incorrectly reported as final tie: %#v", warnings)
		}
	}
	right.changes[0].Impacts = []domain.Impact{impact("b.go", domain.ActionVerify, "SEMANTIC_SIMILARITY"), impact("a.go", domain.ActionVerify, "SEMANTIC_SIMILARITY")}
	strategy.Right = right
	_, warnings, err = strategy.Attach(context.Background(), base, make([]domain.Artifact, 2), mapping.Options{TopK: 2})
	if err != nil || len(warnings) != 1 || warnings[0].Code != "CANDIDATE_TIE" {
		t.Fatalf("final rank tie missing: %v %#v", err, warnings)
	}
}
