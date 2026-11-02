package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestOllamaStructuredTurnAndIdentity(t *testing.T) {
	cloud, drift := false, false
	chats := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			digest := strings.Repeat("a", 64)
			if drift {
				digest = strings.Repeat("b", 64)
			}
			remote := ""
			if cloud {
				remote = "https://outside.invalid"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": digest, "remote_host": remote, "capabilities": []string{"tools"}}}})
		case "/api/version":
			_, _ = w.Write([]byte(`{"version":"fixture-version"}`))
		case "/api/chat":
			chats++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["stream"] != false || body["think"] != false || body["format"] == nil {
				t.Error("missing bounded structured protocol")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": `{"review":{"approved":true,"issues":[]}}`}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: server.URL, Model: "fixture", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := m.Identity()
	r, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer})
	if err != nil || r.Review == nil || !r.Review.Approved {
		t.Fatalf("turn: %+v %v", r, err)
	}
	if identity != m.Identity() {
		t.Fatal("identity changed during inference")
	}
	drift = true
	if _, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer}); err == nil {
		t.Fatal("model drift accepted")
	}
	drift = false
	cloud = true
	if _, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer}); err == nil {
		t.Fatal("cloud alias accepted")
	}
	if chats != 1 {
		t.Fatal("unsafe request sent to model")
	}
}

func TestOllamaRejectsUnsafeEndpoints(t *testing.T) {
	for _, base := range []string{"https://example.com", "http://10.0.0.1:11434", "http://user:pass@127.0.0.1:11434", "http://127.0.0.1:11434/?key=value", "http://127.0.0.1:11434/path"} {
		if _, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: base, Model: "fixture"}); err == nil {
			t.Fatal("accepted", base)
		}
	}
	if _, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "model:cloud"}); err == nil {
		t.Fatal("cloud accepted")
	}
}

func TestOllamaActionSchemaBindsEvidenceAndCardinality(t *testing.T) {
	schema, err := agentActionSchema(planning.Analyst)
	if err != nil {
		t.Fatal(err)
	}
	constrainActionArray(schema, "verification", "decisions", 2)
	constrainActionEvidence(schema, "verification", "decisions", []string{"exact-evidence-id"})
	found := false
	for _, choice := range schema["oneOf"].([]any) {
		props := choice.(map[string]any)["properties"].(map[string]any)
		if verification, ok := props["verification"].(map[string]any); ok {
			array := verification["properties"].(map[string]any)["decisions"].(map[string]any)
			if array["minItems"] != 2 || array["maxItems"] != 2 {
				t.Fatal("discovered candidate count not bound")
			}
			itemProps := array["items"].(map[string]any)["properties"].(map[string]any)
			ids := itemProps["evidence_ids"].(map[string]any)["items"].(map[string]any)["enum"].([]string)
			if len(ids) != 1 || ids[0] != "exact-evidence-id" {
				t.Fatal("evidence IDs not bound")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("verification schema missing")
	}
}

func TestOllamaMalformedTruncatedAndRedirect(t *testing.T) {
	for _, mode := range []string{"unknown-field", "trailing", "truncated", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/tags" {
					_, _ = w.Write([]byte(`{"models":[{"name":"fixture","digest":"` + strings.Repeat("a", 64) + `","capabilities":["tools"]}]}`))
					return
				}
				if r.URL.Path == "/api/version" {
					_, _ = w.Write([]byte(`{"version":"v1"}`))
					return
				}
				if mode == "redirect" {
					http.Redirect(w, r, "http://127.0.0.1:1/elsewhere", http.StatusTemporaryRedirect)
					return
				}
				content := `{"review":{"approved":true,"issues":[]}}`
				reason := "stop"
				if mode == "unknown-field" {
					content = `{"shell":"run this"}`
				}
				if mode == "trailing" {
					content += " {}"
				}
				if mode == "truncated" {
					reason = "length"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": reason, "message": map[string]any{"role": "assistant", "content": content}})
			}))
			defer s.Close()
			m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: s.URL, Model: "fixture", Client: s.Client()})
			if err != nil {
				t.Fatal(err)
			}
			if err := m.Prepare(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer}); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
}
