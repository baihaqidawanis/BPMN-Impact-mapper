# Agent repair after the v10 live run

## Observed failure

Uploaded plan `1c7ea3da6505d120200fb693ca00ae1c` came from session `f55153e0cf55ffaaac0d4edd82f63c02`, Coder 30B, prompt v10. It exported only `docs/API-CONTRACT.md`, remained NEEDS_REVIEW and omitted the three provisional implementation reference files. Existing source hashes still match the scenario reference. This reference is diagnostic, not independent ground truth.

- Core backend/test batch: eight identical search attempts. The original query had more than the source service's 12-word limit. The adapter kept forcing search until success, starving actual file inspection. The engine suppressed repeated execution but still consumed tool/turn budget.
- Other batches: planner repeated or omitted exact file/function targets. An enum of valid indices did not guarantee each required index appeared once.
- One documentation proposal passed model review despite describing insertion of a task and rewiring BPMN. The edited after process already contained that task.
- Reviewer objections sometimes demanded runtime integration of a BPMN task ID rather than checking software behavior.

## Changes

1. Initial repository search uses a host-derived behavior query respecting 12 words and 200 bytes. A failed search attempt no longer forces that same search on every subsequent turn; file inspection still requires actual successful source evidence. Free follow-up search schemas also encode the word limit.
2. Planner `plan.plans` transport is an object with one exact target-index key per candidate. RELEVANT keys are required, WEAK keys optional, rejected keys unavailable. Host converts slots back to canonical file plans and retains exact identity, evidence, action and coverage validation. Duplicate/unknown/noncanonical keys and mixed identities are rejected. Legacy array decoding remains strictly validated for compatibility.
3. A narrow host guard rejects explicit BPMN diagram-edit recipes assigned to non-diagram files. This is not a general semantic correctness checker. Legitimate documentation updates remain allowed.
4. Added-task/gateway results with only documentation proposals and remaining implementation candidates are flagged unresolved rather than silently treated as sufficient software coverage. No code edits are invented to fill a quota.
5. Instructions distinguish the already-edited after blueprint from the existing code; task IDs are requirement labels, not assumed runtime hooks. Reviewer rejection must identify a concrete missing condition, entry point or source fact. Documentation cannot substitute implementation.
6. Redundant common instructions were shortened so the combined BPMN transport regression remains within its configured context budget.

Provider identity is now `ollama-agent-json-v11-*`, engine protocol `agent-loop-v6`. Existing failed sessions and exports are preserved. Restart the backend and start a NEW session; do not resume old protocol checkpoints.

## Offline verification

- `go test ./internal/llm ./internal/planning ./internal/agenttools ./internal/agentjobs ./internal/api`
- `go vet ./internal/llm ./internal/planning`
- Regression tests cover keyed planner targets, same-file/different-function identities, duplicate/unknown slot rejection, original evidence binding, bounded searches, inspection after failed search and misplaced diagram plans.
- A local HTTP stub completes Analyst -> Planner -> Reviewer through the actual Go engine, including host restoration of two distinct targets in the same file. This proves protocol integration with scripted responses, not model reasoning.
- Seven captured batch states reconstructed across three roles: 21 local transport checks pass. Maximum request 26,937 bytes, below the conservative 28,160-byte budget. Replay replies are scripted and do not prove improved mapping accuracy.
- No browser, frontend/backend service startup, lab inference, downloads, ERP edits, commit or push.

## Remaining test

Reopen the saved add-task analysis, verify the Coder model, start a fresh Focused review. Inspect actual source lookups, rejected actions, proposed UI/backend/test changes and final review issues. Keep existing reference and metrics unchanged until the new output is evaluated. Do not infer better precision/recall from these offline checks.

## Follow-up: fa98e249 failed during Analyst

The first flow-change batch stopped in Analyst with 8 consumed tools and 11 turns, not in the new keyed Planner transport. The query used the generated edge ID and returned unrelated source hits. The generic repair message incorrectly described Planner fields to Analyst. The precise rejected response was not persisted; do not claim the historical formatting failure is fully diagnosed.

Provider v12 / engine v7 now derives flow searches from the endpoint task labels and uses role-specific repairs. Bounded adapter rejection details are saved for the next failure, without raw response logging. Output-length, empty-content, model-role mismatch and unfinished-response errors are separate categories. llm/planning/agentjobs tests and vet pass. Restart backend and create a new session; do not resume the failed old version. Actual model quality remains unverified.

