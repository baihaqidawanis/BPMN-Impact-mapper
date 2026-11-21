package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestImportedModelPreparationAndReadinessCategories(t *testing.T) {
	for _, tc := range []struct {
		name, code                string
		capabilities              []string
		missing, remote, thinking bool
	}{
		{name: "imported-completion", capabilities: []string{"completion"}},
		{name: "not-registered", missing: true, code: "MODEL_NOT_INSTALLED"},
		{name: "embedding-only", capabilities: []string{"embedding"}, code: "GENERATION_UNSUPPORTED"},
		{name: "unsupported-thinking", capabilities: []string{"completion"}, thinking: true, code: "THINKING_UNSUPPORTED"},
		{name: "remote-alias", capabilities: []string{"completion"}, remote: true, code: "REMOTE_MODEL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const alias = "imported-fixture:27b-q4_k_m"
			calls := 0
			client := &http.Client{Transport: errorTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet {
					t.Fatal("preparation attempted inference or installation")
				}
				var payload any
				switch r.URL.Path {
				case "/api/tags":
					models := []any{}
					if !tc.missing {
						tag := map[string]any{"name": alias, "digest": strings.Repeat("a", 64), "capabilities": tc.capabilities}
						if tc.remote {
							tag["remote_host"] = "https://private.invalid/token"
						}
						models = append(models, tag)
					}
					payload = map[string]any{"models": models}
				case "/api/version":
					payload = map[string]string{"version": "fixture"}
				default:
					t.Fatal("unexpected request", r.URL.Path)
				}
				body, _ := json.Marshal(payload)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
			})}
			model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: alias, Think: tc.thinking, Client: client})
			if err != nil {
				t.Fatal(err)
			}
			err = model.Prepare(context.Background())
			if tc.code == "" {
				if err != nil || calls != 2 {
					t.Fatal("imported local model not prepared", err, calls)
				}
			} else {
				var classified interface{ ReadinessCode() string }
				if !errors.As(err, &classified) || classified.ReadinessCode() != tc.code || calls != 1 {
					t.Fatal(err, calls)
				}
				if strings.Contains(err.Error(), "private.invalid") {
					t.Fatal("remote metadata leaked")
				}
			}
		})
	}
}
