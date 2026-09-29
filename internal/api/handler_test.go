package api_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/app"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/domain"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/wiring"
)

func TestHandlerReturnsCanonicalReport(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	input := app.Request{RepositoryPath: filepath.Join(root, "repository"), OldBPMNPath: filepath.Join(root, "bpmn", "old.bpmn"), NewBPMNPath: filepath.Join(root, "bpmn", "new.bpmn"), RepositoryName: "synthetic"}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	response := httptest.NewRecorder()
	(api.Handler{Analyzer: wiring.NewAnalyzer()}).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value domain.Report
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.AnalysisID == "" || len(value.Changes) == 0 {
		t.Fatalf("report=%#v", value)
	}
}

func TestHandlerRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewBufferString(`{"unknown":true}`))
	response := httptest.NewRecorder()
	(api.Handler{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestHandlerRejectsGroundTruthAtInferenceBoundary(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewBufferString(`{"repository_path":"repo","old_bpmn_path":"old","new_bpmn_path":"new","ground_truth":"forbidden.json"}`))
	response := httptest.NewRecorder()
	(api.Handler{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("ground truth entered inference boundary, status=%d", response.Code)
	}
}

func TestWebAPIHealthCreateRetrieveAndCleanup(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	temporaryRoot := t.TempDir()
	handler := api.Handler{
		Analyzer: wiring.NewAnalyzer(),
		Store:    filestore.ReportStore{Root: filepath.Join(t.TempDir(), "analyses")},
		TempRoot: temporaryRoot,
	}
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if health.Code != http.StatusOK || !bytes.Contains(health.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("health status=%d body=%s", health.Code, health.Body.String())
	}

	request := multipartRequest(t, map[string]string{
		"repo_path":       filepath.Join(root, "repository"),
		"repository_name": "synthetic-web",
		"revision":        "fixture-v1",
		"method":          "hybrid",
		"top_k":           "5",
	}, map[string]string{
		"old_bpmn": filepath.Join(root, "bpmn", "old.bpmn"),
		"new_bpmn": filepath.Join(root, "bpmn", "new.bpmn"),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var value domain.Report
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.AnalysisID == "" || response.Header().Get("Location") == "" {
		t.Fatalf("response=%#v headers=%v", value, response.Header())
	}
	entries, err := os.ReadDir(temporaryRoot)
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary uploads were not cleaned: entries=%v err=%v", entries, err)
	}

	for _, suffix := range []string{"", "/report.json", "/report.md"} {
		get := httptest.NewRecorder()
		handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/analyses/"+value.AnalysisID+suffix, nil))
		if get.Code != http.StatusOK || get.Body.Len() == 0 {
			t.Fatalf("get %q status=%d body=%s", suffix, get.Code, get.Body.String())
		}
	}
}

func TestWebAPIRejectsOversizedBPMNAndUnknownField(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	handler := api.Handler{
		Analyzer:    wiring.NewAnalyzer(),
		Store:       filestore.ReportStore{Root: filepath.Join(t.TempDir(), "analyses")},
		TempRoot:    t.TempDir(),
		MaxBPMNSize: 16,
	}
	request := multipartRequest(t, map[string]string{"repo_path": filepath.Join(root, "repository")}, map[string]string{
		"old_bpmn": filepath.Join(root, "bpmn", "old.bpmn"),
		"new_bpmn": filepath.Join(root, "bpmn", "new.bpmn"),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("exceeds")) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	unknown := multipartRequest(t, map[string]string{"repo_path": "repo", "ground_truth": "forbidden"}, nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, unknown)
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("unsupported field")) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func multipartRequest(t *testing.T, fields map[string]string, files map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	for name, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile(name, filepath.Base(path))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/analyses", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
