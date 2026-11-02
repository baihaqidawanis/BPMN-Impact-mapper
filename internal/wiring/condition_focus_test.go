package wiring

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
)

type focusProbe struct{ seen *domain.ChangeContext }

func (*focusProbe) Method() domain.AnalysisMethod { return domain.MethodHybrid }
func (p *focusProbe) Attach(_ context.Context, c []domain.Change, _ []domain.Artifact, _ mapping.Options) ([]domain.Change, []domain.Warning, error) {
	p.seen = c[0].Context
	return c, nil, nil
}

func TestConditionFocusIsQueryOnlyAndFallsBackWithoutDelta(t *testing.T) {
	for _, hasDelta := range []bool{true, false} {
		original := &domain.ChangeContext{ProcessName: "Full process", BranchTerms: []string{"Original condition"}, Predecessors: []domain.ElementContext{{ID: "P", Name: "Prior task"}}}
		if hasDelta {
			original.ConditionDelta = []string{"cancelled"}
		}
		before, _ := json.Marshal(original)
		probe := &focusProbe{}
		strategy := contextStudyStrategy{inner: probe, mode: "condition-focus"}
		got, _, err := strategy.Attach(context.Background(), []domain.Change{{ChangeType: domain.ModifyGatewayCondition, Context: original}}, nil, mapping.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if hasDelta {
			if probe.seen.ProcessName != "" || len(probe.seen.Predecessors) != 0 || len(probe.seen.BranchTerms) != 1 || probe.seen.BranchTerms[0] != "cancelled" {
				t.Fatalf("query not focused: %+v", probe.seen)
			}
		} else if probe.seen != original {
			t.Fatal("lost full-context fallback")
		}
		after, _ := json.Marshal(got[0].Context)
		if string(before) != string(after) || got[0].Context != original {
			t.Fatal("canonical context changed by query projection")
		}
	}
	analyzer, err := NewContextStudyAnalyzer(identityOnlyEmbedder{}, "condition-focus")
	if err != nil || !analyzer.ConditionDeltaQueries {
		t.Fatal("experimental annotation disabled")
	}
	production, _ := NewContextStudyAnalyzer(identityOnlyEmbedder{}, "full")
	if production.ConditionDeltaQueries {
		t.Fatal("changed production default")
	}
}
