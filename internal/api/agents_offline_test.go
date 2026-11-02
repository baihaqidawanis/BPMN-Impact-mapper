package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Exercises the actual Ollama HTTP adapter, async API, immutable source tools,
// checkpoint store and canonical plan download. No lab/model/dependency needed.
func TestAgentOfflineContractPipeline(t *testing.T) {
	for _, mode := range []string{"full-success", "full-resume", "success", "forged-target", "forged-evidence", "incomplete-plan", "malformed-recovery", "malformed-stop", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			data := filepath.Join(root, "state")
			if err := os.MkdirAll(repo, 0700); err != nil {
				t.Fatal(err)
			}
			source := []byte("package fixture\nfunc Submit(amount int) int { return amount }\n")
			if err := os.WriteFile(filepath.Join(repo, "source.go"), source, 0600); err != nil {
				t.Fatal(err)
			}
			chats, plannerCalls, reviews := 0, 0, 0
			local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/tags":
					json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "offline-fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
					return
				case "/api/version":
					json.NewEncoder(w).Encode(map[string]string{"version": "offline-contract"})
					return
				case "/api/chat":
				default:
					http.NotFound(w, r)
					return
				}
				chats++
				var request struct {
					Messages []struct {
						Role    string
						Content string
					}
					Format map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				if len(request.Messages) != 2 || request.Format == nil {
					t.Error("missing structured contract")
					w.WriteHeader(400)
					return
				}
				var turn planning.AgentTurn
				if err := json.Unmarshal([]byte(request.Messages[1].Content), &turn); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				instruction := request.Messages[0].Content
				if turn.Role != planning.Analyst && strings.Contains(instruction, "When remaining_tools is zero") {
					t.Error("analyst action leaked to another role")
				}
				if mode == "disconnect" || (mode == "full-resume" && chats == 1) {
					w.WriteHeader(503)
					return
				}
				if mode == "malformed-stop" || (mode == "malformed-recovery" && chats == 1) {
					json.NewEncoder(w).Encode(map[string]any{"model": "offline-fixture", "done": true, "done_reason": "stop", "message": map[string]string{"role": "assistant", "content": "{invalid-json"}})
					return
				}
				change := turn.Bundle.Changes[0]
				c := change.Candidates[0]
				refs := []string{c.Evidence[0].ID, c.Evidence[0].ID}
				reply := planning.AgentReply{}
				switch turn.Role {
				case planning.Analyst:
					if turn.RepositoryChunk != nil {
						if turn.RepositoryChunk.Content != string(source) {
							t.Error("full source body omitted from model request")
						}
						reply.Verification = &planning.VerificationResponse{Decisions: []planning.CandidateDecision{{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictRelevant, Rationale: "Input is submitted without validation.", EvidenceIDs: refs}}}
						break
					}
					searched := false
					for _, o := range turn.Observations {
						if o.Request.Name == "search_source" && o.Result.Error == "" {
							searched = true
						}
					}
					if !searched {
						reply.Tool = &planning.ToolRequest{Name: "search_source", Query: "amount"}
					} else if len(turn.Observations) == 1 {
						reply.Tool = &planning.ToolRequest{Name: "inspect_source", File: c.File, Query: change.ElementName}
					} else {
						reply.Verification = &planning.VerificationResponse{Decisions: []planning.CandidateDecision{{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictRelevant, Rationale: "Submit currently returns its input without a positive amount guard.", EvidenceIDs: refs}}}
					}
				case planning.Planner:
					plannerCalls++
					p := planning.FilePlan{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Module: c.Module, Layer: c.Layer, Action: c.SuggestedAction, Summary: "Reject non-positive amounts before submission.", Steps: []string{"Add the guard before the existing submit behavior."}, ProposedChanges: []string{"Reject zero or negative amounts while preserving positive-amount submission."}, AcceptanceChecks: []string{"Zero and negative amounts are rejected; positive amounts retain existing behavior."}, EvidenceIDs: refs}
					if (mode == "success" && plannerCalls == 1) || mode == "incomplete-plan" {
						p.ProposedChanges = nil
						p.AcceptanceChecks = nil
					}
					if mode == "success" && plannerCalls == 2 && len(turn.Feedback) == 0 {
						t.Error("contract repair feedback missing")
					}
					if mode == "success" && plannerCalls == 3 && len(turn.Draft) == 0 {
						t.Error("review revision lost previous draft")
					}
					if mode == "forged-target" {
						p.File = "invented.go"
					}
					if mode == "forged-evidence" {
						p.EvidenceIDs = []string{"fabricated"}
					}
					reply.Plan = &planning.LayerPlanningResponse{Plans: []planning.FilePlan{p}}
				case planning.Reviewer:
					reviews++
					reply.Review = &planning.AgentReview{Approved: true}
					if mode == "success" && reviews == 1 {
						reply.Review = &planning.AgentReview{Issues: []string{"Keep the positive-amount acceptance check explicit."}, RevisionTarget: planning.Planner}
					}
				}
				encoded, _ := json.Marshal(reply)
				json.NewEncoder(w).Encode(map[string]any{"model": "offline-fixture", "done": true, "done_reason": "stop", "message": map[string]string{"role": "assistant", "content": string(encoded)}})
			}))
			defer local.Close()
			model, err := llm.NewOllamaAgent(llm.OllamaAgentConfig{BaseURL: local.URL, Model: "offline-fixture", ContextSize: 32768, Client: local.Client()})
			if err != nil {
				t.Fatal(err)
			}
			reports := filestore.ReportStore{Root: filepath.Join(data, "reports")}
			plans := filestore.PlanStore{Root: filepath.Join(data, "plans")}
			report := domain.Report{AnalysisID: strings.Repeat("a", 32), InputFingerprint: strings.Repeat("b", 32), Method: domain.MethodHybrid, Repository: domain.Repository{Name: "Offline fixture"}, Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "positive-amount", Name: "Validate positive amount", Type: "userTask"}, Reason: "Reject non-positive input before submit.", Impacts: []domain.Impact{{File: "source.go", Symbol: "Submit", Module: "core", Layer: "application", RequiredAction: domain.ActionVerify, Evidence: []domain.Evidence{{Type: "SOURCE", Location: "source.go:2", Excerpt: "func Submit(amount int) int { return amount }"}}}}}}}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := reports.Save(ctx, report); err != nil {
				t.Fatal(err)
			}
			svc := &agentjobs.Service{Model: model, Base: ctx, Sessions: filestore.AgentSessionStore{Root: filepath.Join(data, "sessions")}, Plans: plans, Source: func(c context.Context, path string) (agentjobs.Snapshot, error) {
				return agenttools.New(c, path, ingest.Options{})
			}}
			handler := api.Handler{Agents: svc, Store: reports, PlanStore: plans}
			input := map[string]any{"repository_path": repo, "change_index": 0}
			if mode == "full-success" || mode == "full-resume" {
				input["mode"] = "full"
			}
			body, _ := json.Marshal(input)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/analyses/"+report.AnalysisID+"/agent-sessions", bytes.NewReader(body)))
			if w.Code != 202 {
				t.Fatal(w.Code, w.Body.String())
			}
			var started agentjobs.View
			json.Unmarshal(w.Body.Bytes(), &started)
			if err := svc.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			done, err := svc.Get(ctx, started.SessionID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "full-resume" {
				if done.Status != "PAUSED" || done.FullReview == nil || done.FullReview.ChunksReviewed != 0 {
					t.Fatal("full interruption lost progress", done.Status)
				}
				input["session_id"] = started.SessionID
				resumedBody, _ := json.Marshal(input)
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/analyses/"+report.AnalysisID+"/agent-sessions", bytes.NewReader(resumedBody)))
				if w.Code != 202 {
					t.Fatal("resume failed", w.Code, w.Body.String())
				}
				if err := svc.Wait(ctx); err != nil {
					t.Fatal(err)
				}
				done, err = svc.Get(ctx, started.SessionID)
				if err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "success", "malformed-recovery", "full-success", "full-resume":
				if done.Status != "COMPLETE" || done.Plan == nil {
					t.Fatalf("no complete plan: %+v", done)
				}
				if mode != "full-success" && mode != "full-resume" && done.Tools != 2 {
					t.Fatalf("source unnecessarily reread: %d", done.Tools)
				}
				w = httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/plans/"+done.Plan.PlanID+"/change-plan.md", nil))
				if w.Code != 200 || !strings.Contains(w.Body.String(), "### Proposed changes") || !strings.Contains(w.Body.String(), "### Acceptance checks") {
					t.Fatal("structured plan missing from download", w.Code)
				}
				if mode == "full-success" || mode == "full-resume" {
					if done.Mode != "full" || done.FullReview == nil || done.FullReview.ChunksReviewed != done.FullReview.ChunksTotal || done.FullReview.FilesReviewed != 1 || !strings.Contains(w.Body.String(), "Full repository review coverage") {
						t.Fatal("full coverage missing", done.FullReview)
					}
				}
				// Restart persistence path: getting a saved plan never repeats inference.
				n := chats
				restarted := &agentjobs.Service{Sessions: svc.Sessions, Plans: plans}
				restored, e := restarted.Get(ctx, started.SessionID)
				if e != nil || restored.Plan == nil || chats != n {
					t.Fatal("checkpoint restoration failed", e)
				}
			case "forged-target", "forged-evidence", "incomplete-plan":
				if done.Status != "NEEDS_REVIEW" || done.Plan != nil {
					t.Fatal("fabricated target accepted", done.Status)
				}
			case "malformed-stop":
				if done.Status != "PAUSED" || chats != 2 {
					t.Fatal("output repair not bounded", done.Status, chats)
				}
			case "disconnect":
				if done.Status != "PAUSED" || chats != 1 {
					t.Fatal("network failure not checkpointed", done.Status, chats)
				}
			}
			actual, err := os.ReadFile(filepath.Join(repo, "source.go"))
			if err != nil || !bytes.Equal(actual, source) {
				t.Fatal("source modified")
			}
		})
	}
}
