package app

import (
	"context"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type allArtifactsStrategy struct {
	method    domain.AnalysisMethod
	calls     int
	seen      []string
	wholeFile bool
}

func (strategy *allArtifactsStrategy) Method() domain.AnalysisMethod { return strategy.method }

func (strategy *allArtifactsStrategy) Attach(_ context.Context, changes []domain.Change, artifacts []domain.Artifact, _ mapping.Options) ([]domain.Change, []domain.Warning, error) {
	strategy.calls++
	for _, artifact := range artifacts {
		strategy.seen = append(strategy.seen, artifact.File)
	}
	for index := range changes {
		for _, artifact := range artifacts {
			symbol := impactSymbol(artifact.Symbol)
			if strategy.wholeFile {
				symbol = ""
			}
			changes[index].Impacts = append(changes[index].Impacts, domain.Impact{
				File: artifact.File, Symbol: symbol, RequiredAction: domain.ActionVerify,
				Evidence: []domain.Evidence{{Type: "TEST_RANKING", Location: artifact.File, Excerpt: "synthetic ranking"}},
			})
		}
	}
	return changes, nil, nil
}

func explicitLabelChange() domain.Change {
	return domain.Change{
		ChangeType: domain.ModifySequenceFlow, Reason: "sequence flow label changed",
		OldElement: &domain.Element{ID: "FLOW-CHECK", Name: "Approved request", Type: "sequenceFlow", Documentation: "request.valid", Incoming: []string{"GATEWAY"}, Outgoing: []string{"APPROVE"}},
		NewElement: &domain.Element{ID: "FLOW-CHECK", Name: "Request accepted", Type: "sequenceFlow", Documentation: "request.valid", Incoming: []string{"GATEWAY"}, Outgoing: []string{"APPROVE"}},
		Context:    &domain.ChangeContext{ProcessName: "Request approval", Predecessors: []domain.ElementContext{{ID: "GATEWAY", Name: "Review request"}}},
	}
}

func TestExplicitFlowLabelWithoutDirectReferenceAbstainsAcrossRetrievers(t *testing.T) {
	artifacts := []domain.Artifact{{ID: "near", File: "request.go", Content: "Review request and approve it. FLOW-CHECK-EXTRA is unrelated."}}
	for _, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion, domain.MethodHybrid} {
		t.Run(string(method), func(t *testing.T) {
			strategy := &allArtifactsStrategy{method: method}
			changes, _, err := attachPrimaryChanges(context.Background(), strategy, []domain.Change{explicitLabelChange()}, artifacts, mapping.Options{TopK: 5})
			if err != nil {
				t.Fatal(err)
			}
			if strategy.calls != 0 || len(changes[0].Impacts) != 0 || len(changes[0].UnresolvedImpact) != 1 {
				t.Fatalf("label-only flow should abstain, got calls=%d change=%#v", strategy.calls, changes[0])
			}
			if !strings.Contains(changes[0].UnresolvedImpact[0], "code impact is unproven") {
				t.Fatalf("missing abstention rationale: %#v", changes[0].UnresolvedImpact)
			}
		})
	}
}

func TestExplicitFlowLabelKeepsOnlyDirectSourceReferences(t *testing.T) {
	artifacts := []domain.Artifact{
		{ID: "label", File: "review.go", Symbol: domain.Symbol{Name: "Review", Kind: "function", StartLine: 10}, Content: "// Approved request is displayed here."},
		{ID: "id", File: "flow.go", Content: "const branch = \"FLOW-CHECK\""},
		{ID: "near", File: "unrelated.go", Content: "Request approval. FLOW-CHECK-EXTRA is unrelated."},
	}
	for _, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodSemantic, domain.MethodFusion, domain.MethodHybrid} {
		t.Run(string(method), func(t *testing.T) {
			strategy := &allArtifactsStrategy{method: method}
			changes, _, err := attachPrimaryChanges(context.Background(), strategy, []domain.Change{explicitLabelChange()}, artifacts, mapping.Options{TopK: 5})
			if err != nil {
				t.Fatal(err)
			}
			if len(strategy.seen) != 2 || strategy.seen[0] != "review.go" || strategy.seen[1] != "flow.go" || len(changes[0].Impacts) != 2 {
				t.Fatalf("wrong eligible set: %#v / %#v", strategy.seen, changes[0].Impacts)
			}
			if changes[0].Impacts[0].Evidence[1].Type != "FLOW_LABEL_TEXT" || changes[0].Impacts[1].Evidence[1].Type != "FLOW_ID_REFERENCE" {
				t.Fatalf("direct reference evidence missing: %#v", changes[0].Impacts)
			}
			if changes[0].Impacts[0].Evidence[1].Location != "review.go:10" {
				t.Fatalf("source line incorrect: %#v", changes[0].Impacts[0].Evidence[1])
			}
		})
	}
}

