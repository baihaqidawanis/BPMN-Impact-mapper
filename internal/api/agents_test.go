package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestAgentsDisabledAndStrictRoutes(t *testing.T) {
	h := api.Handler{}
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"POST", "/api/v1/capabilities/agents/check", 200}, {"POST", "/api/v1/analyses/abc/agent-sessions", 503}, {"GET", "/api/v1/agent-sessions/abc", 503}, {"GET", "/api/v1/capabilities/agents/check", 404}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, bytes.NewReader(nil)))
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if tc.path == "/api/v1/capabilities/agents/check" && tc.method == http.MethodPost && !bytes.Contains(w.Body.Bytes(), []byte(`"enabled":false`)) {
			t.Fatal("disabled model advertised as enabled")
		}
	}
}

type apiAgent struct{}

func (apiAgent) Identity() planning.ProviderIdentity {
	return planning.ProviderIdentity{Provider: "scripted", Model: "api-test", PromptVersion: "v1"}
}
func (apiAgent) Prepare(context.Context) error { return nil }
func (apiAgent) Turn(_ context.Context, turn planning.AgentTurn) (planning.AgentReply, error) {
	if turn.Role == planning.Analyst {
		out := planning.VerificationResponse{}
		for _, change := range turn.Bundle.Changes {
			for _, c := range change.Candidates {
				out.Decisions = append(out.Decisions, planning.CandidateDecision{ChangeType: change.ChangeType, ElementID: change.ElementID, File: c.File, Symbol: c.Symbol, Verdict: planning.VerdictReject, Rationale: "No implementation change established in this fixture.", EvidenceIDs: []string{c.Evidence[0].ID}})
			}
		}
		return planning.AgentReply{Verification: &out}, nil
	}
	if turn.Role == planning.Planner {
		return planning.AgentReply{Plan: &planning.LayerPlanningResponse{Plans: []planning.FilePlan{}}}, nil
	}
	return planning.AgentReply{Review: &planning.AgentReview{Approved: true}}, nil
}

type apiSource struct{}

func (apiSource) Identity() string                          { return "api-source" }
func (apiSource) CheckBundle(planning.EvidenceBundle) error { return nil }
func (apiSource) Execute(context.Context, planning.ToolRequest) (planning.ToolResult, error) {
	return planning.ToolResult{}, nil
}

func TestAgentHTTPAsyncPersistenceAndStrictInput(t *testing.T) {
	root := t.TempDir()
	reports := filestore.ReportStore{Root: filepath.Join(root, "reports")}
	plans := filestore.PlanStore{Root: filepath.Join(root, "plans")}
	report := domain.Report{AnalysisID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", InputFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Method: domain.MethodHybrid, Changes: []domain.Change{{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "task", Name: "Validation", Type: "userTask"}, Impacts: []domain.Impact{{File: "source.go", RequiredAction: domain.ActionVerify, Evidence: []domain.Evidence{{Type: "SOURCE", Location: "source.go:1", Excerpt: "source"}}}}}}}
	if err := reports.Save(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	svc := &agentjobs.Service{Model: apiAgent{}, Sessions: filestore.AgentSessionStore{Root: filepath.Join(root, "sessions")}, Plans: plans, Source: func(context.Context, string) (agentjobs.Snapshot, error) { return apiSource{}, nil }}
	h := api.Handler{Agents: svc, Store: reports, PlanStore: plans}
	path := "/api/v1/analyses/" + report.AnalysisID + "/agent-sessions"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", path, bytes.NewBufferString(`{"repository_path":"repo","ground_truth":"no.json"}`)))
	if w.Code != 400 {
		t.Fatal("leakage field accepted", w.Code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(`{"repository_path":"private-local-path","change_index":0}`)).WithContext(ctx)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	cancel()
	if w.Code != 202 {
		t.Fatalf("start %d %s", w.Code, w.Body.String())
	}
	var started agentjobs.View
	if err := json.Unmarshal(w.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	wait, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := svc.Wait(wait); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/agent-sessions/"+started.SessionID, nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("private-local-path")) {
		t.Fatal("absolute source root leaked")
	}
	var done agentjobs.View
	if err := json.Unmarshal(w.Body.Bytes(), &done); err != nil {
		t.Fatal(err)
	}
	if done.Status != "COMPLETE" || done.Plan == nil {
		t.Fatal("job did not survive request cancellation")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/plans/"+done.Plan.PlanID+"/change-plan.md", nil))
	if w.Code != 200 {
		t.Fatal("plan download missing", w.Code)
	}
}
