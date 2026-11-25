# Evidence-based agent review and developer plans

## Scope

Focused review includes every displayed candidate on every BPMN change. Top 10 on each of 20 changes means up to 200 seed entries, not three global files. Repeated files can appear under different requirements. Source tools can investigate additional approved files. Full review separately scans the complete ingestion-eligible source corpus; secrets, dependencies and excluded evaluation material remain excluded.

## Execution

Current Focused policy: [one combined audit](FOCUSED-COMBINED-REVIEW.md). Prepared Analyst/Planner child drafts are collected without per-batch semantic review. Reviewer audits the combined proposal, with at most one targeted Planner correction. The historical execution notes below describe the earlier pipeline and are retained for interpreting old sessions.

Focused seeds are split into two-candidate child sessions per change, retaining complete before/after semantic graph context. Each child runs Analyst, Planner and Reviewer and has its own bounded tool/turn/revision budget. Completed child checkpoints are reused on resume. The parent merges evidence, preserves diagnostics, and exports one plan. Only COMPLETE child drafts are promoted into that combined plan. Rejected or interrupted drafts remain in child checkpoints and produce explicit withheld-batch issues. Overlapping file proposals are flagged; conflicting proposals are not silently resolved by dropping evidence.

No default whole-job ten-minute cutoff is applied to focused sessions; individual model turns remain bounded and Stop/cancel is available. Increasing coverage can significantly increase latency. Progress reports initial candidates processed and completed candidate batches; these counters do not prove correctness or complete-file comprehension. Source access logs distinguish search hits from returned read excerpts.

## Agent contracts

1. Analyst searches approved source and inspects requirement-related windows for candidates. A search hit is not sufficient proof of validation behavior. It returns RELEVANT/WEAK/REJECT and cited evidence; unknown facts stay unknown.
2. Planner converts observed gaps into a developer proposal with current_behavior, target_behavior and design_rationale, then intended changes, ordered implementation steps, observable acceptance checks and optional assumptions. Existing names are cited; locations for new functions are explicitly proposals.
3. Reviewer checks whether the proposed behavior addresses the requirement in the correct software layer, preserves guards and state/failure behavior, and cites adequate evidence. It can request source follow-up or planner revision. XML gateway implementation inside a migration is inappropriate unless an actual runtime/schema requirement supports it.

This uses task decomposition, tool-grounded evidence and bounded critique/revision. It does not request or expose private chain-of-thought. Rationale is a concise justification with evidence, not hidden reasoning. Longer output is not automatically better.

## Markdown contract

The canonical JSON generates Markdown grouped by file, with linked BPMN requirements, source evidence, current behavior, required behavior, design rationale, assumptions/open decisions, implementation steps, proposed changes and acceptance checks. There are no code patches or execution claims. Old saved plans can omit new fields; fresh model requests require the three behavior/rationale fields.

Example shape (illustrative, not a case-study rule): describe the observed submit handler and missing input validation; propose reusing its existing parameter; validate before status mutation; use the established error response; preserve unrelated actions; add invalid/valid input and permission/state regression checks. Do not merely say add a gateway or update the schema.

## Bounds and validation

Model schemas enforce structured fields and engine validation binds exact change/file/function/evidence tuples. Incomplete detailed output gets one bounded repair attempt. Final combined review first attempts the entire plan. On context overflow it checkpoints detailed reviews grouped by file (all proposals for the same file stay together), then reviews a compact cross-file contract view. That view retains every target behavior, summary, proposed edit and assumption; current behavior, detailed steps, rationale and acceptance checks have already been reviewed in the file phase and are explicitly omitted from the contract phase. All process graphs and changed requirements remain. A failed file review prevents contract approval. Completed receipts and context-limit results are reused on resume. A single file group or the compact contract view can still exceed context; this yields NEEDS_REVIEW without silent truncation. Drafts from failed child reviews are withheld from the combined plan. A failed final consistency check keeps remaining proposals explicitly provisional. New provider/engine identities require fresh sessions after restart.

## Offline verification

Ten-seed focused regression checks all seeds are processed and an interrupted middle batch resumes without replaying completed batches. Service regression checks displayed Top 10 are not reduced to three. Markdown regression verifies behavior/rationale/assumptions survive rendering. API tests reject invented targets/evidence and preserve partial failure diagnostics. Mock tests establish protocol behavior, not Qwen reasoning quality. Actual model quality and latency must be tested in the lab.


## 2026-11-05 regression corrections

Provider prompt identity v7 requires action-specific behavior tracing through UI,
handler, service and state changes when relevant. It distinguishes creation from
submission, checks actual parameter types, preserves unrelated actions and rejects
schema changes unsupported by a persistence gap. These are general code-review
instructions, not case-specific file mappings or a ground-truth answer key.

