package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestContractOnlyReviewUsesConstrainedSchemaAndStillRejectsInventedEvidence(t *testing.T) {
	turn := reviewTurnFixture()
	turn.Bundle.Changes[0].Candidates = nil
	for _, bad := range []bool{false, true} {
		body, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
		if bad {
			body = []byte(strings.Replace(string(body), `"evidence_indices":[]`, `"evidence_indices":[0]`, 1))
		}
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
				return
			}
			if r.URL.Path == "/api/version" {
				w.Write([]byte(`{"version":"fixture"}`))
				return
			}
			calls++
			var request struct {
				Format   map[string]any
				Messages []struct{ Content string }
			}
			if r.URL.Path != "/api/chat" {
				t.Error("unexpected request", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			if request.Format["type"] != "object" || len(request.Messages) != 2 || !strings.Contains(request.Messages[0].Content, "CONTRACT-ONLY PASS") {
				t.Error("contract review not explicitly constrained")
			}
			// No source index can be selected. Required checks remain in the schema.
			encoded, _ := json.Marshal(request.Format)
			if !strings.Contains(string(encoded), `"maxItems":0`) || !strings.Contains(string(encoded), `"failure_tests"`) {
				t.Error("missing zero-source/check constraints")
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": string(body)}})
		}))
		m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, ReviewJSONMode: true, Client: srv.Client()})
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Prepare(context.Background()); err != nil {
			t.Fatal(err)
		}
		reply, err := m.Turn(context.Background(), turn)
		srv.Close()
		if bad {
			if !errors.Is(err, planning.ErrAgentModelOutput) {
				t.Fatal("invented source accepted", err)
			}
		} else if err != nil || reply.Review == nil || !reply.Review.Approved || reply.Review.Findings[0].Basis != "proposal_only" || len(reply.Review.Findings[0].EvidenceIDs) != 0 {
			t.Fatal(reply, err)
		}
		if calls != 1 {
			t.Fatal("unexpected retries", calls)
		}
	}
}

func TestReviewErrorIdentifiesExactMissingCriterion(t *testing.T) {
	turn := reviewTurnFixture()
	raw, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
	raw = []byte(strings.Replace(string(raw), `"contracts":{"finding":"Fixture conclusion, not a live model judgment.","status":"PASS"}`, `"contracts":{"finding":"","status":"PASS"}`, 1))
	_, err := restoreReviewAudit(string(raw), turn)
	if err == nil || !strings.Contains(err.Error(), "plan 0 criterion contracts") {
		t.Fatal("repair feedback omits target criterion", err)
	}
}

func TestSourceReviewRepairUsesSchemaAndDoesNotIgnoreDuplicateFields(t *testing.T) {
	turn := reviewTurnFixture()
	turn.Feedback = []string{"Model response rejected for reviewer: duplicate review JSON field"}
	body, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
	for _, bad := range []bool{false, true} {
		content := string(body)
		if bad {
			content = strings.Replace(content, `"status":"PASS"`, `"status":"REVISE","status":"PASS"`, 1)
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/tags" {
				json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
				return
			}
			if r.URL.Path == "/api/version" {
				w.Write([]byte(`{"version":"fixture"}`))
				return
			}
			var payload struct {
				Format   map[string]any
				Messages []struct{ Content string }
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Format["type"] != "object" {
				t.Error("repair was not schema constrained", err)
			}
			if len(payload.Messages) != 2 || !strings.Contains(payload.Messages[0].Content, "FORMAT REPAIR") {
				t.Error("missing repair instruction")
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": content}})
		}))
		m, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, ReviewJSONMode: true, Client: srv.Client()})
		if err := m.Prepare(context.Background()); err != nil {
			t.Fatal(err)
		}
		_, err := m.Turn(context.Background(), turn)
		srv.Close()
		if bad && !errors.Is(err, planning.ErrAgentModelOutput) || !bad && err != nil {
			t.Fatal("repair weakened duplicate validation", err)
		}
	}
}
