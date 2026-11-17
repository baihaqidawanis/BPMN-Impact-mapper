package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
)

func TestAgentSettingsRejectUnknownEndpointsAndPreserveConfigOnBadInput(t *testing.T) {
	r := agentjobs.RoleSettings{Model: "fixture", Profile: "coding"}
	v := agentjobs.Settings{Analyst: r, Planner: r, Reviewer: r, Context: 32768, Output: 6144}
	builds := 0
	s := &agentjobs.Service{Model: apiAgent{}, Configuration: &agentjobs.Configuration{Current: v, Build: func(agentjobs.Settings) (agentjobs.Model, error) { builds++; return apiAgent{}, nil }}}
	h := api.Handler{Agents: s}
	good, _ := json.Marshal(v)
	for _, tc := range []struct {
		method, body string
		status       int
	}{
		{"GET", "", 200}, {"POST", "", 405}, {"PUT", `{"base_url":"https://remote.example"}`, 400},
		{"PUT", `{}`, 422}, {"PUT", string(good) + ` {}`, 400}, {"PUT", string(good), 200},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, "/api/v1/agent-settings", bytes.NewBufferString(tc.body))
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.body, w.Code, w.Body.String())
		}
	}
	if builds != 1 {
		t.Fatal("bad input reached factory", builds)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Configure(ctx, v); err == nil {
		t.Fatal("cancelled request changed settings")
	}
}
