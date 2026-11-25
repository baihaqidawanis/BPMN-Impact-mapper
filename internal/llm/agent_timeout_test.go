package llm

import (
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"net/http"
	"testing"
	"time"
)

func TestAgentHTTPBudgetAllowsWholeThinkingTurn(t *testing.T) {
	for _, client := range []*http.Client{nil, {}} {
		m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "fixture", Client: client})
		if err != nil {
			t.Fatal(err)
		}
		if m.client.Timeout <= planning.AgentTurnTimeout {
			t.Fatal("HTTP timeout cuts off planning turn", m.client.Timeout)
		}
	}
	custom := &http.Client{Timeout: time.Second}
	m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: "fixture", Client: custom})
	if err != nil || m.client.Timeout != time.Second {
		t.Fatal("explicit client timeout overwritten", err)
	}
}
