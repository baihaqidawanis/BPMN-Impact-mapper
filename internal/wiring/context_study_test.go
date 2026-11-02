package wiring

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/mapping"
	"testing"
)

type contextProbe struct{ sawContext bool }

func (*contextProbe) Method() domain.AnalysisMethod { return domain.MethodHybrid }
func (probe *contextProbe) Attach(_ context.Context, changes []domain.Change, _ []domain.Artifact, _ mapping.Options) ([]domain.Change, []domain.Warning, error) {
	probe.sawContext = changes[0].Context != nil
	return changes, nil, nil
}
func TestContextStudyRemovesQueryContextWithoutMutatingReport(t *testing.T) {
	probe := &contextProbe{}
	original := &domain.ChangeContext{ProcessName: "Example process"}
	input := []domain.Change{{Context: original}}
	result, _, err := (contextStudyStrategy{inner: probe, mode: "no-context"}).Attach(context.Background(), input, nil, mapping.Options{})
	if err != nil || probe.sawContext || input[0].Context != original || result[0].Context != original {
		t.Fatal("context ablation did not isolate query transformation")
	}
}
func TestContextStudyIdentitySeparatesModesAndRejectsUnknown(t *testing.T) {
	seen := map[string]bool{}
	for _, mode := range []string{"full", "no-context", "no-start-end", "both-top10", "multi-hop4", "condition-focus", "generic-windows", "code-windows", "source-callers"} {
		analyzer, err := NewContextStudyAnalyzer(identityOnlyEmbedder{}, mode)
		if err != nil {
			t.Fatal(err)
		}
		_, identity, err := analyzer.Resolver.Resolve(domain.MethodHybrid, "")
		if err != nil || seen[string(identity)] {
			t.Fatal("duplicate/missing study identity", err)
		}
		seen[string(identity)] = true
		if (mode == "multi-hop4") != (analyzer.BaselineContextHops == 4) {
			t.Fatal("unexpected baseline expansion default", mode)
		}
	}
	if _, err := NewContextStudyAnalyzer(identityOnlyEmbedder{}, "typo"); err == nil {
		t.Fatal("accepted unknown mode")
	}
}
func TestBothRankFilterRejectsAbsentAndDoesNotTrustHighConfidence(t *testing.T) {
	for _, item := range []struct {
		excerpt string
		want    bool
	}{
		{"left_rank=3; right_rank=2; k=60", true},
		{"left_rank=5; right_rank=17; k=60", false},
		{"left_rank=absent; right_rank=1; k=60", false},
		{"left_rank=0; right_rank=1; k=60", false},
	} {
		impact := domain.Impact{Confidence: 1, Evidence: []domain.Evidence{{Type: "RRF_FUSION", Excerpt: item.excerpt}}}
		if hasBothRanksWithin(impact, 10) != item.want {
			t.Fatal(item.excerpt)
		}
	}
}
