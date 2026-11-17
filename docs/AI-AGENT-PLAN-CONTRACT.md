# AI Agents and developer plan contract

## Three roles, one configurable model

Analyst inspects approved implementation source and records per-change decisions with evidence. Planner writes a technical proposal; Reviewer checks evidence, requests bounded revisions and finally checks the assembled plan across batches. The Go orchestrator controls tools, budgets, checkpoints and cancellation. Model name belongs in runtime metadata, not the feature name. Changing to another supported Ollama model uses backend configuration; a different provider still requires an appropriate adapter.

LangChain, LangGraph and a separate agent platform are not required for these capabilities in this application. Introducing another orchestration framework would duplicate the current state machine unless a concrete missing capability justifies it. Do not add agents merely to increase the count.

## Experimental reasoning review (10 November)

The default remains ordinary schema review. `--agent-review-json` explicitly opts
into the v26 JSON-review policy; it does not select another model. Independent
`--analyst-thinking`, `--planner-thinking`, and `--reviewer-thinking` options accept
`inherit`, `on`, or `off`; inheritance follows `--agent-think`. Model capability
checks still apply. Changing these options requires a fresh AI session, while
the saved mapping remains reusable.

JSON review supplies the schema explicitly and validates all returned checks and
evidence on the host. For source-backed CHANGE proposals, the model sees actual
edits and checks alongside source rather than repeated Planner behavior claims.
NO_ACTION and the source-free final contract check retain the behavior statements
needed for verification. Full plans remain intact in storage and exports.

With JSON review enabled, detailed reviews run by file
before the compact cross-file check. Every call is budgeted/checkpointed and a
failed file cannot be approved by the final check. Completed file receipts survive
resume. Detailed source findings now survive the final combined export too.

This is **experimental**, not autonomous sign-off. The v23 source-control trial
produced 5/7 expected verdicts (6/7 valid): two specific defects were caught, an
unrelated-action change was missed, and one response exhausted output. The v24
runtime wiring preserves source-free contract inputs and selects the checkpointed
file-review path; a complete-case live result under that composition is pending.
Cold reasoning calls exceeded current HTTP limits in diagnostics; no timeout or
output default was raised. See [measured results](LAB-MODEL-PROBE-2026-11-10.md).

## Expected plan

One canonical JSON object renders one Markdown document. Include model/source identity, status, scan and final-review coverage, review issues, and a BPMN change coverage table. Group detailed recommendations by file, then by requirement. Each recommendation retains the BPMN element/change, function/component when supported, layer, action, evidence IDs, current/target behavior, proposed changes, implementation sequence and observable acceptance checks.

A useful recommendation identifies where validation belongs, which existing input is reused, when failure must occur relative to state mutation, affected UI behavior and how to test success and failure. It does not invent migrations/endpoints or generate code automatically. Uncertain requirements remain explicit; COMPLETE describes protocol completion, not independent correctness.

## Root README orientation

Full review first reads an ingestion-approved root README.md (case-insensitive).
A dedicated Analyst turn returns a background summary, with no candidate decisions.
The immutable orientation checkpoint is bound to source/model/input identity;
replaying a successful session reuses the summary. The raw README is capped at
8 KiB on a UTF-8 boundary, with explicit truncation metadata. Its summary is
limited to 1600 characters and treated as unverified background in every role.
The README is not added to implementation candidates or evidence. No README is
also valid. Nested documents are accessed only through approved targeted tools.
Failed attempts may be retried; "once" means one successful stored orientation,
not a guarantee of exactly-once network delivery across transport failures.

## Corpus policy

Ingestion excludes secrets, generated output, vendored dependencies and research leakage first. Exhaustive agent scanning additionally skips Markdown/MDX/RST/AsciiDoc and common dependency manifests/lockfiles. Schema, application configuration, implementation and tests remain eligible. Approved source tools can still retrieve a relevant dependency declaration or documentation fragment in targeted follow-up. This is not a guarantee that no prose is ever sent: Hybrid and focused mode have their own bounded evidence inputs.

Coverage counts only eligible bulk-scan files/chunks; it does not claim all physical files or accuracy. Full-scan binding includes the chunk manifest so changing this policy cannot silently resume a different corpus.

## Progress

Analyst progress counts successfully classified chunks. Planner progress counts finished batches, including their local review. Final Reviewer uses an activity indicator while its model call runs, then reports approved/needs review/incomplete. No percentage is fabricated for a model call. The three roles execute sequentially; final conflicts remain issues rather than automatic global rewrites.

## Scan identity binding

Scan responses contain only change_index, verdict and rationale. The adapter binds each decision to the one source-chunk candidate and its evidence ID. Missing, duplicate and unknown indices remain invalid. This prevents copying errors in long BPMN IDs/paths from masquerading as evidence failure; relevance and rationale still come from the model and require evaluation. A changed response protocol requires a fresh bound session.

## Focused multi-change batches

Focused review runs one bounded change session at a time, then assembles the decisions and plans for a final consistency review. All changes and the full before/after process structure remain available across the resulting plan. Each child retains the existing per-change seed limits and its own tool/turn/revision budget. The service still limits total focused runtime; resume reuses completed child checkpoints. The original top-three count is not reduced as a workaround for context overflow. Final consistency-review overflow produces an explicit review issue, never silent truncation.

Current per-file turns also carry compact related_proposals for coordination. These are future contracts, never source evidence. Local target_behavior is retained as an obligation. Related contracts and repair feedback are bound to receipt identities; long compound IDs are hashed to respect the unchanged filesystem limit. Non-thinking v26 controls still failed qualification (3/7); the current actual trial has no final export. See the measured report before enabling experimental review.
