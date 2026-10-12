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
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
)

func TestReviewPackEndpointIsBlindAndSnapshotBound(t *testing.T) {
	store := filestore.ReportStore{Root: t.TempDir()}
	change := domain.Change{ChangeType: domain.ModifyTaskLabel, OldElement: &domain.Element{ID: "TASK", Name: "Old", Type: "task"}, NewElement: &domain.Element{ID: "TASK", Name: "New", Type: "task"}, Impacts: []domain.Impact{{File: "secret/candidate.go", Confidence: .9}}}
	for index, method := range []domain.AnalysisMethod{domain.MethodBM25, domain.MethodHybrid} {
		report := domain.Report{AnalysisID: strings.Repeat(string(rune('a'+index)), 32), InputFingerprint: "snapshot", Method: method, Repository: domain.Repository{Name: "fixture", Revision: "rev-1"}, Changes: []domain.Change{change}, Warnings: []domain.Warning{}}
		if err := store.Save(context.Background(), report); err != nil {
			t.Fatal(err)
		}
	}
	handler := api.Handler{Store: store}
	post := func(value any) *httptest.ResponseRecorder {
		body, _ := json.Marshal(value)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("POST", "/api/v1/review-packs", bytes.NewReader(body)))
		return response
	}
	input := map[string]any{"analysis_ids": []string{strings.Repeat("a", 32), strings.Repeat("b", 32)}, "input_fingerprint": "snapshot", "reviewer_a": "alice", "reviewer_b": "bob"}
	response := post(input)
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	if bytes.Contains(response.Body.Bytes(), []byte("secret/candidate.go")) || bytes.Contains(response.Body.Bytes(), []byte(`"confidence"`)) || !bytes.Contains(response.Body.Bytes(), []byte("REVIEW_REQUIRED")) || !bytes.Contains(response.Body.Bytes(), []byte(`"pack_id"`)) {
		t.Fatalf("review pack leaked prediction data or lacks provenance: %s", response.Body)
	}
	if second := post(input); second.Body.String() != response.Body.String() {
		t.Fatal("review pack response is not deterministic")
	}
	input["input_fingerprint"] = "other"
	if got := post(input); got.Code != 422 {
		t.Fatal("mismatched snapshot accepted")
	}
	input["input_fingerprint"] = "snapshot"
	input["analysis_ids"] = []string{strings.Repeat("a", 32), strings.Repeat("a", 32)}
	if got := post(input); got.Code != 422 {
		t.Fatal("duplicate method accepted")
	}
	input["analysis_ids"] = []string{strings.Repeat("a", 32)}
	input["reviewer_b"] = "alice"
	if got := post(input); got.Code != 422 {
		t.Fatal("identical reviewer IDs accepted")
	}
}
