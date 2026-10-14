package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestOpenAICompatibleProviderUsesBoundedTypedContract(t *testing.T) {
	responses := []string{
		`{"decisions":[{"change_type":"ADD_TASK","element_id":"TASK-1","file":"internal/request.go","verdict":"RELEVANT","rationale":"Evidence supports the target.","evidence_ids":["E1"]}]}`,
		`{"plans":[{"change_type":"ADD_TASK","element_id":"TASK-1","file":"internal/request.go","module":"internal","layer":"application","action":"VERIFY","summary":"Inspect the bounded target.","steps":["Review the cited behavior."],"evidence_ids":["E1"]}]}`,
		`{"approved":true,"issues":[],"revised":[]}`,
	}
	var lock sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("unexpected request path=%s auth=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		var body struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.Model != "fixture-model" || body.ResponseFormat.Type != "json_object" || len(body.Messages) != 2 {
			t.Errorf("invalid request body=%#v err=%v", body, err)
		}
		lock.Lock()
		content := responses[calls]
		calls++
		lock.Unlock()
		_ = json.NewEncoder(writer).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
	defer server.Close()

	provider, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{BaseURL: server.URL + "/v1", APIKey: "test-secret", Model: "fixture-model", PromptVersion: "planning-v1", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	change := planning.ChangeInput{ChangeType: "ADD_TASK", ElementID: "TASK-1", Candidates: []planning.Candidate{{File: "internal/request.go", Module: "internal", Layer: "application", Evidence: []planning.EvidenceItem{{ID: "E1"}}}}}
	verification, err := provider.VerifyCandidates(context.Background(), planning.VerificationRequest{Change: change})
	if err != nil || len(verification.Decisions) != 1 {
		t.Fatalf("verification=%#v err=%v", verification, err)
	}
	bundle := planning.EvidenceBundle{Changes: []planning.ChangeInput{change}}
	planned, err := provider.PlanLayer(context.Background(), planning.LayerPlanningRequest{Layer: "application", Bundle: bundle, Decisions: verification.Decisions})
	if err != nil || len(planned.Plans) != 1 {
		t.Fatalf("planned=%#v err=%v", planned, err)
	}
	review, err := provider.ReviewPlan(context.Background(), planning.ReviewRequest{Bundle: bundle, Draft: planned.Plans})
	if err != nil || !review.Approved || calls != 3 {
		t.Fatalf("review=%#v calls=%d err=%v", review, calls, err)
	}
	if provider.Identity().Provider != "openai-compatible" || provider.Identity().PromptVersion != "planning-v1" {
		t.Fatalf("identity=%#v", provider.Identity())
	}
}

func TestOpenAICompatibleProviderRejectsUnsafeOrInvalidResponses(t *testing.T) {
	if _, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{BaseURL: "http://example.com/v1", Model: "m", PromptVersion: "p"}); err == nil {
		t.Fatal("remote plaintext LLM endpoint accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(writer).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"decisions":[],"unexpected":true}`}}}})
	}))
	defer server.Close()
	provider, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{BaseURL: server.URL, Model: "m", PromptVersion: "p", Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.VerifyCandidates(context.Background(), planning.VerificationRequest{})
	if err == nil || !strings.Contains(err.Error(), "decode candidate verification JSON") {
		t.Fatalf("invalid provider response error=%v", err)
	}
}
