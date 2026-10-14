package api

import (
	"context"
	"errors"
	"strings"
)

// Classify stages without returning source excerpts, local paths, or subprocess output.
func analysisFailureMessage(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "Analysis timed out or was cancelled. Retry without method comparison; check the backend log if it repeats."
	}
	message := err.Error()
	for _, stage := range []struct{ prefix, message string }{
		{"inventory repository:", "Repository could not be read. Check the server-local path, permissions, and inventory limits."},
		{"parse old BPMN:", "Baseline BPMN is invalid or exceeds parser limits. Check duplicate IDs and flow references; see the backend log."},
		{"parse new BPMN:", "New BPMN is invalid or exceeds parser limits. Check duplicate IDs and flow references; see the backend log."},
		{"apply ", "Mapping failed after BPMN parsing. Check the backend log for the selected method and local embedding runtime."},
	} {
		if strings.HasPrefix(message, stage.prefix) {
			return stage.message
		}
	}
	return "Analysis could not be completed. Check the backend log for the detailed cause."
}
