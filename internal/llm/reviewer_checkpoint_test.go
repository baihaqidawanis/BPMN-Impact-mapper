//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Opt-in replay of a real oversized reviewer input. Replies are scripted: this
// verifies bounded partitioning and exact evidence, not actual model quality.
func TestLocalOversizedReviewerCheckpointThroughEngine(t *testing.T) {
	file := os.Getenv("BPMN_REVIEW_CHECKPOINT")
	if file == "" {
		t.Skip("opt-in local checkpoint; no lab calls")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var captured struct{ Session planning.AgentSession }
	if err = json.Unmarshal(data, &captured); err != nil {
		t.Fatal(err)
	}
	s := captured.Session
	for i := range s.Decisions {
		s.Decisions[i].Verdict = planning.VerdictRelevant
	}
	reviewed := map[string]int{}
	contracts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			var body struct {
				Messages []struct{ Content string }
				Format   map[string]any
			}
			json.NewDecoder(r.Body).Decode(&body)
			var turn planning.AgentTurn
			json.Unmarshal([]byte(body.Messages[1].Content), &turn)
			var action any
			if turn.Role == planning.Planner {
				slots := map[string]any{}
				for _, p := range s.Draft {
					found := false
					for i, target := range turnTargets(turn) {
						if target.candidate.File != p.File || target.candidate.Symbol != p.Symbol {
							continue
						}
						var item map[string]any
						raw, _ := json.Marshal(p)
						json.Unmarshal(raw, &item)
						for _, key := range []string{"change_type", "element_id", "file", "symbol", "module", "layer", "action", "evidence_ids"} {
							delete(item, key)
						}
						refs := []int{}
						for _, id := range p.EvidenceIDs {
							for j, e := range target.candidate.Evidence {
								if id == e.ID {
									refs = append(refs, j)
								}
							}
						}
						if len(refs) != len(p.EvidenceIDs) {
							t.Error("captured evidence was lost")
						}
						item["evidence_indices"] = refs
						slots[strconv.Itoa(i)] = item
						found = true
						break
					}
					if !found {
						t.Error("captured target missing", p.File)
					}
				}
				action = map[string]any{"plan": map[string]any{"plans": slots}}
			} else if turn.Role == planning.Reviewer {
				if len(turn.Feedback) == 0 {
					t.Error("oversized review reached transport without partition context")
				}
				instruction := strings.Join(turn.Feedback, " ")
				if strings.HasPrefix(instruction, "FINAL FILE") {
					if len(turn.Draft) != 1 {
						t.Error("wrong file partition", len(turn.Draft))
					}
					for _, p := range turn.Draft {
						reviewed[p.File]++
					}
				} else if strings.HasPrefix(instruction, "FINAL CONTRACT") {
					contracts++
					if len(turn.Draft) != len(s.Draft) {
						t.Error("contract review lost plans")
					}
				} else {
					t.Error("unexpected review transport", instruction)
				}
				action = reviewAuditFixture(turn, "PASS")
			} else {
				t.Error("unexpected role", turn.Role)
			}
			content, _ := json.Marshal(action)
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "message": map[string]string{"role": "assistant", "content": string(content)}})
		}
	}))
	defer srv.Close()
	model, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, Client: srv.Client()})
	if err = model.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = model.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer, Bundle: s.Bundle, Draft: s.Draft, Feedback: s.Feedback}); !errors.Is(err, planning.ErrAgentContextLimit) {
		t.Fatal("fixture no longer exercises overflow", err)
	}
	engine := planning.AgentEngine{Model: model, Tools: noSourceTools{}, Store: filestore.AgentSessionStore{Root: t.TempDir()}, InitialDecisions: s.Decisions, Budget: planning.AgentBudget{MaxTurns: 18, MaxRevisions: 2}}
	result, err := engine.Run(context.Background(), "captured-review-fixture", s.Bundle)
	if err != nil || result.Status != "COMPLETE" || len(result.Draft) != len(s.Draft) {
		t.Fatal(result.Status, result.Feedback, err)
	}
	if len(reviewed) != len(s.Draft) || contracts != 1 {
		t.Fatal("incomplete review", reviewed, contracts)
	}
	for file, n := range reviewed {
		if n != 1 {
			t.Fatal("file reviewed repeatedly", file, n)
		}
	}
	t.Logf("partitioned %d exact file proposals plus one complete cross-file review; %d budgeted engine turns", len(reviewed), result.Turns)
}
