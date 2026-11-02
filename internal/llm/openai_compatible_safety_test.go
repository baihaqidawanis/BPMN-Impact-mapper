package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/llm"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
)

func TestPlanningCompletionRejectsMalformedOrIncompleteAnswers(t *testing.T) {
	for _, test := range []struct {
		name, content, finishReason, suffix string
		valid                               bool
	}{
		{name: "approved", content: `{"approved":true,"issues":[]}`, finishReason: "stop", valid: true},
		{name: "not-approved", content: `{"approved":false,"issues":["Check the evidence."]}`, valid: true},
		{name: "null", content: `null`},
		{name: "missing-fields", content: `{}`},
		{name: "missing-issues", content: `{"approved":true}`},
		{name: "null-approved", content: `{"approved":null,"issues":[]}`},
		{name: "null-issues", content: `{"approved":true,"issues":null}`},
		{name: "wrong-type", content: `{"approved":"yes","issues":[]}`},
		{name: "unknown-field", content: `{"approved":true,"issues":[],"execute":true}`},
		{name: "trailing-content", content: `{"approved":true,"issues":[]} {}`},
		{name: "trailing-envelope", content: `{"approved":true,"issues":[]}`, suffix: `{}`},
		{name: "token-limit", content: `{"approved":true,"issues":[]}`, finishReason: "length"},
		{name: "filtered", content: `{"approved":true,"issues":[]}`, finishReason: "content_filter"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{
					"finish_reason": test.finishReason,
					"message":       map[string]string{"content": test.content},
				}}})
				_, _ = w.Write([]byte(test.suffix))
			}))
			defer server.Close()
			provider := mockProvider(t, server, 0)
			_, err := provider.ReviewPlan(context.Background(), planning.ReviewRequest{})
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
		})
	}
}

func TestPlanningProviderNeverFollowsRedirectWithInjectedClient(t *testing.T) {
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
	}))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := server.Client()
	provider, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{
		BaseURL: server.URL, Model: "mock", PromptVersion: "v1", Client: client,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.ReviewPlan(context.Background(), planning.ReviewRequest{})
	if err == nil || !strings.Contains(err.Error(), "HTTP 307") || redirected.Load() {
		t.Fatalf("redirected=%v error=%v", redirected.Load(), err)
	}
	if client.CheckRedirect != nil || client.Timeout != 0 {
		t.Fatal("adapter mutated the caller's HTTP client")
	}
}

func TestPlanningProviderRejectsAmbiguousEndpointURLs(t *testing.T) {
	for _, endpoint := range []string{
		"https://user:password@example.invalid/v1", "https://example.invalid/v1?key=secret",
		"https://example.invalid/v1?", "https://example.invalid/v1#fragment",
	} {
		_, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{BaseURL: endpoint, Model: "mock", PromptVersion: "v1"})
		if err == nil {
			t.Fatal("endpoint with credentials, query or fragment accepted")
		}
	}
}

func TestPlanningProviderBoundsRequestsAndPreservesCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	}))
	defer server.Close()
	provider := mockProvider(t, server, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.ReviewPlan(ctx, planning.ReviewRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost: %v", err)
	}
	// JSON encoded inside the message is escaped a second time. Bound the
	// actual outbound payload, not only the inner evidence JSON.
	_, err := provider.VerifyCandidates(context.Background(), planning.VerificationRequest{
		Change: planning.ChangeInput{Reason: strings.Repeat(`"`, 1500000)},
	})
	if err == nil || !strings.Contains(err.Error(), "serialized request exceeds") || calls.Load() != 0 {
		t.Fatalf("oversized request reached server: calls=%d error=%v", calls.Load(), err)
	}
}

func TestPlanningProviderBoundsResponsesAndDoesNotEchoProviderBody(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusTooManyRequests} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("private-provider-detail"))
		}))
		limit := int64(8)
		if status == http.StatusTooManyRequests {
			limit = 128
		}
		provider := mockProvider(t, server, limit)
		_, err := provider.ReviewPlan(context.Background(), planning.ReviewRequest{})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "private-provider-detail") {
			t.Fatalf("provider body exposed or invalid response accepted: %v", err)
		}
		if status == http.StatusTooManyRequests && !strings.Contains(err.Error(), "HTTP 429") {
			t.Fatalf("missing status diagnostic: %v", err)
		}
	}
}

func mockProvider(t *testing.T, server *httptest.Server, limit int64) *llm.OpenAICompatibleProvider {
	t.Helper()
	provider, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{
		BaseURL: server.URL, Model: "mock", PromptVersion: "v1", Client: server.Client(), MaxResponse: limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}
