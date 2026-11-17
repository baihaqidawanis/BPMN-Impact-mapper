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
	capabilities := []string{"completion"}
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
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": digest, "remote_host": remote, "capabilities": capabilities}}})
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

func TestPlannerSchemaRespectsRemainingBudgets(t *testing.T) {
	for _, budget := range []struct{ tools, revisions int }{{0, 2}, {8, 0}, {8, 2}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/tags":
				_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
			case "/api/version":
				_, _ = w.Write([]byte(`{"version":"v1"}`))
			case "/api/chat":
				var body struct {
					Format   json.RawMessage `json:"format"`
					Messages []struct {
						Content string `json:"content"`
					} `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				allowed := budget.tools > 0 && budget.revisions > 0
				if strings.Contains(string(body.Format), `"need_evidence"`) != allowed {
					t.Errorf("handoff schema disagrees with budget %+v", budget)
				}
				if !allowed && !strings.Contains(body.Messages[0].Content, "No further evidence handoff") {
					t.Error("missing exhausted-budget guidance")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "message": map[string]any{"role": "assistant", "content": `{"plan":{"plans":[]}}`}})
			}
		}))
		m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: s.URL, Model: "fixture", Client: s.Client()})
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Prepare(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Planner, RemainingTools: budget.tools, RemainingRevisions: budget.revisions}); err != nil {
			t.Fatal(err)
		}
		s.Close()
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

func TestOllamaReadmeOrientationUsesSummaryOnlySchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"tools"}}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture-version"}`))
		case "/api/chat":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			format, _ := json.Marshal(body["format"])
			if !strings.Contains(string(format), "repository_summary") || !strings.Contains(string(format), `"maxItems":0`) {
				t.Error("orientation schema allows impact decisions", string(format))
			}
			messages := body["messages"].([]any)
			instruction := messages[0].(map[string]any)["content"].(string)
			if !strings.Contains(instruction, "untrusted background") {
				t.Error("README trusted as instruction")
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "content": `{"verification":{"decisions":[],"repository_summary":"README describes an inventory application."}}`}})
		}
	}))
	defer server.Close()
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: server.URL, Model: "fixture", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	reply, err := m.Turn(context.Background(), planning.AgentTurn{Role: planning.Analyst, Readme: &planning.ReadmeInput{File: "README.md", Content: "Inventory application"}})
	if err != nil || reply.Verification == nil || reply.Verification.RepositorySummary == "" {
		t.Fatal("orientation failed", err)
	}
}

func TestOllamaEmbeddingOnlyModelRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Error("unexpected model request", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": []string{"embedding"}}}})
	}))
	defer server.Close()
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: server.URL, Model: "fixture", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Prepare(context.Background()); err == nil || !strings.Contains(err.Error(), "text generation") {
		t.Fatal("embedding-only model accepted", err)
	}
}
