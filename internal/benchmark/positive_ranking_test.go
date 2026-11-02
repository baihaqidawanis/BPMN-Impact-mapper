package benchmark

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"testing"
)

func TestPositiveRankingAggregationWeightsCasesAndSkipsNegativeOnlyScenarios(t *testing.T) {
	m := meanMetrics([]evaluation.Metrics{{PositiveRanking: &evaluation.RankingSummary{Count: 1, MAP: 1, RPrecision: 1}}, {}, {PositiveRanking: &evaluation.RankingSummary{Count: 3, MAP: 0, RPrecision: 0}}})
	if m.PositiveRanking == nil || m.PositiveRanking.Count != 4 || m.PositiveRanking.MAP != 0.25 || m.PositiveRanking.RPrecision != 0.25 {
		t.Fatalf("incorrect positive weighting: %#v", m.PositiveRanking)
	}
}
