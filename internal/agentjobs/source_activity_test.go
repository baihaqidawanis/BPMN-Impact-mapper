package agentjobs

import (
	"context"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"testing"
)

type traceStore map[string]planning.AgentSession

func (s traceStore) Load(_ context.Context, id string) (planning.AgentSession, error) {
	v, ok := s[id]
	if !ok {
		return v, planning.ErrAgentSessionMissing
	}
	return v, nil
}
func (s traceStore) Save(_ context.Context, v planning.AgentSession) error { s[v.ID] = v; return nil }

func TestRestoredFocusedTraceIncludesAllChildLookups(t *testing.T) {
	store := traceStore{"trace": {ID: "trace", Status: "PAUSED", Bundle: planning.EvidenceBundle{Changes: make([]planning.ChangeInput, 2)}},
		"trace-focused-0": {Observations: []planning.ToolObservation{{Request: planning.ToolRequest{Name: "search_source", Query: "reason"}, Result: planning.ToolResult{Candidates: []planning.Candidate{{File: "a.go"}, {File: "a.go"}, {File: "b.go"}}, Truncated: true}}}},
		"trace-focused-1": {Observations: []planning.ToolObservation{{Request: planning.ToolRequest{Name: "read_source", File: "b.go"}, Result: planning.ToolResult{Error: "Source lookup rejected or unavailable."}}}}}
	v, err := (&Service{Sessions: store}).Get(context.Background(), "trace")
	if err != nil || len(v.SourceActivity) != 2 {
		t.Fatal(v, err)
	}
	if v.SourceActivity[0].Tool != "search_source" || len(v.SourceActivity[0].Files) != 2 || !v.SourceActivity[0].Truncated {
		t.Fatal("search metadata lost")
	}
	if v.SourceActivity[1].ChangeIndex != 1 || v.SourceActivity[1].Error == "" || len(v.SourceActivity[1].Files) != 0 {
		t.Fatal("failed read shown as successful access")
	}
}

func TestSingleChangeTraceIncludesAllCandidateBatches(t *testing.T) {
	store := traceStore{"trace": {ID: "trace", Status: "PAUSED", FocusedReview: &planning.FocusedReviewProgress{BatchesTotal: 3}, Bundle: planning.EvidenceBundle{Changes: make([]planning.ChangeInput, 1)}}}
	for _, id := range []string{"trace-focused-0", "trace-focused-1", "trace-focused-2"} {
		store[id] = planning.AgentSession{Bundle: planning.EvidenceBundle{Changes: make([]planning.ChangeInput, 1)}, Observations: []planning.ToolObservation{{Request: planning.ToolRequest{Name: "read_source", File: "source.go"}}}}
	}
	v, err := (&Service{Sessions: store}).Get(context.Background(), "trace")
	if err != nil || len(v.SourceActivity) != 3 {
		t.Fatal("candidate batch trace missing", len(v.SourceActivity), err)
	}
	for _, item := range v.SourceActivity {
		if item.ChangeIndex != 0 {
			t.Fatal("batch index mistaken for BPMN change")
		}
	}
}
