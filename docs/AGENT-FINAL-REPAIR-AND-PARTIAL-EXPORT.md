# Final review repair and partial export

## Failure addressed

The lab returned `duplicate review JSON field` repeatedly in final file review. Repeating the unchanged prompt did not repair it, and a paused aggregate hid otherwise inspectable proposals.

## Updated behavior

- Experimental Reviewer policy v28 includes a deterministic transport rule: after explicit invalid-output feedback, source-backed review retries use constrained schema decoding. Contract-only passes already use the schema. Duplicate keys, invented evidence and missing criteria are still rejected by the host.
- Final consistency receipts save their specific validation feedback. A retry receives this feedback from the same receipt, retaining the same three-attempt limit. Only invalid output is retried automatically; network interruption pauses rather than spinning.
- Per-candidate AgentEngine retains its existing output-repair budget; it does not acquire three extra retries through nested review calls.
- A final review that exhausts retries produces NEEDS_REVIEW, not silent approval. A structurally validated partial plan can be exported without requiring every RELEVANT target to have a plan. Missing targets stay explicit in coverage and issues; target identity, evidence, layer and text checks are not bypassed.
- **View partial plan** exports a saved checkpoint without loading a model, changing its status, resetting counters or executing repository source. It works for older checkpoints too. These exports are always NEEDS_REVIEW, even when some file proposals passed earlier review.

## Recovered historical result

Session `14a10c2d0f150c1c50c8d1f9945852de` was exported locally to `outputs/agent-recovery-14a10c2d/`:

- `partial-plan.md` and `partial-plan.json`: two parent assessments, both NO_ACTION. There is no accepted edit proposal for ADD_TASK in this parent result.
- `batch-drafts.md`: 14 stored child draft assessments, including withheld or rejected proposals. This is a diagnostic record, not a final selected-file list or benchmark result.

Original checkpoint bytes were preserved. No new inference was performed to generate these files. This recovery exposes the actual incomplete model result; it does not claim the agent solved the scenario.

## Verification

Regression tests cover final retry feedback, reuse of successful receipts, duplicate-field rejection during schema repair, bounded attempts, partial export preserving unresolved targets, rejection of invented evidence, and exporting paused checkpoints without inference or checkpoint mutation. Go test/vet, frontend type/encoding and diff checks are required.

The old backend must be restarted once for the new API/UI behavior. Old sessions can be inspected/exported immediately; a live generation trial under v28 requires a new session because the model protocol identity changed. Reuse the saved mapping; do not upload BPMN again merely to inspect an old draft.

Browser layout and live model quality are separate verification gaps. The tunnel endpoint was unavailable during the final recovery checks, so no v28 live quality claim is made.
