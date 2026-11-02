// Package selection builds an optional review shortlist. It does not determine
// required source changes or turn retrieval scores into probabilities.
package selection

import (
	"fmt"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"strconv"
	"strings"
)

type Policy struct {
	Name  string `json:"name"`
	Bound int    `json:"rank_bound,omitempty"`
}
type Deferred struct {
	Candidate domain.Impact `json:"candidate"`
	Reason    string        `json:"reason"`
}
type Result struct {
	Selected       []domain.Impact `json:"selected"`
	Deferred       []Deferred      `json:"deferred"`
	DuplicateFiles int             `json:"duplicate_files"`
}

// Apply uses only recorded ranks in the original report. No labels, repository
// reads, new retrieval, confidence thresholds, or required-action reassignment.
func Apply(input []domain.Impact, policy Policy) (Result, error) {
	result := Result{Selected: []domain.Impact{}, Deferred: []Deferred{}}
	if policy.Name != "all" && policy.Name != "both-within" {
		return result, fmt.Errorf("unknown review-selection policy")
	}
	if policy.Name == "both-within" && (policy.Bound < 1 || policy.Bound > 1000) {
		return result, fmt.Errorf("rank bound must be 1..1000")
	}
	seen := map[string]bool{}
	for _, candidate := range input {
		if candidate.File == "" {
			continue
		}
		if seen[candidate.File] {
			result.DuplicateFiles++
			continue
		}
		seen[candidate.File] = true
		if policy.Name == "all" {
			result.Selected = append(result.Selected, candidate)
			continue
		}
		left, right, valid := ranks(candidate)
		if !valid {
			result.Deferred = append(result.Deferred, Deferred{candidate, "two unambiguous branch ranks unavailable"})
			continue
		}
		if left <= policy.Bound && right <= policy.Bound {
			result.Selected = append(result.Selected, candidate)
		} else {
			result.Deferred = append(result.Deferred, Deferred{candidate, "outside frozen agreement bound"})
		}
	}
	return result, nil
}

func ranks(candidate domain.Impact) (int, int, bool) {
	count := 0
	left, right := 0, 0
	valid := false
	for _, e := range candidate.Evidence {
		if e.Type != "RRF_FUSION" {
			continue
		}
		count++
		fields := map[string]string{}
		duplicates := false
		for _, part := range strings.Split(e.Excerpt, ";") {
			pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(pair) != 2 {
				continue
			}
			key, value := strings.TrimSpace(pair[0]), strings.TrimSpace(pair[1])
			if _, exists := fields[key]; exists {
				duplicates = true
			}
			fields[key] = value
		}
		var a, b error
		left, a = strconv.Atoi(fields["left_rank"])
		right, b = strconv.Atoi(fields["right_rank"])
		valid = !duplicates && a == nil && b == nil && left > 0 && right > 0
	}
	return left, right, count == 1 && valid
}
