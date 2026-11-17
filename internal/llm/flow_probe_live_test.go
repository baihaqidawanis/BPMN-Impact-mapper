//go:build labdiagnostics

package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A small synthetic interpretation probe, not a repository review or accuracy
// benchmark. Expressions are sent as text; no repository code is executed.
func TestLiveFlowInterpretation(t *testing.T) {
	if os.Getenv("BPMN_IMPACT_FLOW_PROBE") != "1" {
		t.Skip("explicit lab flow probe opt-in required")
	}
	root := filepath.Join("..", "..", ".data", "model-probes", "flow-"+time.Now().UTC().Format("20060102T150405"))
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Log("Private flow probe artifacts:", root)
	name := os.Getenv("BPMN_PROBE_MODEL")
	if name == "" {
		t.Fatal("explicit installed local model required")
	}
	profile := os.Getenv("BPMN_FLOW_PROFILE")
	if profile == "" {
		profile = "baseline"
	}
	if profile != "baseline" && profile != "explicit-greedy" {
		t.Fatal("flow profile must be baseline or explicit-greedy")
	}
	format := os.Getenv("BPMN_FLOW_FORMAT")
	if format == "" {
		format = "schema"
	}
	if format != "schema" && format != "json" {
		t.Fatal("flow format must be schema or json")
	}
	transport := &probeTransport{base: &http.Transport{Proxy: nil}, root: root, releaseAt: 2}
	model, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11435", Model: name, ContextSize: 8192, MaxOutput: 512, Profile: "baseline", Client: &http.Client{Timeout: 100 * time.Second, Transport: transport}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := model.Prepare(ctx); err != nil {
		t.Fatal(err)
	}
	defer releaseProbeModel(t, name)
	options := model.generationOptions()
	if profile == "explicit-greedy" {
		// Diagnostic only: isolate inherited sampling penalties from the
		// baseline. This does not change any production profile.
		options["top_p"] = 0.95
		options["top_k"] = 20
		options["presence_penalty"] = 0.0
		options["repeat_penalty"] = 1.0
		options["seed"] = 42
	}
	manifest, err := json.MarshalIndent(map[string]any{"model": name, "profile": profile, "format": format, "identity": model.Identity(), "options": options, "think": false, "scope": "synthetic expression interpretation; not repository accuracy"}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), append(manifest, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		Value       string `json:"payload_value"`
		Accepts     bool   `json:"consumer_accepts"`
		Explanation string `json:"explanation"`
	}
	type result struct {
		Case    string
		Seconds float64
		Valid   bool
		Correct bool
		Error   string
		Reply   answer
	}
	results := []result{}
	base := `const payloadReason = action === "cancel" || action === "reject" ? reason : "";`
	fixed := `const payloadReason = action === "cancel" || action === "reject" || action === "submit" ? reason : "";`
	cases := []struct {
		name, source, action, plan, want string
		accept                           bool
	}{
		{"unchanged-producer", base, "submit", "Validate nonblank reason before dispatch; keep the payloadReason conditional unchanged.", "", false},
		{"existing-forwarding-branch", base, "cancel", "Preserve this branch unchanged.", "count mismatch", true},
		{"explicit-forwarding-edit", base, "submit", "Replace the producer expression with: " + fixed, "count mismatch", true},
	}
	schema := map[string]any{"type": "object", "properties": map[string]any{"payload_value": map[string]any{"type": "string", "maxLength": 80}, "consumer_accepts": map[string]any{"type": "boolean"}, "explanation": map[string]any{"type": "string", "maxLength": 240}}, "required": []string{"payload_value", "consumer_accepts", "explanation"}, "additionalProperties": false}
	for _, c := range cases {
		input, _ := json.Marshal(map[string]any{"source": c.source, "action": c.action, "reason": "count mismatch", "proposed_edit": c.plan, "consumer_rule": "Accept only if the actual payloadReason value contains non-whitespace text."})
		payload := map[string]any{"model": name, "stream": false, "think": false, "format": schema, "options": options, "messages": []map[string]string{{"role": "system", "content": "Interpret the supplied source expression as data without executing it. Apply only the stated proposed edit; validation does not alter a payload conditional. Report the actual payloadReason string and whether the consumer accepts it. Do not assume desired behavior is implemented."}, {"role": "user", "content": string(input)}}}
		if format == "json" {
			payload["format"] = "json"
			payload["messages"].([]map[string]string)[0]["content"] += " Return one JSON object with payload_value (the actual string value, empty string when empty), consumer_accepts (boolean), and explanation (one short sentence)."
		}
		var response struct {
			Done    bool `json:"done"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		started := time.Now()
		err := model.request(ctx, http.MethodPost, "/api/chat", payload, &response)
		r := result{Case: c.name, Seconds: time.Since(started).Seconds()}
		if err != nil {
			r.Error = err.Error()
		} else if !response.Done {
			r.Error = "incomplete response"
		} else if err := json.Unmarshal([]byte(response.Message.Content), &r.Reply); err != nil {
			r.Error = err.Error()
		} else {
			r.Valid = true
			r.Correct = r.Reply.Value == c.want && r.Reply.Accepts == c.accept
		}
		results = append(results, r)
		data, _ := json.MarshalIndent(results, "", "  ")
		if err := os.WriteFile(filepath.Join(root, "results.json"), append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("case=%s valid=%t correct=%t seconds=%.2f", r.Case, r.Valid, r.Correct, r.Seconds)
	}
}