## Latest: v14 / engine v8 / action-window source tools

The old 9B run used prompt v6, processed 10 displayed candidates in seven batches and selected six files including all three provisional references (precision 50%, recall 100%). It still remained NEEDS_REVIEW with technical errors. The newer Coder v12 run exported one unrelated test plan. Models, quantization and program versions differ; this is not a controlled model comparison. Bigger parameter count does not establish better task performance. No model was changed or benchmark overwritten during this repair.

### Reviewer overflow

Oversized child Reviewer turns now use the existing persisted per-file review plus cross-file contract check. Every extra model turn is charged and saved against the child's existing budget before the call. Completed file receipts are reused after an interruption. A rejected file, insufficient budget, cancellation or oversized individual partition never implies approval. Revision routing from file review is preserved. No source/evidence is silently dropped to fit context.

The failed four-file reviewer checkpoint was replayed through the actual adapter and engine against localhost scripted replies: original whole-turn overflow reproduced, four exact file proposals reviewed separately, all four retained for cross-file review, eight budgeted engine turns. Captured WEAK decisions are promoted ONLY inside this test fixture to exercise transport under the new selection policy; this is not a production decision or accuracy evaluation.

### Source grounding and selection

Source inspection prioritizes a contiguous eight-line dispatch window when a literal case label matches a requirement query word. This exposes the current condition/state mutation for the relevant operation rather than disconnected keyword lines. It is a textual anchor, not an inferred dependency graph; cross-file callers still need source tools. A synthetic publish/archive case verifies the requested branch wins over unrelated reason checks.

Final agent proposals now require RELEVANT source decisions. WEAK candidates remain visible as unresolved and cannot be planned until the Analyst obtains sufficient support. This deliberately replaces v11's optional WEAK planning policy to prevent unsupported helper/test proposals from entering the final recommendation. Documentation may still be a justified auxiliary target; it cannot stand in for unresolved implementation coverage. Prompts require evidence for operation-to-helper paths and reject assumptions that a shared helper is the requested operation's entry point.

### Verification and limits

- Tests pass for llm, planning, agenttools, agentjobs and API; vet passes for the same packages.
- Regressions cover partition resume/cached file reviews, rejection withholding, remaining turn budget, enum verdicts, keyed targets, weak-target abstention and branch-local source windows.
- No browser, actual lab inference, ERP source changes, model switch, benchmark update, commit or push.
- Real model reasoning is not validated by scripted replies. After backend restart, run one NEW Focused session with the same saved analysis. Use the exact same application version/scenario to compare 9B and Coder later if needed; do not compare v6 versus v14 as a model-only experiment.

## V14 actual result and v15 freeze

Plan 50917a4a/session 3f1bcb35 exported zero proposals because all Analyst decisions were WEAK and v14 no longer plans uncertain targets. This prevents earlier speculative plans, but does not constitute improved mapping and does not explain away the lack of useful output.

The real checkpoint exposes a source-window merge defect: inspect_source returned lines 134-141, beginning with the submit branch and DRAFT guard. The per-item FIFO evidence merge retained the two original search hits and the LAST six returned lines, losing precisely the operation boundary/guard. Later search hits could also replace inspected proof.

V15 preserves up to two original anchors and the FIRST six new source-window items, in order, within the existing eight-item cap. A later search hit cannot displace an inspected window. An AgentEngine regression with a generic publish/archive dispatcher verifies the dispatch, guard and state mutation survive merging. Analyst instructions now directly define affected-file classification: missing new behavior can make its existing implementation path relevant; relevance is not a claim that current code complies. WEAK remains unresolved; no automatic relevance promotion is added.

### Controlled comparison next

Freeze v15, snapshot/source hashes, saved analysis, Focused mode, top-k, context/output budgets, temperature and thinking setting. Start fresh sessions for qwen3.5:9b-q8_0 and bpmn-qwen-coder:30b-q4_k_m, preserving both outputs. Evaluate file selection against the same provisional reference and technical correctness separately. Record failures, unresolved rate and latency. Neither model receives the reference. No evidence establishes that parameter count, quantization, or model family alone caused the regression. No further prompt/policy changes between the paired runs.

Verification remains offline: affected package tests/vet and whitespace checks; no lab inference, browser, source-case edits, model deployment or benchmark overwrite. Model quality remains to be measured.
