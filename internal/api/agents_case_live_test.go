//go:build labdiagnostics

package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agenttools"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/ingest"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

// Explicit opt-in exercises the real HTTP handler, job service, source snapshot,
// role adapters, saved analysis/BPMN pair and plan export. It starts no BE/FE
// server, runs no analyzed source, and never loads a reference answer file.
func TestAgentHTTPLiveSavedCase(t *testing.T) {
	if os.Getenv("BPMN_IMPACT_LAB_CASE_REVIEW") != "1" {
		t.Skip("explicit live saved-case review opt-in required")
	}
	analysisID, repo := os.Getenv("BPMN_LAB_ANALYSIS_ID"), os.Getenv("BPMN_LAB_REPO")
	if analysisID == "" || repo == "" {
		t.Fatal("explicit saved analysis and approved repository required")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".data", "model-probes", "pipeline-"+time.Now().UTC().Format("20060102T150405")))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("Private API pipeline artifacts:", root)
	resumeID := os.Getenv("BPMN_LAB_RESUME_SESSION")
	if resumeID != "" {
		previous, err := filepath.Abs(os.Getenv("BPMN_LAB_RESUME_ROOT"))
		if err != nil || filepath.Dir(previous) != filepath.Dir(root) || previous == root {
			t.Fatal("resume root must be a previous private pipeline directory")
		}
		entries, err := os.ReadDir(filepath.Join(previous, "sessions"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "sessions"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(previous, "sessions", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "sessions", entry.Name()), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		t.Log("Resuming copied checkpoint", resumeID, "from", previous)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	reports := filestore.ReportStore{Root: filepath.Join("..", "..", ".data", "analyses")}
	report, err := reports.Get(ctx, analysisID)
	if err != nil {
		t.Fatal(err)
	}
	if report.ProcessContext == nil || report.ProcessContext.Before == nil || report.ProcessContext.After == nil {
		t.Fatal("saved before/after BPMN graphs required")
	}
	t.Logf("Before=%d nodes/%d flows; after=%d nodes/%d flows; changes=%d", len(report.ProcessContext.Before.Elements), len(report.ProcessContext.Before.Flows), len(report.ProcessContext.After.Elements), len(report.ProcessContext.After.Flows), len(report.Changes))
	routes := map[planning.AgentRole]llm.PreparedAgent{}
	var metricsMu sync.Mutex
	onInference := func(metric llm.InferenceMetric) {
		metricsMu.Lock()
		defer metricsMu.Unlock()
		f, err := os.OpenFile(filepath.Join(root, "inference-metrics.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Error(err)
			return
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(metric); err != nil {
			t.Error(err)
		}
		t.Logf("INFERENCE role=%s model=%s outcome=%s ms=%d load_ms=%d input=%d generated=%d", metric.Role, metric.Model, metric.Outcome, metric.Milliseconds, metric.LoadMilliseconds, metric.PromptTokens, metric.GeneratedTokens)
	}
	for role, env := range map[planning.AgentRole]string{planning.Analyst: "BPMN_LAB_ANALYST", planning.Planner: "BPMN_LAB_PLANNER", planning.Reviewer: "BPMN_LAB_REVIEWER"} {
		name := os.Getenv(env)
		if name == "" {
			t.Fatal("explicit model for each role required", env)
		}
		profile := os.Getenv(env + "_PROFILE")
		if profile == "" {
			profile = os.Getenv("BPMN_PROBE_PROFILE")
		}
		model, err := llm.NewOllamaAgent(llm.OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: name, ContextSize: 32768, MaxOutput: 4096, Profile: profile, ReviewComparisons: os.Getenv("BPMN_LAB_REVIEW_COMPARISONS") == "1", OnInference: onInference})
		if err != nil {
			t.Fatal(err)
		}
		routes[role] = model
	}
	model, err := llm.NewRoleRouter(routes)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		manifest, _ := json.MarshalIndent(map[string]any{"analysis_id": analysisID, "input_fingerprint": report.InputFingerprint, "agent_protocol": planning.AgentProtocolVersion, "focused_protocol": planning.FocusedProtocolVersion, "role_models": model.RoleIdentities(), "default_profile": os.Getenv("BPMN_PROBE_PROFILE"), "analyst_profile_override": os.Getenv("BPMN_LAB_ANALYST_PROFILE"), "mode": "focused", "context": 32768, "output_budget": 4096, "thinking": false, "comparison_audit": os.Getenv("BPMN_LAB_REVIEW_COMPARISONS") == "1", "resumed_from": os.Getenv("BPMN_LAB_RESUME_ROOT"), "resumed_session": resumeID}, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifest, 0600); err != nil {
			t.Error(err)
		}
	}()
	plans := filestore.PlanStore{Root: filepath.Join(root, "plans")}
	svc := &agentjobs.Service{Model: model, Sessions: filestore.AgentSessionStore{Root: filepath.Join(root, "sessions")}, Plans: plans, Base: ctx, Source: func(c context.Context, path string) (agentjobs.Snapshot, error) {
		if err := agenttools.CheckOutputBoundary(path, root); err != nil {
			return nil, err
		}
		return agenttools.New(c, path, ingest.Options{})
	}}
	svc.LogError = func(err error) { t.Log("Pipeline diagnostic:", err) }
	h := api.Handler{Agents: svc, Store: reports, PlanStore: plans}
	body, _ := json.Marshal(map[string]any{"repository_path": repo, "change_index": -1, "mode": "focused", "session_id": resumeID})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/analyses/"+analysisID+"/agent-sessions", bytes.NewReader(body)))
	if w.Code != http.StatusAccepted {
		t.Fatal(w.Code, w.Body.String())
	}
	var start agentjobs.View
	if err = json.Unmarshal(w.Body.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	t.Log("Session", start.SessionID)
	done := make(chan error, 1)
	go func() { done <- svc.Wait(ctx) }()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
wait:
	for {
		select {
		case err = <-done:
			if err != nil {
				t.Log("Wait:", err)
			}
			break wait
		case <-ticker.C:
			v, e := svc.Get(ctx, start.SessionID)
			if e == nil {
				t.Logf("PROGRESS status=%s role=%s turns=%d tools=%d revisions=%d %s", v.Status, v.Role, v.Turns, v.Tools, v.Revisions, v.Message)
			}
		case <-ctx.Done():
			t.Fatal("bounded case review exceeded duration; checkpoints retained")
		}
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/agent-sessions/"+start.SessionID, nil))
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = os.WriteFile(filepath.Join(root, "http-result.json"), w.Body.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	var result agentjobs.View
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	t.Logf("RESULT status=%s turns=%d tools=%d revisions=%d issues=%v", result.Status, result.Turns, result.Tools, result.Revisions, result.Issues)
	if result.Plan == nil {
		t.Fatalf("no final plan; preserved checkpoint for diagnosis: %s", result.Message)
	}
	for _, format := range []string{"json", "md"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/plans/"+result.Plan.PlanID+"/change-plan."+format, nil))
		if w.Code != http.StatusOK {
			t.Fatal("plan export", w.Code)
		}
		if err = os.WriteFile(filepath.Join(root, "change-plan."+format), w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// COMPLETE is still a model judgment. This test does not calculate accuracy
	// or claim a NEEDS_REVIEW draft is an approved developer plan.
}
