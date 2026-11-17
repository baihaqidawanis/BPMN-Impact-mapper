package llm

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

type failedResponseReader struct{ cause error }

func (r failedResponseReader) Read([]byte) (int, error) { return 0, r.cause }

func TestResponseReadFailurePreservesCauseWithoutLeakingDetails(t *testing.T) {
	transportFailure := &net.OpError{Op: "read", Net: "tcp", Err: errors.New("private-endpoint-secret")}
	for _, tc := range []struct {
		name  string
		cause error
		body  bool
	}{
		{"connection", transportFailure, false},
		{"response-connection", transportFailure, true},
		{"response-deadline", context.DeadlineExceeded, true},
		{"response-cancelled", context.Canceled, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "fixture", Client: &http.Client{Transport: errorTransport(func(*http.Request) (*http.Response, error) {
				if !tc.body {
					return nil, tc.cause
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(failedResponseReader{tc.cause}), Header: make(http.Header)}, nil
			})}})
			if err != nil {
				t.Fatal(err)
			}
			var out any
			err = m.request(context.Background(), http.MethodPost, "/api/chat", nil, &out)
			if !errors.Is(err, tc.cause) || strings.Contains(err.Error(), "private-endpoint-secret") || strings.Contains(err.Error(), "check VPN/tunnel") {
				t.Fatal("cause lost or diagnosis unsafe", err)
			}
		})
	}
}

type errorTransport func(*http.Request) (*http.Response, error)

func (f errorTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestDistinguishesDeadlineCancellationAndMemoryFailure(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		m, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "fixture", Client: &http.Client{Transport: errorTransport(func(*http.Request) (*http.Response, error) { return nil, cause })}})
		var out any
		err := m.request(context.Background(), http.MethodPost, "/api/chat", nil, &out)
		if !errors.Is(err, cause) || strings.Contains(err.Error(), "check VPN/tunnel") {
			t.Fatal("misdiagnosed cancellation/timeout", err)
		}
	}
	m, _ := NewOllamaAgent(OllamaAgentConfig{BaseURL: "http://127.0.0.1:11434", Model: "fixture", Client: &http.Client{Transport: errorTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"error":"out of memory at /private/model/path"}`)), Header: make(http.Header)}, nil
	})}})
	var out any
	err := m.request(context.Background(), http.MethodPost, "/api/chat", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "could not fit") || strings.Contains(err.Error(), "/private/") {
		t.Fatal("unsafe/unhelpful model error", err)
	}
}
