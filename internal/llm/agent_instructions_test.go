package llm

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"strings"
	"testing"
)

func TestAgentRoleInstructionsDoNotContradictActions(t *testing.T) {
	for _, role := range []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer} {
		instructions := ollamaAgentInstruction(role)
		if role != planning.Analyst && strings.Contains(instructions, "When remaining_tools is zero") {
			t.Fatal("analyst-only action leaked", role)
		}
		if role == planning.Planner && (!strings.Contains(instructions, "proposed_changes") || !strings.Contains(instructions, "acceptance_checks")) {
			t.Fatal("planner contract absent")
		}
	}
	schema, _ := agentActionSchema(planning.Planner)
	var encoded string
	for _, choice := range schema["oneOf"].([]any) {
		props := choice.(map[string]any)["properties"].(map[string]any)
		if plan, ok := props["plan"].(map[string]any); ok {
			item := plan["properties"].(map[string]any)["plans"].(map[string]any)["items"].(map[string]any)
			required := item["required"].([]string)
			encoded = strings.Join(required, ",")
		}
	}
	if !strings.Contains(encoded, "proposed_changes") || !strings.Contains(encoded, "acceptance_checks") {
		t.Fatal("required plan fields missing", encoded)
	}
}

func TestAgentObservationCompactionRetainsPagingWithoutMutation(t *testing.T) {
	original := []planning.ToolObservation{{Request: planning.ToolRequest{Name: "read_source", File: "file.tsx"}, Result: planning.ToolResult{NextLine: 20, Truncated: true, Candidates: []planning.Candidate{{File: "file.tsx", Evidence: []planning.EvidenceItem{{ID: "source", Excerpt: strings.Repeat("x", 240)}}}}}}}
	compact := compactAgentObservations(original)
	if compact[0].Result.NextLine != 20 || len(compact[0].Result.Candidates[0].Evidence) != 0 || len(original[0].Result.Candidates[0].Evidence) != 1 {
		t.Fatal("compaction lost continuity or mutated session")
	}
}
