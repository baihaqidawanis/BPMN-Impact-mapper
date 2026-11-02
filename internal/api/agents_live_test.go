package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
)

func TestAgentHTTPLiveSynthetic(t *testing.T) {
	if os.Getenv("BPMN_IMPACT_LAB_WEB_SMOKE") != "1" {
		t.Skip("explicit opt-in lab test")
	}
	root := filepath.Join("..", "..", "outputs", "lab-web-smoke-"+time.Now().UTC().Format("20060102T150405"))
	repo := filepath.Join(root, "repo")
	data := filepath.Join(root, "state")
	if err := os.MkdirAll(repo, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "source.go"), []byte("package fixture\nfunc Submit(amount int) int { return amount }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reports := filestore.ReportStore{Root: filepath.Join(data, "reports")}
	plans := filestore.PlanStore{Root: filepath.Join(data, "plans")}
	r := domain.Report{AnalysisID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", InputFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Method: domain.MethodHybrid, Repository: domain.Repository{Name: "Synthetic API fixture"}, Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "validate-positive", Name: "Validate positive amount before submit", Type: "userTask"}, Reason: "Reject zero or negative amounts before submission.", Impacts: []domain.Impact{{File: "source.go", Symbol: "Submit", Module: "core", Layer: "application", RequiredAction: domain.ActionVerify, Evidence: []domain.Evidence{{Type: "SOURCE", Location: "source.go:2", Excerpt: "func Submit(amount int) int { return amount }"}}}}}}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := reports.Save(ctx, r); err != nil {
		t.Fatal(err)
	}
	m, err := llm.NewOllamaAgent(llm.OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "qwen3.5:9b-q8_0", ContextSize: 32768})
	if err != nil {
		t.Fatal(err)
	}
	svc := &agentjobs.Service{Model: m, Sessions: filestore.AgentSessionStore{Root: filepath.Join(data, "sessions")}, Plans: plans, Base: ctx, Source: func(c context.Context, path string) (agentjobs.Snapshot, error) {
		if err := agenttools.CheckOutputBoundary(path, data); err != nil {
			return nil, err
		}
		return agenttools.New(c, path, ingest.Options{})
	}}
	h := api.Handler{Agents: svc, Store: reports, PlanStore: plans}
	svc.LogError = func(err error) { t.Logf("agent diagnostic: %v", err) }
	body, _ := json.Marshal(map[string]any{"repository_path": repo, "change_index": 0})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/analyses/"+r.AnalysisID+"/agent-sessions", bytes.NewReader(body)))
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	var start agentjobs.View
	if err := json.Unmarshal(w.Body.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	if err := svc.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/agent-sessions/"+start.SessionID, nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if err := os.WriteFile(filepath.Join(root, "http-result.json"), w.Body.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var done agentjobs.View
	if err := json.Unmarshal(w.Body.Bytes(), &done); err != nil {
		t.Fatal(err)
	}
	t.Logf("Artifacts %s; status=%s turns=%d tools=%d revisions=%d message=%s", root, done.Status, done.Turns, done.Tools, done.Revisions, done.Message)
	if done.Plan == nil || (done.Status != "COMPLETE" && done.Status != "NEEDS_REVIEW") {
		t.Fatal("no saved validated plan")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/plans/"+done.Plan.PlanID+"/change-plan.md", nil))
	if w.Code != 200 {
		t.Fatal("download", w.Code)
	}
}
