package evaluation

import (
	"fmt"
	"sort"
)

// ReviewControlOutcome contains only the exported decision. Nil means no valid
// decision was returned. Reference labels must never enter model input.
type ReviewControlOutcome struct {
	ID       string `json:"id"`
	Approval *bool  `json:"approval"`
}

type ReviewControlSummary struct {
	Expected int      `json:"expected_controls"`
	Valid    int      `json:"valid_decisions"`
	Correct  int      `json:"correct_decisions"`
	Passed   bool     `json:"passed_all_controls"`
	Issues   []string `json:"issues"`
}

// EvaluateReviewControls checks a declared set, not a model's self-assessment.
// Passing these controls is not proof of general plan correctness.
func EvaluateReviewControls(expected map[string]bool, actual []ReviewControlOutcome) ReviewControlSummary {
	out := ReviewControlSummary{Expected: len(expected), Issues: []string{}}
	seen := map[string]bool{}
	answers := map[string]*bool{}
	for _, answer := range actual {
		if _, ok := expected[answer.ID]; !ok {
			out.Issues = append(out.Issues, fmt.Sprintf("Unknown control %q.", answer.ID))
			continue
		}
		if seen[answer.ID] {
			answers[answer.ID] = nil
			out.Issues = append(out.Issues, fmt.Sprintf("Duplicate control %q.", answer.ID))
			continue
		}
		seen[answer.ID] = true
		answers[answer.ID] = answer.Approval
	}
	keys := make([]string, 0, len(expected))
	for id := range expected {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		answer := answers[id]
		if answer == nil {
			out.Issues = append(out.Issues, fmt.Sprintf("Control %q has no unique valid decision.", id))
			continue
		}
		out.Valid++
		if *answer != expected[id] {
			out.Issues = append(out.Issues, fmt.Sprintf("Control %q: expected approval=%t, received %t.", id, expected[id], *answer))
			continue
		}
		out.Correct++
	}
	if len(expected) == 0 {
		out.Issues = append(out.Issues, "No control definitions supplied.")
	}
	sort.Strings(out.Issues)
	out.Passed = out.Expected > 0 && out.Correct == out.Expected && len(out.Issues) == 0
	return out
}