func TestFlowLabelGateDoesNotHideBehavioralChangesOrConfiguration(t *testing.T) {
	artifacts := []domain.Artifact{{ID: "context", File: "request.go", Content: "Request approval logic"}}
	tests := []struct {
		name   string
		change domain.Change
		method domain.AnalysisMethod
	}{
		{name: "explicit condition changed", change: func() domain.Change {
			c := explicitLabelChange()
			c.NewElement.Documentation = "request.valid && budget.ok"
			return c
		}(), method: domain.MethodHybrid},
		{name: "label is condition surrogate", change: func() domain.Change {
			c := explicitLabelChange()
			c.OldElement.Documentation = ""
			c.NewElement.Documentation = ""
			return c
		}(), method: domain.MethodBM25},
		{name: "endpoint changed", change: func() domain.Change { c := explicitLabelChange(); c.NewElement.Outgoing = []string{"REJECT"}; return c }(), method: domain.MethodSemantic},
		{name: "explicit config mapping", change: explicitLabelChange(), method: domain.MethodConfiguration},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			strategy := &allArtifactsStrategy{method: test.method}
			changes, _, err := attachPrimaryChanges(context.Background(), strategy, []domain.Change{test.change}, artifacts, mapping.Options{})
			if err != nil || strategy.calls != 1 || len(changes[0].Impacts) != 1 {
				t.Fatalf("behavioral/configuration mapping was hidden: changes=%#v calls=%d err=%v", changes, strategy.calls, err)
			}
		})
	}
}

func TestFlowLabelReferenceExcerptContainsLateMatch(t *testing.T) {
	line := strings.Repeat("prefix ", 50) + "Approved request"
	kind, number, excerpt := referenceLine(line, "Approved request", false)
	if kind != "FLOW_LABEL_TEXT" || number != 1 || !strings.Contains(excerpt, "Approved request") || len([]rune(excerpt)) > 240 {
		t.Fatalf("bounded reference lost matching text: %s:%d %q", kind, number, excerpt)
	}
}

func TestSupportingFlowKeepsDirectSourceReferences(t *testing.T) {
	for _, side := range []string{"both", "new", "old"} {
		for _, reference := range []bool{false, true} {
			change := explicitLabelChange()
			change.PartOf = "GATEWAY"
			change.NewElement.Documentation = "budget available"
			if side == "new" {
				change.OldElement = nil
			}
			if side == "old" {
				change.NewElement = nil
			}
			content := "unrelated approval text"
			if reference {
				content = `const flow = "FLOW-CHECK"`
			}
			strategy := &allArtifactsStrategy{method: domain.MethodBM25}
			result, _, err := attachPrimaryChanges(context.Background(), strategy, []domain.Change{change}, []domain.Artifact{{ID: "direct", File: "flow.go", Content: content}}, mapping.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if reference {
				if len(result[0].Impacts) != 1 || result[0].PartOf != "" || result[0].Impacts[0].Evidence[1].Type != "FLOW_ID_REFERENCE" {
					t.Fatalf("%s direct reference hidden: %#v", side, result)
				}
			} else if strategy.calls != 0 || result[0].PartOf != "GATEWAY" {
				t.Fatalf("%s supporting flow retrieved again: %#v", side, result)
			}
		}
	}
}

func TestFlowReferenceSurvivesWholeFileCandidateSelection(t *testing.T) {
	c := explicitLabelChange()
	c.PartOf = "GATEWAY"
	strategy := &allArtifactsStrategy{method: domain.MethodHybrid, wholeFile: true}
	result, _, err := attachPrimaryChanges(context.Background(), strategy, []domain.Change{c}, []domain.Artifact{{ID: "ref", File: "flow.go", Symbol: domain.Symbol{Name: "Apply", Kind: "function", StartLine: 10}, Content: `const id = "FLOW-CHECK"`}}, mapping.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result[0].Impacts) != 1 || result[0].Impacts[0].Evidence[1].Location != "flow.go:10" {
		t.Fatalf("whole-file source reference lost: %#v", result)
	}
}
