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
)

func (h Handler) agentSettings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/agent-models" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeAPIError(w, 405, "METHOD_NOT_ALLOWED", "Use GET.")
			return
		}
		if h.Agents == nil {
			writeAPIError(w, 503, "AGENTS_DISABLED", "Model service is disabled.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		models, err := h.Agents.AvailableModels(ctx)
		if err != nil {
			writeAPIError(w, 503, "MODEL_LIST_UNAVAILABLE", "Installed model list unavailable. Check the tunnel or wait for the active review.")
			return
		}
		writeJSON(w, 200, map[string]any{"models": models})
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Set("Allow", "GET, PUT")
		writeAPIError(w, 405, "METHOD_NOT_ALLOWED", "Use GET or PUT.")
		return
	}
	if h.Agents == nil {
		writeAPIError(w, 503, "AGENTS_DISABLED", "Start the backend with --agent-model first.")
		return
	}
	if _, ok := h.Agents.Settings(); !ok {
		writeAPIError(w, 503, "SETTINGS_DISABLED", "Web configuration is unavailable on this backend.")
		return
	}
	if r.Method == http.MethodPut {
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeAPIError(w, 400, "INVALID_REQUEST", "Send same-origin JSON settings.")
			return
		}
		var v agentjobs.Settings
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		d.DisallowUnknownFields()
		if err := d.Decode(&v); err != nil {
			writeAPIError(w, 400, "INVALID_REQUEST", "Invalid model settings.")
			return
		}
		if err := d.Decode(&struct{}{}); err != io.EOF {
			writeAPIError(w, 400, "INVALID_REQUEST", "Unexpected trailing settings.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		err := h.Agents.Configure(ctx, v)
		if errors.Is(err, agentjobs.ErrBusy) {
			writeAPIError(w, 409, "AGENT_BUSY", "Wait for the running review to finish or stop it before applying settings.")
			return
		}
		if err != nil {
			message := agentjobs.ModelCheckMessage(err)
			if errors.Is(err, agentjobs.ErrConfiguration) {
				message = "Check token limits, model names and generation profiles."
			}
			writeAPIError(w, 422, "SETTINGS_REJECTED", "Settings were not applied. "+message+" Previous settings remain active.")
			return
		}
	}
	v, _ := h.Agents.Settings()
	writeJSON(w, 200, v)
}
