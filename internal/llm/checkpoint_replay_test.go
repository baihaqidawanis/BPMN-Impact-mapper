//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalCheckpointTransportReplay(t *testing.T) {
	pattern := os.Getenv("BPMN_REPLAY_PATTERN")
	if pattern == "" {
		t.Skip("opt-in local checkpoint replay; never calls lab")
	}
	files, e := filepath.Glob(pattern)
	if e != nil || len(files) == 0 {
		t.Fatal("no checkpoints")
	}
	type capturedPayload struct {
		Messages []struct{ Content string } `json:"messages"`
		Format   json.RawMessage            `json:"format"`
	}
	captured := make(chan capturedPayload, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			var body capturedPayload
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				http.Error(w, "invalid request", 400)
				return
			}
			captured <- body
			var turn planning.AgentTurn
			json.Unmarshal([]byte(body.Messages[1].Content), &turn)
			content := `{"verification":{"decisions":[]}}`
			if turn.Role == planning.Planner {
				content = `{"plan":{"plans":[]}}`
			}
			if turn.Role == planning.Reviewer {
				content = `{"review":{"approved":true,"issues":[]}}`
				if len(turn.Draft) > 0 {
					raw, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
					content = string(raw)
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "message": map[string]string{"role": "assistant", "content": content}})
		}
	}))
	defer srv.Close()
	m, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, Client: srv.Client()})
	if e := m.Prepare(context.Background()); e != nil {
		t.Fatal(e)
	}
	roles := []planning.AgentRole{""} // empty means exact checkpoint role
	if os.Getenv("BPMN_REPLAY_ROLES") == "all" {
		roles = []planning.AgentRole{planning.Analyst, planning.Planner, planning.Reviewer}
	}
	for _, p := range files {
		for _, role := range roles {
			t.Run(filepath.Base(p)+"/"+string(role), func(t *testing.T) {
				data, _ := os.ReadFile(p)
				var v struct{ Session planning.AgentSession }
				if e := json.Unmarshal(data, &v); e != nil {
					t.Fatal(e)
				}
				s := v.Session
				if role != "" {
					s.Role = role
				}
				_, e := m.Turn(context.Background(), planning.AgentTurn{Role: s.Role, Bundle: s.Bundle, Decisions: s.Decisions, Draft: s.Draft, Feedback: s.Feedback, Observations: s.Observations, RemainingTools: s.Budget.MaxTools - s.Tools, RemainingRevisions: s.Budget.MaxRevisions - s.Revisions})
				if e != nil {
					t.Fatal(e)
				}
				var body capturedPayload
				select {
				case body = <-captured:
				default:
					t.Fatal("no model payload reached stub")
				}
				if len(body.Messages) != 2 {
					t.Fatal("unexpected message structure")
				}
				var wire struct {
					Bundle struct {
						Changes []struct {
							Candidates []struct {
								TargetIndex *int `json:"target_index"`
								Evidence    []struct {
									Index *int `json:"evidence_index"`
								} `json:"evidence"`
							} `json:"candidates"`
						} `json:"changes"`
					} `json:"bundle"`
					Facts []struct {
						ID     string `json:"element_id"`
						Before bool   `json:"before_present"`
						After  bool   `json:"after_present"`
					} `json:"change_facts"`
				}
				if err := json.Unmarshal([]byte(body.Messages[1].Content), &wire); err != nil {
					t.Fatal(err)
				}
				if len(wire.Bundle.Changes) != len(s.Bundle.Changes) {
					t.Fatal("lost selected changes")
				}
				targets, evidence := 0, 0
				for _, c := range wire.Bundle.Changes {
					for _, candidate := range c.Candidates {
						if candidate.TargetIndex == nil || *candidate.TargetIndex != targets {
							t.Fatal("unstable target index")
						}
						targets++
						for i, e := range candidate.Evidence {
							if e.Index == nil || *e.Index != i {
								t.Fatal("unstable evidence index")
							}
							evidence++
						}
					}
				}
				if pc := s.Bundle.ProcessContext; pc != nil && pc.Before != nil && pc.After != nil {
					if len(wire.Facts) != len(s.Bundle.Changes) {
						t.Fatal("missing delta facts")
					}
					for i, c := range s.Bundle.Changes {
						f := wire.Facts[i]
						before := pc.Before.Elements[c.ElementID] != nil || pc.Before.Flows[c.ElementID] != nil
						after := pc.After.Elements[c.ElementID] != nil || pc.After.Flows[c.ElementID] != nil
						if f.ID != c.ElementID || f.Before != before || f.After != after {
							t.Fatal("incorrect before/after fact", f)
						}
					}
				}
				if s.Role == planning.Planner && strings.Contains(string(body.Format), `"action":`) {
					t.Fatal("unrestricted action field returned to model")
				}
				requestBytes := len(body.Messages[0].Content) + len(body.Messages[1].Content) + len(body.Format)
				t.Logf("PAYLOAD_AUDIT role=%s changes=%d targets=%d evidence=%d input_bytes=%d request_bytes=%d reserve_limit=%d", s.Role, len(wire.Bundle.Changes), targets, evidence, len(body.Messages[1].Content), requestBytes, 32768-4096-512)
			})
		}
	}
}
