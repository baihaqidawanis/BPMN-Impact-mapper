package agentjobs

import (
	"context"
	"errors"
)

// ModelCheckMessage maps adapter categories to fixed, safe UI guidance.
// Unknown errors are deliberately not echoed to the browser.
func ModelCheckMessage(err error) string {
	if err == nil {
		return "Local model metadata is available. No source or inference request was sent."
	}
	if errors.Is(err, ErrBusy) {
		return "A review is running. Model metadata remains pinned until it finishes."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Model metadata check timed out. Check the tunnel and lab runtime; no inference was requested."
	}
	var classified interface{ ReadinessCode() string }
	if errors.As(err, &classified) {
		switch classified.ReadinessCode() {
		case "MODEL_NOT_INSTALLED":
			return "Ollama is reachable, but a selected model alias is not installed. Register the local GGUF in Ollama, refresh the model list and apply settings. A folder alone is not an installed alias."
		case "THINKING_UNSUPPORTED":
			return "A selected model does not advertise thinking support. Turn Thinking off for that role and apply settings."
		case "GENERATION_UNSUPPORTED":
			return "A selected model does not advertise text generation. Select a local completion model instead of an embedding-only model."
		case "REMOTE_MODEL":
			return "A selected alias points to a remote service. Select a model installed on the lab machine; cloud inference is disabled here."
		case "MODEL_IDENTITY_CHANGED":
			return "Model or Ollama metadata changed. Reapply settings and start a new AI session; existing checkpoints keep their original identity."
		case "MODEL_METADATA_INVALID":
			return "Ollama responded, but required model metadata is missing or invalid. Check the installed model and runtime before starting a review."
		}
	}
	return "Model metadata unavailable. Check the SSH tunnel and lab runtime. This check does not establish a VPN failure."
}
