package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/evaluation"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/review"
)

func TestEvaluationEndpointsUseStoredReportsAndExplicitDecisions(t *testing.T) {
	store := filestore.ReportStore{Root: t.TempDir()}
	change := domain.Change{ChangeType: domain.AddTask, NewElement: &domain.Element{ID: "TASK", Type: "task"}, Impacts: []domain.Impact{{File: "service.go", RequiredAction: domain.ActionModify}}, UnresolvedImpact: []string{}}
	prediction := domain.Report{AnalysisID: strings.Repeat("a", 32), InputFingerprint: "snapshot", Method: domain.MethodHybrid, Changes: []domain.Change{change}, Warnings: []domain.Warning{}}
	if err := store.Save(context.Background(), prediction); err != nil {
		t.Fatal(err)
	}
	original, _ := store.Get(context.Background(), prediction.AnalysisID)
	// No analyzer is configured: evaluation cannot re-run inference.
	handler := api.Handler{Store: store}
	expected := evaluation.Case{ChangeType: domain.AddTask, ElementID: "TASK", Expected: []evaluation.ExpectedImpact{{File: "service.go", Action: domain.ActionModify}}}
	a := review.Annotation{Version: "1", ReviewerID: "a", Cases: []evaluation.Case{expected}}
	b := review.Annotation{Version: "1", ReviewerID: "b", Cases: []evaluation.Case{{ChangeType: domain.AddTask, ElementID: "TASK", Expected: []evaluation.ExpectedImpact{}}}}
	post := func(route string, value any) *httptest.ResponseRecorder {
		t.Helper()
		body, _ := json.Marshal(value)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", route, bytes.NewReader(body)))
		return response
	}
	pair := map[string]any{"reviewer_a": a, "reviewer_b": b}
	rec := post("/api/v1/reviews/reconcile", pair)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "reviewers disagree") {
		t.Fatalf("reconcile: %d %s", rec.Code, rec.Body)
	}
	input := map[string]any{"reviewer_a": a, "reviewer_b": b, "analysis_ids": []string{prediction.AnalysisID}, "input_fingerprint": "snapshot", "snapshot_confirmed": true, "resolutions": []evaluation.Case{}}
	if got := post("/api/v1/evaluations", input); got.Code != 422 {
		t.Fatalf("unresolved accepted: %d", got.Code)
	}
	input["resolutions"] = []evaluation.Case{expected}
	good := post("/api/v1/evaluations", input)
	if good.Code != 200 {
		t.Fatalf("evaluation: %d %s", good.Code, good.Body)
	}
	var result struct {
		Results  []evaluation.Report `json:"results"`
		Markdown string              `json:"markdown"`
	}
	if err := json.Unmarshal(good.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Results) != 1 || result.Results[0].Metrics.MRR != 1 || result.Results[0].Metrics.ActionAccuracy != 1 || result.Markdown == "" {
		t.Fatalf("wrong metrics: %+v", result)
	}
	if second := post("/api/v1/evaluations", input); second.Body.String() != good.Body.String() {
		t.Fatal("evaluation not deterministic")
	}
	after, _ := store.Get(context.Background(), prediction.AnalysisID)
	beforeBytes, _ := json.Marshal(original)
	afterBytes, _ := json.Marshal(after)
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("prediction mutated")
	}
	input["input_fingerprint"] = "other"
	if got := post("/api/v1/evaluations", input); got.Code != 422 {
		t.Fatal("mismatch accepted")
	}
	input["input_fingerprint"] = "snapshot"
	input["analysis_ids"] = []string{prediction.AnalysisID, prediction.AnalysisID}
	if got := post("/api/v1/evaluations", input); got.Code != 422 {
		t.Fatal("duplicate method accepted")
	}
	input["analysis_ids"] = []string{prediction.AnalysisID}
	input["snapshot_confirmed"] = false
	if got := post("/api/v1/evaluations", input); got.Code != 422 {
		t.Fatal("unconfirmed snapshot accepted")
	}
	a.Cases = nil
	b.Cases = nil
	input["reviewer_a"] = a
	input["reviewer_b"] = b
	input["snapshot_confirmed"] = true
	input["resolutions"] = []evaluation.Case{}
	if got := post("/api/v1/evaluations", input); got.Code != 422 {
		t.Fatal("empty coverage accepted")
	}
}

func TestReviewBoundaryRejectsMalformedUnboundedAndUnknownInput(t *testing.T) {
	for _, body := range []string{
		"{} {}", `{"ground_truth":"secret"}`, strings.Repeat(" ", 2<<20) + "{}",
		`{"reviewer_a":{"version":"1","reviewer_id":"a","cases":[]},"reviewer_b":{"version":"1","reviewer_id":"a","cases":[]}}`,
	} {
		response := httptest.NewRecorder()
		(api.Handler{}).ServeHTTP(response, httptest.NewRequest("POST", "/api/v1/reviews/reconcile", strings.NewReader(body)))
		if response.Code < 400 {
			t.Fatalf("invalid review accepted: %d", response.Code)
		}
	}
}
