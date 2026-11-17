# Agent request recovery

Session `715a9d6128a0ed4492ba759054c95630` remains stored under `.data/agent-sessions`. Do not delete this folder. No inference, session rewrite or server restart was performed for this repair.

## Findings

- The endpoint metadata check succeeded during investigation; this does not prove connectivity throughout the earlier failed generation.
- The saved session has both context-limit and output-limit failures in child batches, plus an interrupted Reviewer. These are distinct failure categories.
- No out-of-memory error was established. An empty `/api/ps` response only means no model was loaded at the time of inspection.
- The HTTP client allows 85 seconds and orchestration allows 90 seconds per model turn. These limits were retained. Disabling thinking is a controlled next experiment, not a guaranteed cure.

## Repair

- A generic PAUSED session no longer instructs the user to restore VPN as though network failure had been proven.
- Transport errors retain their cause for structured diagnosis without exposing raw endpoint details.
- Deadlines/cancellation while reading the response are no longer flattened into an ambiguous response error.
- Regression tests cover request/response interruption, privacy, and the existing save/resume path.

## Next trial

Stop the old backend with Ctrl+C, then start it with the same models, context and output budget, changing only Reviewer thinking:

```powershell
cd C:\BPMN-Impact-Mapper; $env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run .\cmd\bpmn-impact-mapper-server --agent-model qwen3.5:35b-a3b-q4_K_M --planner-model glm-4.7-flash:latest --agent-context 32768 --agent-output 6144 --agent-profile coding --reviewer-thinking off --agent-review-json
```

Keep the tunnel open. Reopen the same saved mapping analysis and start a NEW AI session, because the model generation configuration changes. Do not use Resume for this comparison. Retain the old checkpoint for inspection.

No BPMN upload is needed if the saved analysis includes the pair and source snapshot still matches. Do not rerun mapping merely to hide old errors. Compare source selection and technical plan quality separately from completion time/status. A context limit must remain visible; no requirement was silently removed to make the session pass.
