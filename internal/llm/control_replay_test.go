//go:build labdiagnostics

package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
)

// Replays exported decisions only. No model, network, source execution or hidden
// reasoning is used. Strict mode fails the test when any model misses a control.
func TestSavedReviewerControlGate(t *testing.T) {
	paths := os.Getenv("BPMN_REVIEW_CONTROL_RESULTS")
	if paths == "" {
		t.Skip("explicit saved control results required")
	}
	expected := map[string]bool{"source-authored-proposal": true, "invalid-string-nil-guard": false, "missing-producer-payload-edit": false}
	if definitions := os.Getenv("BPMN_REVIEW_CONTROL_DEFINITIONS"); definitions != "" {
		var err error
		expected, err = loadReviewControlDefinitions(definitions)
		if err != nil {
			t.Fatal(err)
		}
	}
	type row struct {
		Model   string                          `json:"model"`
		Study   string                          `json:"study"`
		Summary evaluation.ReviewControlSummary `json:"summary"`
	}
	var summaries []row
	for _, path := range strings.Split(paths, ";") {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var results []roleProbeResult
		if err = json.Unmarshal(data, &results); err != nil {
			t.Fatal(err)
		}
		models := map[string][]evaluation.ReviewControlOutcome{}
		for _, result := range results {
			if result.Role != "reviewer" {
				continue
			}
			var approval *bool
			if result.Error == "" && result.Validation == "ADAPTER_ACCEPTED" && result.Reply.Review != nil {
				value := result.Reply.Review.Approved
				approval = &value
			}
			models[result.Model] = append(models[result.Model], evaluation.ReviewControlOutcome{ID: result.Variant, Approval: approval})
		}
		for model, outcomes := range models {
			summary := evaluation.EvaluateReviewControls(expected, outcomes)
			summaries = append(summaries, row{model, filepath.Base(filepath.Dir(path)), summary})
			t.Logf("model=%s valid=%d/%d correct=%d/%d passed=%t", model, summary.Valid, summary.Expected, summary.Correct, summary.Expected, summary.Passed)
			if !summary.Passed && os.Getenv("BPMN_REVIEW_CONTROL_STRICT") == "1" {
				t.Errorf("Reviewer not qualified on declared controls: %s / %s", model, path)
			}
		}
	}
	if len(summaries) == 0 {
		t.Fatal("no Reviewer controls found")
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Study != summaries[j].Study {
			return summaries[i].Study < summaries[j].Study
		}
		return summaries[i].Model < summaries[j].Model
	})
	if output := os.Getenv("BPMN_REVIEW_CONTROL_OUTPUT"); output != "" {
		data, _ := json.MarshalIndent(summaries, "", "  ")
		if err := os.WriteFile(output, append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

// This file is read by the offline scorer only, never sent to the model.
func loadReviewControlDefinitions(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]*bool
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("control definitions must not be empty")
	}
	expected := make(map[string]bool, len(raw))
	for id, approval := range raw {
		if strings.TrimSpace(id) == "" || approval == nil {
			return nil, fmt.Errorf("control definitions require nonblank IDs and boolean labels")
		}
		expected[id] = *approval
	}
	return expected, nil
}

func TestReviewControlDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		valid      bool
	}{
		{"labels", `{"good@file":true,"bad@file":false}`, true},
		{"empty", `{}`, false},
		{"null-suite", `null`, false},
		{"null-label", `{"bad":null}`, false},
		{"blank-id", `{" ":true}`, false},
		{"nonboolean", `{"bad":"false"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "labels.json")
			if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			labels, err := loadReviewControlDefinitions(path)
			if (err == nil) != tc.valid {
				t.Fatalf("labels=%v error=%v", labels, err)
			}
			if tc.valid && (!labels["good@file"] || labels["bad@file"] || len(labels) != 2) {
				t.Fatalf("labels changed: %v", labels)
			}
		})
	}
}
