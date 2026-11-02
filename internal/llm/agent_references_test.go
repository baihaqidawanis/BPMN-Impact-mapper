package llm

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"reflect"
	"testing"
)

func TestAgentEvidenceReferencesAreSetsWithoutTrustPromotion(t *testing.T) {
	got := uniqueAgentReferences([]string{"known", "known", "unknown", "known", "unknown"})
	if !reflect.DeepEqual(got, []string{"known", "unknown"}) {
		t.Fatal(got)
	}
	// Unknown references must survive normalization so the planning validator
	// can reject them rather than silently accepting fabricated evidence.
}

func TestUnreadAgentSourceUsesActualEvidenceLocation(t *testing.T) {
	turn := planning.AgentTurn{Bundle: planning.EvidenceBundle{Changes: []planning.ChangeInput{{Candidates: []planning.Candidate{{File: "panel.tsx", Evidence: []planning.EvidenceItem{{Location: "panel.tsx:16"}}}, {File: "store.go"}}}}}}
	file, index := unreadAgentSource(turn)
	if file != "panel.tsx" || index != 0 || agentSourceLine(turn, file) != 12 {
		t.Fatal(file, index)
	}
	turn.Observations = []planning.ToolObservation{{Request: planning.ToolRequest{Name: "read_source", File: file}, Result: planning.ToolResult{Error: "lookup failed"}}}
	if file, _ := unreadAgentSource(turn); file != "panel.tsx" {
		t.Fatal("failed read counted as inspected")
	}
	turn.Observations[0].Result = planning.ToolResult{Candidates: []planning.Candidate{{File: file}}}
	if file, _ := unreadAgentSource(turn); file != "store.go" {
		t.Fatal("next file not inspected")
	}
	if agentSourceLine(turn, "store.go") != 1 {
		t.Fatal("invalid fallback")
	}
}
