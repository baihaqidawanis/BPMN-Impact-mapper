# Focused review: one combined audit

Updated 17 November 2026. This policy changes Focused mode only. Full repository review retains its existing scan/planning policy.

## Problem observed

The saved 9B session `6b59a2816212154c5d577bd6c012bd6b` repeatedly consumed per-batch review/revision budgets. Context failures caused more batch splitting. Reviewer sometimes objected that a proposed new guard did not exist in old source. The user requested cancellation; the backend confirmed PAUSED with completed checkpoints preserved. This is workflow amplification, not proof that 9B is inherently slower or worse than 35B.

## Current flow

Focused v12 distinguishes output-token exhaustion from malformed JSON. An exhausted multi-target Planner call partitions into existing per-target planning; an exhausted Analyst batch can split into smaller seed batches. Reviewer output-capacity failures follow the context partition path. A single-target exhaustion stays explicitly incomplete without spending another format-repair turn on the identical capacity problem. The API reports output capacity separately from VPN/timeout errors. New session required after BE restart; no partial JSON is accepted or recovered by guessing.

Focused v11 coordinates later Planner batches with unapproved prior proposals for the same files. Those contracts never count as source evidence or as Analyst verdicts, and they participate in child binding checks. Review transport preserves current_behavior claims for source comparison. Source-backed REVISE targets can receive the single bounded correction even when another target needs evidence; targets with NEED_EVIDENCE remain unresolved. No recommendation is silently dropped to improve precision.

Thinking trial timeout: each production agent inference call (including scan, README orientation, partitioned Planner and final review) now allows 10 minutes. The default Ollama HTTP client allows 10 minutes 15 seconds; metadata checks retain a 10-second limit. Explicit custom clients can impose shorter limits. Cancellation, checkpoint persistence and existing turn/output/context budgets remain active. Restart BE to apply; enabling Thinking does not increase the configured output-token budget.

Latest update: Focused v10 adds a target-level fallback when a complete file review still exceeds context. Each proposal retains its full steps, checks and source evidence. All detailed checks must pass before the cross-file contract review can approve the result. Completed target receipts are reused on resume. An oversized individual target or contract pass still returns an explicit incomplete review; approval is never inferred. New session required after restarting the backend.

Evidence-scope update: Focused v9 / agent-loop v12 keeps file-level source windows separate from named function targets. Exact function matches take priority. Final review receives evidence for all proposed targets, excluding unplanned retrieval inventory, while retaining all BPMN changes and process context. Restart the backend and start a new session after this update; do not resume a checkpoint bound to the previous policy.

1. Analyst inspects evidence and classifies each target in bounded candidate batches. Planner can request additional source within the existing tool/revision budget.
2. Planner prepares source-bound proposals. Host validation still rejects unknown targets, invented evidence, missing relevant targets and incomplete plan fields.
3. A prepared child is stored as NEEDS_REVIEW with `awaiting_combined_review: true`. It has not received semantic approval. The parent collects these drafts without calling a Reviewer per batch. Planning failures remain issues.
4. Reviewer audits the assembled plan once. It tries the complete input first. Only an actual context overflow triggers the existing file partitions and cross-file contract audit. These are parts of one final review phase, not promises of exactly one inference call.
5. Source-bound REVISE findings targeting Planner allow one correction restricted to those targets. It receives the prior draft, findings and verified evidence; it has no source tools, no Analyst handoff and no further revision cycle. NEED_EVIDENCE targets are left unresolved rather than guessed.
6. The corrected combined plan is audited again. Remaining problems produce NEEDS_REVIEW with the structurally valid draft retained. No semantic approval is inferred from a timeout, invalid response or budget exhaustion.

Review JSON format repair remains separately bounded. Coverage is processing, not accuracy. `batches_prepared` is not `batches_approved`; the web distinguishes them. Scores and references are not sent to inference.

## Light review

Light uses the same BPMN grouping and final review as Focused Standard. For each change it starts with the top three distinct files and adds the highest-ranked candidate for any missing frontend, backend or test role. It opens the remaining mapped candidates if the draft has no edit or leaves an available role uncovered. The candidate counter always reports the full mapped pool and the number processed.

Thinking is configured separately. Light has no whole-review deadline. Compare Light and Standard with the same model and analysis, and measure elapsed time, model turns, and file coverage against the scenario reference.

## Restart and trial

The running backend successfully accepted all three role models as `qwen3.5:35b-a3b-q4_K_M`, Thinking Off, Coding, context 32768, output 6144. That metadata-only configuration did not load weights or run inference. Restart is still required to load this new workflow:

```powershell
cd C:\BPMN-Impact-Mapper; $env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run .\cmd\bpmn-impact-mapper-server --agent-model qwen3.5:35b-a3b-q4_K_M --agent-context 32768 --agent-output 6144 --agent-profile coding --agent-review-json
```

Stop the existing backend with Ctrl+C before running the command. Keep the SSH tunnel open. Reuse analysis `89a7e1615e6e85aa107dfe9e04f6dbfd` only while its source and BPMN remain unchanged. Start a new Focused session. Old policy checkpoints cannot resume under the new binding.

## Verification and limits

Offline regression tests cover all-seed coverage, resume without replaying completed batches, no per-batch Reviewer, preservation of unapproved drafts, one targeted correction, no Analyst replay during correction, unchanged targets retained, remaining rejection never promoted to COMPLETE, missing source does not cause speculative correction, malformed output handling and policy-binding changes. API tests retain schema and evidence guards. Model accuracy and real latency require a fresh lab run; no accuracy gain is claimed from mock tests.
