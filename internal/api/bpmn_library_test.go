package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/bpmnlib"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/filestore"
)

const libraryXML = `<?xml version="1.0" encoding="UTF-8"?><definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"><process id="p"><startEvent id="s"/><exclusiveGateway id="g"/><endEvent id="e"/><sequenceFlow id="f1" sourceRef="s" targetRef="g"/><sequenceFlow id="f2" sourceRef="g" targetRef="e"/></process></definitions>`

func TestLibraryPersistReloadAndPreserveXML(t *testing.T) {
	root := filepath.Join(t.TempDir(), "library")
	handler := api.Handler{Library: filestore.BPMNLibrary{Root: root}}
	input := map[string]any{"repository": "Example", "process": "Review", "role": "Staff", "level": 3, "version": "v1", "xml": libraryXML}
	data, _ := json.Marshal(input)
	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/bpmn-library", bytes.NewReader(data)))
		return w
	}
	w := post()
	if w.Code != 201 {
		t.Fatalf("save %d: %s", w.Code, w.Body.String())
	}
	var entry bpmnlib.Entry
	if err := json.Unmarshal(w.Body.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range entry.Findings {
		if f.Code == "ACTIVITY_AFTER_GATEWAY" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing strict gateway convention warning")
	}
	if entry.XML != libraryXML {
		t.Fatal("XML was rewritten")
	}
	if post().Code != 201 {
		t.Fatal("duplicate save must be idempotent")
	}
	// A fresh handler/store simulates a backend restart.
	handler = api.Handler{Library: filestore.BPMNLibrary{Root: root}}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/bpmn-library", nil))
	var list struct {
		Entries []bpmnlib.Entry `json:"entries"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if w.Code != 200 || len(list.Entries) != 1 || list.Entries[0].XML != "" {
		t.Fatalf("invalid list: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/bpmn-library/"+entry.ID, nil))
	var restored bpmnlib.Entry
	json.Unmarshal(w.Body.Bytes(), &restored)
	if w.Code != 200 || restored.XML != libraryXML {
		t.Fatal("persisted XML not restored")
	}
	input["version"] = "v2"
	data, _ = json.Marshal(input)
	if post().Code != 201 {
		t.Fatal("new version failed")
	}
	values, err := handler.Library.List()
	if err != nil || len(values) != 2 {
		t.Fatal("version should not replace baseline")
	}
}

func TestLibraryRejectsInvalidInputAndIDs(t *testing.T) {
	handler := api.Handler{Library: filestore.BPMNLibrary{Root: t.TempDir()}}
	for _, body := range []string{`{}`, `{"repository":"X","process":"Y","version":"1","level":9,"xml":"oops"}`, `{"repository":"X","process":"Y","version":"1","level":3,"xml":"<!DOCTYPE x><x/>"}`} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/bpmn-library", bytes.NewBufferString(body)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("invalid accepted: %d", w.Code)
		}
	}
	for _, id := range []string{"missing", "../../secret", "%2e%2e%2fsecret"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/bpmn-library/"+id, nil))
		if w.Code != 404 {
			t.Fatalf("invalid ID %q: %d", id, w.Code)
		}
	}
}
