package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/baihaqidawan/bpmn-impact-mapper/internal/agentjobs"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/planning"
	"github.com/baihaqidawan/bpmn-impact-mapper/internal/reportstore"
)

func isAgentStart(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/analyses/"), "/")
	return strings.HasPrefix(path, "/api/v1/analyses/") && len(parts) == 2 && parts[0] != "" && parts[1] == "agent-sessions"
}

func (h Handler) agentCapability(w http.ResponseWriter, r *http.Request) {
	if h.Agents == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "available": false, "message": "Start the backend with --agent-model to enable Qwen review."})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err := h.Agents.Check(ctx)
	message := "Local model metadata is available. No source or inference request was sent."
	if err != nil {
		message = "Model unavailable. Check VPN, SSH tunnel and installed model."
	}
	if errors.Is(err, agentjobs.ErrBusy) {
		message = "A review is running. Model metadata remains pinned until it finishes."
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "available": err == nil, "provider": h.Agents.Model.Identity(), "message": message})
}

func (h Handler) startAgents(w http.ResponseWriter, r *http.Request) {
	if h.Agents == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "AGENTS_DISABLED", "Qwen review is not enabled on this backend.")
		return
	}
	if h.Store == nil {
		h.internalError(w, errors.New("analysis store missing"))
		return
	}
	var input struct {
		RepositoryPath string `json:"repository_path"`
		ChangeIndex    int    `json:"change_index"`
		SessionID      string `json:"session_id,omitempty"`
		Mode           string `json:"mode,omitempty"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(&input); err != nil {
		writeAPIError(w, 400, "INVALID_REQUEST", "Repository path and selected change are required.")
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		writeAPIError(w, 400, "INVALID_REQUEST", "Request contains trailing data.")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/analyses/"), "/")
	report, err := h.Store.Get(r.Context(), parts[0])
	if errors.Is(err, reportstore.ErrNotFound) {
		writeAPIError(w, 404, "NOT_FOUND", "Analysis report not found.")
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}
	v, err := h.Agents.Start(r.Context(), report, strings.TrimSpace(input.RepositoryPath), input.ChangeIndex, input.SessionID, input.Mode)
	if errors.Is(err, agentjobs.ErrBusy) {
		writeAPIError(w, 409, "AGENT_BUSY", "Another Qwen session is running. Wait for it to finish.")
		return
	}
	if err != nil {
		writeAPIError(w, 422, "AGENT_START_FAILED", "Session could not start. Check the selected change and existing session identity.")
		return
	}
	writeJSON(w, http.StatusAccepted, v)
}

func (h Handler) getAgentSession(w http.ResponseWriter, r *http.Request) {
	if h.Agents == nil {
		writeAPIError(w, 503, "AGENTS_DISABLED", "Qwen review is not enabled.")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/agent-sessions/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	v, err := h.Agents.Get(r.Context(), id)
	if errors.Is(err, agentjobs.ErrMissing) || errors.Is(err, planning.ErrAgentSessionMissing) {
		writeAPIError(w, 404, "NOT_FOUND", "Agent session not found. Start a new session.")
		return
	}
	if err != nil {
		writeAPIError(w, 422, "SESSION_UNAVAILABLE", "Agent checkpoint could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, v)
}