Focused binding v3 and full-repository binding v6 require new sessions after
restart. Historical plans and their saved evaluations are unchanged. NEEDS_REVIEW
Markdown now starts with an explicit unapproved-draft notice. Withholding a draft
reduces exported coverage and must not be presented as proof that a file is irrelevant.

Offline regression checks cover rejected drafts in both engines, interrupted file
review and resume, preserving all cross-file contracts, and cross-file conflict
rejection. These verify orchestration; live model quality still requires a new run.


## Coder identity and delta regression (2026-11-07)

The 30B Coder run ff9997c692f60b387f5e2030b3352e55 used prompt v7 and
exported only business rules and migration proposals. All three primary
ReasonReview reference files were absent from its exported plan: provisional
selection precision/recall/F1 were zero. Manual plan assessment was 25/100.
This is not a controlled model comparison: model, prompts and withholding policy
changed from the earlier 9B run. Preserve both historical outputs.

Provider v8 sends host-derived change_facts containing original before/after IDs
and presence. New IDs absent from before are expected additions. Shared graph
entries belong to both versions. The complete graphs remain available; no
business-rule truth or case-specific source mapping is added to inference.

Verification and planning schemas request target_index and evidence_indices,
annotated on candidate/evidence entries in each immutable turn. The host restores
exact original change/file/function/module/layer and evidence IDs. No automatic
correction guesses a function from its name. Invalid/mixed indices are rejected;
legacy named responses still undergo the existing engine tuple validation.
Persisted plans, source logs and exports retain human-readable original IDs.

The Reviewer is directed to compare observed current source with intended after
requirements. It must not require a proposed new guard to already exist in old
code/tests. A BPMN-only edit proposal or column-presence check is insufficient as
a software implementation plan. These instructions do not guarantee the model
will reason correctly; actual v8 output must be tested after backend restart.

Three roles remain Analyst, Planner, Reviewer. The Go engine is the orchestrator:
it controls routing, source tools, bounded revision, checkpoint/resume, candidate
batching and final consistency. A fourth model agent or external orchestration
framework is not introduced. Reviewer approval remains a model judgement, not
independent ground truth or permission to execute source changes.

Offline tests cover indexed decision/plan round trips, invalid/mixed references,
null evidence, immutable before/after presence and source binding. All seven
captured Coder child checkpoints replay through a localhost model stub. This
proves transport/protocol behavior, not a quality improvement in live inference.


## V8 live outcome and V9 action binding (2026-11-07)

Uploaded plan ebd003e57fb26747b1c947c7c8de9565, session
f78cb1f48ec714fdbff8c82f7d8eccd3, used Coder and provider v8. It exported
zero plans and remains NEEDS_REVIEW. Analyst child decisions mark all three
primary reference files RELEVANT, but five planning batches failed action
validation with values such as modify, add or ADD_TASK. The indexed schema
introduced by the adapter left action unconstrained, a host integration defect.
This outcome cannot support a clean judgement of model plan quality. Named target
mix-ups were not the dominant failure in this run, but semantic quality is not
established by successful identity binding.

V9 removes action from new indexed plan responses, binding it to the original
candidate action in the host alongside identity. Explicit legacy action values
remain strictly validated; invalid taxonomy/case strings are never normalized or
promoted to MODIFY. Technical proposals remain model-generated and subject to
review. Final consistency returns an explicit incomplete result without a model
call when no plans survived. Regression tests cover action omission, invalid
legacy actions, schema field exclusion, HTTP round-trip and empty-plan handling.

Restart backend and start a fresh Focused session to test V9. Historical V8 output
remains unchanged; do not claim improved final precision or plan quality before
the new inference exists.


## Relevant versus weak planning coverage (v10, 2026-11-07)

Deeper audit found the loop required one file plan for every non-REJECT decision,
including WEAK candidates. This encouraged speculative documentation/schema
plans to meet an output quota. V10 keeps RELEVANT coverage mandatory but permits
WEAK omissions. JSON schema minItems counts RELEVANT, maxItems counts non-REJECT;
engine tuple/evidence validation and relevant-coverage checks remain authoritative.
Unsupported candidates remain weak/unresolved, not automatically irrelevant.
The combined focused view reports unresolved coverage rather than false completion.

Protocol agent-loop-v5 and provider prompt v10 require a new session. Indexed
host-bound actions from v9 are retained. Tests prove WEAK-only source does not
force a fake plan and missing RELEVANT plans are still rejected. Seven captured
v8 failure checkpoints replay locally including their actual feedback; replay
only establishes transport/budget behavior. Full Go test suite and vet pass.
Live plan quality remains to be measured; no lab inference was started here.
