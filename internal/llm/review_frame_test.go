package llm

import (
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"testing"
)

func TestReviewFrameSeparatesFuturePlanAndDocumentTestResponsibilities(t *testing.T) {
	turn := reviewTurnFixture()
	turn.Role = planning.Reviewer
	turn.Draft = []planning.FilePlan{{File: "docs/policy.md"}, {File: "adapter/flow_test.go"}, {File: "service.ts"}}
	before, _ := json.Marshal(turn)
	output, err := annotateAgentInput(before, turn)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Frame struct {
			Source string            `json:"source_time"`
			Draft  string            `json:"draft_time"`
			Roles  map[string]string `json:"file_responsibilities"`
		} `json:"review_frame"`
	}
	if err := json.Unmarshal(output, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Frame.Source != "BEFORE" || wire.Frame.Draft != "PROPOSED_AFTER, not implemented" || len(wire.Frame.Roles) != 3 || wire.Frame.Roles["0"] == wire.Frame.Roles["1"] || wire.Frame.Roles["1"] == wire.Frame.Roles["2"] {
		t.Fatal("missing temporal/file distinction", string(output))
	}
	after, _ := json.Marshal(turn)
	if string(before) != string(after) {
		t.Fatal("checkpoint mutated")
	}
	turn.Role = planning.Analyst
	raw, _ := json.Marshal(turn)
	output, _ = annotateAgentInput(raw, turn)
	var analyst map[string]any
	json.Unmarshal(output, &analyst)
	if analyst["review_frame"] != nil {
		t.Fatal("review claims leaked to classification")
	}
}
