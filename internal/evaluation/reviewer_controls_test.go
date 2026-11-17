package evaluation_test

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"testing"
)

func TestReviewControlGateDoesNotConfuseValidJSONWithCorrectDecision(t *testing.T) {
	yes, no := true, false
	expected := map[string]bool{"valid": true, "invalid-type": false, "missing-transfer": false}
	tests := []struct {
		name     string
		outcomes []evaluation.ReviewControlOutcome
		passed   bool
		correct  int
	}{
		{"all-correct", []evaluation.ReviewControlOutcome{{"valid", &yes}, {"invalid-type", &no}, {"missing-transfer", &no}}, true, 3},
		{"rubber-stamp", []evaluation.ReviewControlOutcome{{"valid", &yes}, {"invalid-type", &yes}, {"missing-transfer", &yes}}, false, 1},
		{"truncated-output", []evaluation.ReviewControlOutcome{{"valid", nil}, {"invalid-type", &no}, {"missing-transfer", &no}}, false, 2},
		{"missing-control", []evaluation.ReviewControlOutcome{{"valid", &yes}, {"invalid-type", &no}}, false, 2},
		{"duplicate-control", []evaluation.ReviewControlOutcome{{"valid", &yes}, {"valid", &yes}, {"invalid-type", &no}, {"missing-transfer", &no}}, false, 2},
		{"unknown-control", []evaluation.ReviewControlOutcome{{"valid", &yes}, {"invalid-type", &no}, {"missing-transfer", &no}, {"other", &yes}}, false, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluation.EvaluateReviewControls(expected, tt.outcomes)
			if got.Passed != tt.passed || got.Correct != tt.correct {
				t.Fatal(got)
			}
		})
	}
	if empty := evaluation.EvaluateReviewControls(nil, nil); empty.Passed {
		t.Fatal("empty control suite passed")
	}
}
