# Agent settings in the web workspace

Planning / AI Agents now has a **Model settings** disclosure. Review scope (Focused or Full repository) is independent from generation settings.

For each of Analyst, Planner and Reviewer, select an installed local Ollama alias from the dropdown, thinking on/off and generation profile (baseline/coding). Context and output token limits are shared. Refresh model list after registering a GGUF in Ollama; embeddings and remote/cloud aliases are excluded. The current configured alias remains visible if the catalog is unavailable, but Apply still verifies metadata.

Use **Apply settings** before starting a new review. Apply checks metadata only: no inference, download, source execution or repository change. Invalid settings or unavailable/unsupported models preserve the previous configuration. Output includes thinking tokens; more output or context is not a guarantee of quality or speed.

## Lifetime and safety

- Start the updated backend once with `--agent-model`; its CLI flags provide initial defaults. Models must already be installed on the server-owned Ollama endpoint.
- Web settings apply to the shared local backend while that process is running. They survive page refresh, but restarting the backend restores CLI defaults. They are not a persistent user account preference.
- Settings cannot change while a review is active. Stop and wait for the worker to finish before applying a different configuration.
- New generation settings produce a different provider identity. A checkpoint with a different model identity cannot resume under the new configuration. Old checkpoints and saved mapping analyses are preserved.
- The endpoint and credentials cannot be changed through the settings payload. Cloud aliases cannot be used to bypass the existing local-model validation.
- Changing settings does not rerun deterministic mapping or alter ranking weights. Reuse an intact saved analysis and start a new AI session.

## Next local trial

Use the [current Focused review trial](FOCUSED-COMBINED-REVIEW.md): Qwen3.5 35B-A3B for all roles, Thinking Off, Coding, context 32768, output 6144. Qwen3.8 import was blocked by disk space and further installation is deferred per lab guidance. Keep the same source/BPMN/candidate analysis. Start a new session after a model or prompt change. Offline checks do not establish better model accuracy.

Check model and Apply distinguish a missing Ollama alias, unsupported thinking, embedding-only models, remote aliases and changed model identity. Unknown transport failures do not automatically mean the VPN disconnected. Raw server responses and URLs are not shown.

## Implementation and verification

Configuration contracts and locking live in `internal/agentjobs/configuration.go`; Ollama construction stays in the server composition root. API routes expose settings and a local metadata catalog. The UI form is a separate `AgentSettings` component.

Regression coverage includes invalid settings, failed metadata preparation preserving the old model, no inference on configuration, active-job locking, cross-configuration resume rejection, strict JSON input and local generation-model catalog filtering. Existing architecture checks keep core packages independent from HTTP/Ollama adapters.

No application servers or browser were started for this change. Automated Go tests and frontend type/encoding checks do not prove browser layout or live lab model quality.
