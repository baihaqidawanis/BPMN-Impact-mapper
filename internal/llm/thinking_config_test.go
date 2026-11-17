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

func TestThinkingIsExplicitCapabilityCheckedAndIdentityBound(t *testing.T) {
	capabilities := []string{"completion", "thinking"}
	chats := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			json.NewEncoder(w).Encode(map[string]any{"models": []any{map[string]any{"name": "fixture", "digest": strings.Repeat("a", 64), "capabilities": capabilities}}})
		case "/api/version":
			w.Write([]byte(`{"version":"fixture"}`))
		case "/api/chat":
			chats++
			var body struct {
				Think   bool `json:"think"`
				Options struct {
					NumPredict int `json:"num_predict"`
				} `json:"options"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Think || body.Options.NumPredict != 8192 {
				t.Error("thinking/output config missing", body, err)
			}
			json.NewEncoder(w).Encode(map[string]any{"model": "fixture", "done": true, "done_reason": "stop", "message": map[string]any{"role": "assistant", "thinking": "private reasoning must not enter the plan", "content": `{"review":{"approved":true,"issues":[]}}`}})
		}
	}))
	defer srv.Close()
	newModel := func(think bool) *OllamaAgent {
		m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: srv.URL, Model: "fixture", ContextSize: 32768, MaxOutput: 8192, Think: think, Client: srv.Client()})
		if err != nil {
			t.Fatal(err)
		}
		if err = m.Prepare(context.Background()); err != nil {
			t.Fatal(err)
		}
		return m
	}
	on, off := newModel(true), newModel(false)
	if on.Identity() == off.Identity() {
		t.Fatal("generation profile drift can resume an old session")
	}
	result, err := on.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer})
	if err != nil || result.Review == nil || !result.Review.Approved {
		t.Fatal(result, err)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "private reasoning") {
		t.Fatal("thinking leaked into application output")
	}
	capabilities = []string{"completion"}
	if err = on.Prepare(context.Background()); err == nil {
		t.Fatal("unsupported thinking accepted")
	}
	if _, err = on.Turn(context.Background(), planning.AgentTurn{Role: planning.Reviewer}); err == nil || chats != 1 {
		t.Fatal("unsupported model inference attempted", err, chats)
	}
}

func TestCodingProfileOverridesInheritedDefaultsAndPinsIdentity(t *testing.T) {
	newModel := func(profile string) *OllamaAgent {
		m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "fixture", ContextSize: 32768, Profile: profile})
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	baseline, coding := newModel("baseline"), newModel("coding")
	opts := coding.generationOptions()
	if opts["temperature"] != 0.6 || opts["presence_penalty"] != 0.0 || opts["repeat_penalty"] != 1.0 || opts["seed"] != 42 || opts["top_p"] != 0.95 || opts["top_k"] != 20 {
		t.Fatal("incomplete explicit coding profile", opts)
	}
	if baseline.Identity() == coding.Identity() || len(baseline.generationOptions()) != 3 {
		t.Fatal("profile drift or silent baseline change")
	}
	if _, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "fixture", Profile: "unverified"}); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
