package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestReviewJSONModeKeepsHostValidation(t *testing.T) {
	turn := reviewTurnFixture()
	good, _ := json.Marshal(reviewAuditFixture(turn, "PASS"))
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"complete", string(good), true},
		{"bare-approval", `{"review":{"approved":true}}`, false},
		{"invented-reference", strings.Replace(string(good), `"evidence_indices":[0]`, `"evidence_indices":[999]`, 1), false},
		{"null-reference", strings.Replace(string(good), `"evidence_indices":[0]`, `"evidence_indices":[null]`, 1), false},
		{"missing-check", strings.Replace(string(good), `"contracts":`, `"unknown":`, 1), false},
		{"invalid-status", strings.Replace(string(good), `"PASS"`, `"MAYBE"`, 1), false},
		{"duplicate-status", strings.Replace(string(good), `"status":"PASS"`, `"status":"REVISE","status":"PASS"`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/tags":
					json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"completion"}}}})
				case "/api/version":
					w.Write([]byte(`{"version":"fixture"}`))
				case "/api/chat":
					var body struct {
						Format   string                     `json:"format"`
						Messages []struct{ Content string } `json:"messages"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Format != "json" || len(body.Messages) != 2 {
						t.Error("explicit JSON transport missing", err)
					} else if !strings.Contains(body.Messages[0].Content, "JSON schema:") || !strings.Contains(body.Messages[0].Content, "evidence_indices") {
						t.Error("model not told required schema")
					}
					json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": tc.content}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			cfg := OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, ReviewJSONMode: true, Client: srv.Client()}
			model, err := NewOllamaAgent(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if err = model.Prepare(context.Background()); err != nil {
				t.Fatal(err)
			}
			reply, err := model.Turn(context.Background(), turn)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t reply=%+v error=%v", tc.valid, reply, err)
			}
			if tc.valid && (reply.Review == nil || !reply.Review.Approved || len(reply.Review.Findings) != len(turn.Draft)) {
				t.Fatal("canonical host-bound review missing", reply)
			}
			cfg.ReviewJSONMode = false
			baseline, err := NewOllamaAgent(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if baseline.Identity() == model.Identity() || !strings.HasPrefix(model.Identity().PromptVersion, "ollama-agent-json-v28-") {
				t.Fatal("output policies share identity")
			}
		})
	}
}

func TestImplementationReviewDraftKeepsEditsAndNoActionWithoutMutating(t *testing.T) {
	original := []planning.FilePlan{{Action: domain.ActionVerify, Summary: "Intent", CurrentBehavior: "Planner claim", TargetBehavior: "Desired", DesignRationale: "Self-justification", ProposedChanges: []string{"Keep the actual dispatch unchanged."}, Steps: []string{"Add validation."}, AcceptanceChecks: []string{"Desired payload must reach consumer."}}, {Action: domain.ActionNoAction, CurrentBehavior: "Existing contract is adequate."}}
	turn := reviewTurnFixture()
	original[0].File = turn.Draft[0].File
	original[0].Symbol = turn.Draft[0].Symbol
	original[0].ElementID = turn.Draft[0].ElementID
	original[0].ChangeType = turn.Draft[0].ChangeType
	original = append(original, planning.FilePlan{File: "contract-only.ts", Action: domain.ActionVerify, TargetBehavior: "Keep this requirement in the final cross-file review.", Summary: "Proposal contract"})
	turn.Draft = original
	before, _ := json.Marshal(original)
	projected := implementationReviewDraft(turn)
	if projected[0].Summary != "" || projected[0].CurrentBehavior != original[0].CurrentBehavior || projected[0].TargetBehavior != original[0].TargetBehavior || projected[0].DesignRationale != "" || len(projected[0].ProposedChanges) != 1 || len(projected[0].Steps) != 1 || len(projected[0].AcceptanceChecks) != 1 || projected[1].CurrentBehavior == "" {
		t.Fatal("lost concrete edits/checks or retained Planner assertions", projected)
	}
	if projected[2].TargetBehavior != original[2].TargetBehavior || projected[2].Summary != original[2].Summary {
		t.Fatal("proposal-only final contract requirements removed", projected[2])
	}
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("checkpoint plan mutated")
	}
}
