package api_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/api"
)

type unavailableAgent struct {
	apiAgent
	err error
}

func (m unavailableAgent) Prepare(context.Context) error { return m.err }

type missingAliasError struct{}

func (missingAliasError) Error() string         { return "private metadata token=do-not-expose" }
func (missingAliasError) ReadinessCode() string { return "MODEL_NOT_INSTALLED" }

func TestAgentReadinessAndApplyShowSafeSpecificGuidance(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"missing-model", fmt.Errorf("wrapped: %w", missingAliasError{}), "folder alone"},
		{"transport", errors.New("private metadata token=do-not-expose"), "does not establish a VPN failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			role := agentjobs.RoleSettings{Model: "local", Profile: "coding"}
			settings := agentjobs.Settings{Analyst: role, Planner: role, Reviewer: role, Context: 32768, Output: 6144}
			svc := &agentjobs.Service{Model: unavailableAgent{err: tc.err}, Configuration: &agentjobs.Configuration{Current: settings, Build: func(agentjobs.Settings) (agentjobs.Model, error) { return unavailableAgent{err: tc.err}, nil }}}
			h := api.Handler{Agents: svc}
			for _, method := range []string{"POST", "PUT"} {
				path := "/api/v1/capabilities/agents/check"
				body := ""
				status := 200
				if method == "PUT" {
					path = "/api/v1/agent-settings"
					body = `{"analyst":{"model":"new","thinking":false,"profile":"coding"},"planner":{"model":"new","thinking":false,"profile":"coding"},"reviewer":{"model":"new","thinking":false,"profile":"coding"},"context":32768,"output":6144}`
					status = 422
				}
				r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != status || !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "do-not-expose") {
					t.Fatal(w.Code, w.Body.String())
				}
				if method == "POST" && !strings.Contains(w.Body.String(), `"available":false`) {
					t.Fatal("unavailable model advertised as ready")
				}
			}
			current, _ := svc.Settings()
			if current != settings {
				t.Fatal("rejected settings replaced the original configuration")
			}
		})
	}
}
