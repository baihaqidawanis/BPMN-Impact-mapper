// Package review reconciles independent reviewer annotations without silently
// resolving conflicts or exposing them to inference.
package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

type Annotation struct {
	Version    string            `json:"version"`
	ReviewerID string            `json:"reviewer_id"`
	Cases      []evaluation.Case `json:"cases"`
}

type Disagreement struct {
	ChangeType string                      `json:"change_type"`
	ElementID  string                      `json:"element_id"`
	ReviewerA  []evaluation.ExpectedImpact `json:"reviewer_a"`
	ReviewerB  []evaluation.ExpectedImpact `json:"reviewer_b"`
	Reason     string                      `json:"reason"`
}

type Reconciliation struct {
	Version       string                 `json:"version"`
	ReviewerA     string                 `json:"reviewer_a"`
	ReviewerB     string                 `json:"reviewer_b"`
	ReviewPack    *PackProvenance        `json:"review_pack,omitempty"`
	Agreed        evaluation.GroundTruth `json:"agreed_ground_truth"`
	Disagreements []Disagreement         `json:"disagreements"`
}

type PackProvenance struct {
	PackID           string `json:"pack_id"`
	InputFingerprint string `json:"input_fingerprint"`
	CaseDigest       string `json:"case_digest"`
}

func Reconcile(a, b Annotation) (Reconciliation, error) {
	if err := validate(a); err != nil {
		return Reconciliation{}, fmt.Errorf("reviewer A: %w", err)
	}
	if err := validate(b); err != nil {
		return Reconciliation{}, fmt.Errorf("reviewer B: %w", err)
	}
	if a.ReviewerID == b.ReviewerID {
		return Reconciliation{}, errors.New("reviewers must be independent identifiers")
	}
	aCases, bCases := caseMap(a.Cases), caseMap(b.Cases)
	keys := map[string]bool{}
	for key := range aCases {
		keys[key] = true
	}
	for key := range bCases {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	result := Reconciliation{Version: "1", ReviewerA: a.ReviewerID, ReviewerB: b.ReviewerID, Agreed: evaluation.GroundTruth{Version: "1", Cases: []evaluation.Case{}}, Disagreements: []Disagreement{}}
	for _, key := range ordered {
		left, leftOK := aCases[key]
		right, rightOK := bCases[key]
		if leftOK && rightOK && equalExpected(left.Expected, right.Expected) {
			left.Expected = sortedExpected(left.Expected)
			result.Agreed.Cases = append(result.Agreed.Cases, left)
			continue
		}
		changeType, elementID := "", ""
		if leftOK {
			changeType, elementID = string(left.ChangeType), left.ElementID
		} else {
			changeType, elementID = string(right.ChangeType), right.ElementID
		}
		reason := "reviewers disagree on expected impacts"
		if !leftOK || !rightOK {
			reason = "case is missing from one reviewer"
		}
		result.Disagreements = append(result.Disagreements, Disagreement{ChangeType: changeType, ElementID: elementID, ReviewerA: sortedExpected(left.Expected), ReviewerB: sortedExpected(right.Expected), Reason: reason})
	}
	return result, nil
}

func validate(annotation Annotation) error {
	if annotation.Version != "1" || annotation.ReviewerID == "" {
		return errors.New("version 1 and reviewer_id are required")
	}
	truth := evaluation.GroundTruth{Version: "1", Cases: annotation.Cases}
	return truth.Validate()
}

func caseMap(cases []evaluation.Case) map[string]evaluation.Case {
	result := map[string]evaluation.Case{}
	for _, item := range cases {
		result[string(item.ChangeType)+"|"+item.ElementID] = item
	}
	return result
}

func equalExpected(a, b []evaluation.ExpectedImpact) bool {
	left, _ := json.Marshal(sortedExpected(a))
	right, _ := json.Marshal(sortedExpected(b))
	return string(left) == string(right)
}

func sortedExpected(values []evaluation.ExpectedImpact) []evaluation.ExpectedImpact {
	result := append([]evaluation.ExpectedImpact(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		left := result[i].File + "|" + result[i].Symbol + "|" + result[i].Module + "|" + result[i].Layer + "|" + string(result[i].Action)
		right := result[j].File + "|" + result[j].Symbol + "|" + result[j].Module + "|" + result[j].Layer + "|" + string(result[j].Action)
		return left < right
	})
	return result
}
