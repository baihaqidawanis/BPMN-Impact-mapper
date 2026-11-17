# Analyst comparison - 10 November 2026

## Controlled comparison

Installed local models: qwen3.5:9b-q8_0 and qwen3.5:35b-a3b-q4_K_M. Two frozen inputs: one backend target, then the original five-target batch (backend, integration test and three documents). No reference answer was sent. This is not the full scenario and does not measure full-case recall.

Both models used context 32768, output 4096, baseline sampling, thinking disabled, the same source/graph/system/schema/options, and sequential loading. Input hashes matched between the two models for each variant. Quantization and architecture differ, so this compares the installed configurations rather than isolating parameter count.

## First run: v17

Artifacts: .data/model-probes/20261110T013211/.

- 9B: valid decisions on both inputs; backend/test RELEVANT, three documents WEAK on the batch.
- 35B: valid isolated result; batch classifications matched 9B, but referenced evidence index 7 for a document containing only indices 0-6. Adapter correctly rejected it.
- The shared Analyst array schema allowed a global maximum of 7, even for shorter per-target evidence lists. This was a transport constraint deficiency as well as a model reference error, not proof that 35B reasoned worse.

## Repair and matched rerun: v18

Analyst decisions now use required keyed slots, with a separate evidence bound for each target. Original candidate/evidence identities remain host-bound. Unknown references are still rejected; none are clipped or silently corrected. Regressions cover unequal evidence-list lengths. Full repository chunk classification retains its separate change-index schema. A fresh agent session is required after this prompt/protocol change.

Artifacts: .data/model-probes/20261110T013614/.

| Model | Single backend target | Five-target batch |
|---|---|---|
| 9B Q8 | Valid, RELEVANT | Valid; all five RELEVANT |
| 35B-A3B Q4 | Valid, RELEVANT | Valid; backend/test RELEVANT, three documents WEAK |

Single observed times: 9B 20.25s cold / 12.26s warm; 35B 44.14s cold / 14.52s warm. Loading is included where applicable. These are not repeated latency benchmarks.

35B was more selective toward implementation in the v18 batch. Neither protocol validity nor matching these classifications proves all rationales correct. Some wording remains imprecise. The 9B classifications also changed between v17 and v18, demonstrating that output contract changes can affect judgment even with temperature zero. Do not combine these runs into a model-only leaderboard or claim global superiority from this sample.

## Next full-case trial

Trial configuration: 35B Analyst (baseline), GLM-4.7-Flash Planner (coding), 35B Reviewer (coding). This is a candidate configuration, not a validated winner. The saved inventory analysis supplies 8 before nodes/8 flows and 9 after nodes/9 flows, with 3 changes. Focused review includes all saved candidates and approved source discovery; it is not exhaustive model reading of the full repository.

The full-case trial under `.data/model-probes/pipeline-20261110T013951/` ended PAUSED after 10 orchestration turns, 4 source tool calls and 2 revisions. The model connection interrupted a Planner retry; no final approved plan was exported. The checkpoint is preserved. Protocol-valid intermediate responses do not establish final plan correctness or full-case Precision/Recall. Historical benchmark results and user root reports have not been overwritten.

## Web report

Research now includes **Why these models?**, with a provisional role configuration, quantitative tables and short source-review limitations. The downloadable sanitized snapshot is `web/public/benchmarks/lab-model-selection-2026-11-10.json`; rendering is in `web/src/components/ModelSelectionSummary.tsx`. It contains no source excerpts, prompts or credentials. Opening it makes no model call and does not change runtime routing.

The Analyst table uses matched v18 inputs. The Planner table uses the matched v16 partial-case comparison. The Reviewer table uses the latest clarified v17 control comparison, with 9B excluded because it was not rerun under that exact prompt. These separate studies must not be pooled into a single accuracy leaderboard. Selection remains provisional until complete-case tests and independent assessment support it.

## Resumed full-case trial

Artifact: `.data/model-probes/pipeline-20261110T020518/`. Checkpoints were copied from the interrupted trial before resuming; originals remain unchanged. All seven batches finished. Final status: **NEEDS_REVIEW**. No complete valid plan could be exported because relevant flow-change targets had no approved file plan. The diagnostic test correctly failed; 69/69 structurally validated model responses did not imply a valid final plan.

Cumulative charged work: 87 orchestration turns, 35 tools, 6 revisions. This resume logged 69 actual inference calls, 1286.6s total model-call time, including 750.9s loading (58.4%). These single observed durations are not repeated speed benchmarks. Role switching repeatedly loaded weights.

The assembled ADD_TASK draft selected the three provisional reference files, giving draft-subset file precision/recall 3/3 = 1.0. This is **not** whole-scenario accuracy or plan correctness: two flow changes are unlabelled by that reference, complete export failed, and the draft has ambiguous error handling, missing whitespace/state-preservation checks, incomplete success-test fixture updates, and potentially redundant frontend state. MAP/MRR are undefined for this unranked draft. Private evaluation: `draft-selection-evaluation.json` beside the run.

### Repairs after observing this run

- Provider v19 separates source facts from draft/feedback claims, prohibits requirement-unrelated architecture cleanup, preserves unrelated action contracts, and requires explicit success-fixture updates when validation tightens.
- Source tools v10 attach bounded declaration evidence from the approved snapshot. Declaration/type anchors survive action-window merges; this addresses discovered-function parameter evidence being evicted. Regression checks retain both the parameter declaration and state mutation.
- Focused v4 groups up to ten seeds within a 12000-byte candidate payload, retaining every seed and a discovery batch for unseeded changes. Actual model context guards and existing partitioning still apply. A single oversized seed is retained, not truncated. Resume tests cover all 25 synthetic candidates exactly once across interruption; identities prevent resuming v3 under v4 rules.
- No ERP changes or source execution. New identities require a fresh live session, while the saved BPMN analysis remains reusable.

## Matched Reviewer controls after v19

Artifact: `.data/model-probes/roles-20261110T022758/`. Same two inputs/options/schema across installed 35B Q4 and 9B Q8 configurations; source-authored reasonable plan and deliberately invalid Go string nil guard. These are partial controls, not full-case Precision/Recall. 35B reused resident weights, so observed latency is not a fair cold-start comparison.

| Model | Reasonable control | Invalid control | Observed times |
|---|---|---|---|
| 35B-A3B Q4 | Accepted | Rejected; identified string contract conflict | 9.76s / 10.08s |
| 9B Q8 | Accepted | Incorrectly accepted | 30.96s / 17.66s |

35B remains the provisional Reviewer. This does not establish general reliability; the v18 full case had false and missed rejections. Inputs also differ from the older v17 controls, so improvement is not a prompt-only causal result. The web model-selection report includes this separate study, without overwriting historical results.

Subsequent v19 trials are recorded below; none establishes a fully correct, approved complete-case plan.

## Capacity trials and current policy

- `.data/model-probes/pipeline-20261110T023111/`: provider v19, source tools v10, focused v4. NEEDS_REVIEW, 30 turns, 16 tools, 1 revision, 28 inference calls. A draft export exists, but the ADD_TASK seed group exceeded the context guard. Model-call total 449.4s including 231.0s loading; wall time 457.77s. Its shorter duration does not mean better complete-case performance.
- `.data/model-probes/pipeline-20261110T024058/`: same role models/profiles, focused v5 context splitting. All 10 seeds processed across 5 completed batches, but final NEEDS_REVIEW and no valid complete export: relevant ADD_TASK targets lacked approved plans. 79 turns, 43 tools, 4 revisions, 73 actual inference calls. Model-call total 1208.7s including 675.2s loading; wall time 1225.08s. Context splitting worked, but planning/review partitions exhausted their child turn budget and model judgments remained inconsistent.
- Current **focused v6** also splits multi-seed groups after structured partition-budget/turn exhaustion, preserving failed work in counters/checkpoints and leaving each child budget unchanged. Regression tests cover context failure, interruption/resume, budget failure, single oversized seeds and exact seed coverage. The v6 complete-case lab outcome is **not yet measured**. Do not label v5 results as v6 validation.

## Matched three-target Planner probe, v19

Artifact: `.data/model-probes/roles-20261110T030153/`. Source/graphs and captured Analyst decisions were held constant for backend, integration test and frontend targets. No reference answer or previous draft was sent. Each model received the same messages/options/schema hash. This tests draft quality with supplied targets, not file retrieval or autonomous target selection. One call per configuration; 35B reused resident weights, so observed time is not a fair cold-start speed comparison.

| Model | Format | Observed time | Provisional source review |
|---|---|---:|---|
| 35B-A3B Q4 | Valid, unapproved | 29.40s | Concrete non-whitespace backend guard and success-test fixture update; misses explicit frontend payload-condition change, with unverified error/persistence details and incomplete failure-state checks. |
| GLM-4.7-Flash Q4 | Valid, unapproved | 47.96s | Concrete backend guard; misses explicit existing success-fixture update and assumes submit already forwards reason. |
| Qwen3 14B Q4 | Valid, unapproved | 54.09s | Incorrect NO_ACTION for all three targets. |
| Qwen3.5 9B Q8 | Valid, unapproved | 60.94s | Invents TransitionSubmit, fragmented steps, incorrect frontend NO_ACTION and unrelated action claims. |

**No model produced a complete correct proposal under this source review.** Both 35B and GLM miss the crucial frontend conditional: current actionReason forwards reason for reject/cancel/reverse and uses empty text for submit. Validation alone does not fix the payload. These are qualitative assistant assessments on a development case, not independent grading or an accuracy percentage. Coder was not retested under this setup; 27B is unverified/unregistered, not evaluated.

For the next trial, 35B is a provisional Planner candidate as well as Analyst/Reviewer. This is not an automatic backend routing change. A shared-weight configuration may reduce model switching, but an all-35B complete-case run has not been measured. Three agent roles remain sufficient for the observed scope; adding a fourth model would not by itself repair missing source facts or faulty plan-review judgments.

Verification: Go tests/vet and frontend type/encoding checks are recorded separately from model quality. No ERP source was executed or edited. User root reports were preserved. Raw source/model payloads remain in ignored private `.data` artifacts.

## v20: input-path planning and extended Reviewer controls

Planner/Reviewer instructions now distinguish current source behavior from intended changes. The review contract checks compatibility of types/signatures/callers while allowing behavior changes explicitly required by after BPMN; it no longer treats agreement with old state behavior as the contract criterion. Planner instructions trace producer conditions, default values, payload and consumer. These are generic rules, not ERP-specific mapping rules. Provider identity is v20; a fresh agent session is required.

Artifact `.data/model-probes/roles-20261110T032101/`: the 35B fixed-target Planner draft now explicitly changes the frontend branch to forward the captured reason for submit, and updates existing successful test inputs. It still omits whitespace handling and has incomplete error/failure-state details. One observation does not establish a correct complete plan or prompt-only causality.

Reviewer controls are source-authored proposals, not output from the mapper: (a) a reasonable three-file proposal, (b) a Go-string nil-guard defect, (c) a proposal that adds validation but explicitly leaves the producer payload condition unchanged. Reference approval labels are used only after inference. The expected outcome for (c) is rejection: valid submit still sends empty text despite the proposed backend guard. Fixed target order/case checks prevent applying these source-authored controls to an unrelated scenario.

### Matched extended controls, thinking disabled

Artifacts `.data/model-probes/roles-20261110T032101/` and `roles-20261110T033651/`. All three message/schema/options hashes match across the four installed configurations. Profile coding, context 32768, output 4096. One call per control/model.

| Reviewer | Accept reasonable | Reject nil defect | Reject payload defect | Expected verdicts |
|---|---|---|---|---|
| Qwen3.5 35B Q4 | Yes | Yes | No | 2/3 |
| GLM-4.7-Flash | Yes | No | No | 1/3 |
| Qwen Coder 30B Q4 | Yes | No | No | 1/3 |
| Qwen3 14B Q4 | Yes | No | No | 1/3 |

35B's nil-defect rejection rationale is unreliable: it refers to absent observable test outcomes even though the proposal contains acceptance checks, rather than clearly identifying the type error. Correct verdict counts do not establish correct explanations. No tested Reviewer passes the complete control set. This is a development diagnostic, not general accuracy or independent adjudication.

### Thinking/output budget experiment

- `roles-20261110T032525/`: 35B thinking enabled, output 4096; all three final outputs truncated before a usable JSON response (102.02s, 55.08s, 51.22s). These are invalid responses, not three semantic misclassifications.
- `roles-20261110T033020/`: thinking enabled, output 6144; two controls attempted. Accepted the reasonable proposal (135.97s including 46.40s cold loading) and incorrectly accepted the missing-payload proposal (76.88s). The nil control was not attempted; do not combine a result from another configuration to claim full coverage.
- Thinking did not improve the payload verdict on this sample. The cold 6144 call exceeds current production turn/client time limits. No thinking default, output default or runtime timeout was raised. Hidden reasoning is not included in the published report.

### Explicit control gate

`internal/evaluation/reviewer_controls.go` evaluates a declared suite separately from response formatting. Missing, invalid, duplicate, unknown or incorrect decisions fail qualification. Tests cover false approvals, truncated/missing output and incomplete/duplicate suites. This decision gate is necessary on these controls, not sufficient to prove correct rationale or unseen-case performance.

`TestSavedReviewerControlGate` (labdiagnostics) replays saved decisions with no model/network calls. With `BPMN_REVIEW_CONTROL_STRICT=1`, it correctly fails all tested configurations on the expanded suite. The private summary is `.data/model-probes/reviewer-control-gate-2026-11-10.json`; a sanitized summary is available in Research. This is a research/release check; it does not silently disable or reroute runtime models. A comparison diagnostic completing successfully is not a quality pass.

### Conclusion and next boundary

Keep Analyst, Planner and Reviewer; adding another agent/framework has no supporting benefit from these tests. 35B remains an experimental Analyst/Planner candidate, with a measured sample-level Planner improvement. Reviewer selection is not finalized. The next development target is stronger evidence-backed data-flow/plan consistency assessment and a control suite that catches both missed defects and false rejections, before another long complete-case claim.

The lab's current Ollama tags do not include a registered 27B model. Its folder identity has not been verified. Direct SSH is blocked in this execution environment; the existing Ollama tunnel works. No remote model was imported or downloaded, no ERP source was edited, and no application server/browser was started.

## Comparison-receipt audit experiment

Added an explicit comparison of actual implementation items versus desired outcomes. A Reviewer selects one original source-evidence index and one original proposed-change/step index; the host binds their excerpts, adds the comparison evidence ID, and rejects missing/out-of-range/null edit references. Summary, target behavior and acceptance checks cannot substitute for implementation edits. NO_ACTION instead references current behavior. A fifth valid_path check asks whether a valid input can reach the consumer under the actual proposed edits and unchanged dispatch conditions.

Selected excerpts persist in canonical JSON, Markdown and the web Reviewer checks. Markdown uses literal code spans with safe backtick fences; React renders excerpts as text. Contract compaction preserves NO_ACTION current behavior so its comparison basis is not erased. These are auditable observations, not independent semantic proof.

Artifact `.data/model-probes/roles-20261110T041553/`: 35B coding, non-thinking, output 4096, fixed three-target controls. Accepted the reasonable proposal (65.63s), incorrectly accepted the nil-type defect (17.82s), and incorrectly accepted the missing payload edit (16.43s). All responses were structurally valid, but only 1/3 expected verdicts matched. The prototype is therefore **not adopted as default**. Captured identities retain a legacy v20 display prefix despite their changed comparison contract hash; captures are unchanged. The explicit experimental policy is now labelled v21, with separate identity and a disabled-by-default configuration flag.

Use `--agent-review-comparisons` only for experimental review; the regular four-check policy remains default. Role diagnostics use `BPMN_PROBE_REVIEW_COMPARISONS=1`; complete-case diagnostics use `BPMN_LAB_REVIEW_COMPARISONS=1`. Both require fresh sessions. No inference defaults were silently changed or model qualification claimed from valid formatting.

Regression tests cover actual-edit references versus desired outcomes, source/draft index bounds, no-source consistency checks, NO_ACTION comparison retention, literal Markdown excerpts, and separation of baseline versus experimental policy. No new ERP source changes or external dependencies.


## Per-file Reviewer diagnostic

Artifact `.data/model-probes/roles-20261110T061205/`: same fixed source-authored controls, scoped to one candidate file and its Analyst decision per request. Full before/after graphs remain available. Model 35B Q4, v20 baseline review policy, coding profile, thinking off, output 4096. This is a diagnostic-only switch (`BPMN_PROBE_PER_FILE=1`), not a production scope change.

All three reasonable single-file proposals were accepted; both planted defects were also accepted. Thus 3/5 expected verdicts matched, but **0/2 defects were rejected**. The all-approval behavior is not a qualified Reviewer. Observed times: 44.57s for the first/cold call, then 6.23s, 7.53s, 4.38s and 4.29s. Reducing review scope alone did not solve the issue; no per-file default was adopted.

Offline scoring accepts an explicit host-only label map through `BPMN_REVIEW_CONTROL_DEFINITIONS`. The five file-scoped controls must not be scored against the older three-control IDs. Labels remain outside model requests. Saved definitions and gate summary are beside the private results. Missing/null/nonboolean labels fail loading. Replay produced valid 5/5, correct 3/5, passed=false. This is a verdict control, not a claim of 60% real-world plan accuracy.

## Small value-flow interpretation diagnostic

Artifacts `flow-20261110T062000/` and `flow-20261110T062938/`, under `.data/model-probes/`. These contain three tiny synthetic producer/consumer examples, not repository retrieval. Text expressions are interpreted by the model; no source code is executed. Cases preserve an empty submit payload, preserve an existing valid forwarding branch, or explicitly add submit forwarding. Expected values and consumer decisions are scored locally, outside the model input.

Both trials returned parseable JSON for 3/3 cases, but 0/3 exact value-and-decision pairs matched. This needs careful interpretation: the first case correctly rejected the empty payload but encoded the value as two literal quote characters instead of the empty string. Other cases contain wrong or internally inconsistent values/acceptance decisions. This does not mean every individual reasoning statement was wrong, nor does it quantify full-case mapping quality.

The first trial used baseline sampling, which overrides temperature but inherits the model's presence_penalty=1.5. The second explicitly set temperature=0, presence_penalty=0, repeat_penalty=1, top_p=0.95, top_k=20, seed=42, context=8192, output=512, thinking=false. Exact options and identity are saved in its manifest. The second trial still missed all exact pairs (37.05s cold, then 1.45s/1.55s). Earlier Reviewer controls used the coding profile with explicit penalties already; the inherited baseline penalty does not explain those failures. Short structured output and these small prompts limit inference; no production settings were changed from this experiment.

**Current boundary:** source evidence and a valid response format help auditing, but neither guarantees technical correctness. The Reviewer remains unqualified for autonomous sign-off. Keep the three roles, preserve NEEDS_REVIEW and explicit unresolved items, and evaluate another complete draft against source before claiming an accurate plan. No fourth agent, cloud fallback, runtime source edits, new model installation, or application startup occurred during these diagnostics.

Closure checks after these diagnostics: `go test ./...`, `go vet ./...`, labdiagnostics control-definition regression/replay, `npm run typecheck`, `npm run check:encoding`, gofmt and `git diff --check` passed. Offline replay passed as an execution check while correctly reporting model qualification=false. No browser or live application end-to-end run was performed; those checks do not establish a correct AI plan.


## Follow-up: JSON output, concrete edits, and file-scoped reasoning

The native JSON-mode value-flow diagnostic (`flow-20261110T063745`) produced 3/3 parseable responses and 2/3 exact value/decision pairs. The explicit forwarding edit remained misunderstood. JSON mode also added a short field instruction, so this is not a format-only causal experiment. It does not measure repository retrieval.

The v22 Reviewer trial (`roles-20261110T063950`) used JSON mode with an explicit schema prompt and unchanged host validation: accepted the reasonable proposal, rejected the nil-type defect with a correct rationale, and accepted the payload defect. Result 2/3. The opt-in v23 trial additionally omitted Planner summaries and self-reported behavior from CHANGE model input, retaining actual source, concrete edits, acceptance checks and full process graphs. NO_ACTION retains its current-behavior claim for verification; saved/exported plans are not mutated.

Matched v23 non-thinking first-three controls have identical request hashes across 35B, 9B and Coder. Artifacts: `roles-20261110T064333`, `roles-20261110T064548`, `roles-20261110T064822`. Expected verdicts: 35B 2/3, 9B 1/3, Coder 1/3. The 35B nil rejection still has an unreliable rationale. Two additional negative controls explicitly tighten unrelated actions or drop existing forwarding branches. 9B rejected the global guard with an unreliable rationale but accepted lost forwarding (2/5 overall); Coder accepted both (1/5 overall). Do not compare those five-control percentages directly with the older three-control suite.

### File-scoped reasoning result

Artifacts `roles-20261110T065118` (two frontend controls) and `roles-20261110T065520` (five disjoint controls). Verified identical model identity: 35B Q4, v23, coding, thinking on, output 6144, context 32768, JSON mode, one candidate file per request. Full before/after graphs remain available. Expected labels were never sent to the model. Private derived suite, source-result SHA256 manifest, definitions and offline gate: `.data/model-probes/reviewer-json-v23-controls/`.

| Control | Expected | Observed |
|---|---|---|
| Reasonable frontend plan | Accept | Accepted |
| Reasonable backend plan | Accept | Accepted |
| Reasonable integration-test plan | Accept | Accepted |
| Invalid Go string nil guard | Reject | Rejected; identified the type error |
| Validation but unchanged empty producer payload | Reject | Rejected; identified the unchanged conditional |
| Global reason guard affecting unrelated actions | Reject | Incorrectly accepted |
| Submit-only forwarding drops existing action branches | Reject | Incomplete output at token limit |

Offline gate: **valid 6/7, correct 5/7, passed=false**. Good proposals accepted 3/3; planted defects rejected 2/4, with one of the other two incomplete. The payload rejection's primary requirement reason is correct, but its additional assertion about absent failure-test specifications is unsupported. This is partial improvement on development controls, not independent semantic proof or whole-case Precision/Recall/MAP/MRR. The two source runs are combined only after verifying identical identity and unique control IDs; raw runs are not rewritten.

First/cold calls took 112.76s and 111.05s, above the production HTTP timeout. Warm calls ranged from 55.40s to 91.37s; the last call exhausted output. Production timeouts/output defaults were not increased and this policy is not enabled by default. It still needs a complete-case trial under application runtime limits, particularly final cross-file reasoning and non-fabricated review findings. The later fixed-target trial below is separate from this control suite.

### Program changes and boundaries

- Added fixed per-role thinking overrides (`--analyst-thinking`, `--planner-thinking`, `--reviewer-thinking`: inherit/on/off), preserving explicit false and model identity. A thinking-only override now creates a distinct adapter; no automatic fallback or model switching is introduced.
- Added opt-in `--agent-review-json`. v23 uses an explicit schema in the prompt, native JSON output and identical strict host validation. Combined with Reviewer thinking, checkpointed file reviews run first, followed by the existing compact cross-file check. All source requirements remain supplied; no fourth agent or hidden inference is added. Default flags retain the ordinary review policy.
- Fixed loss of detailed source Review findings during partitioned final review. Source evidence/comparison receipts survive both failed file review and successful final contract review, alongside proposal-only contract findings.
- Added regression coverage for strict JSON validation, immutable draft projection/NO_ACTION, independent role thinking, fixed routing, file-review budget/resume, cross-file checks and retained source findings. Source snapshots and user root reports were not modified. No application server/browser was started.

The final runtime composition is identified as **v24**, distinct from the v23
direct source-control captures: it preserves behavior requirements on source-free
contract inputs and explicitly selects checkpointed file reviews for the opt-in
reasoning policy. The source-backed projection/prompt used by those controls is
unchanged; no complete-case live validation of v24 is claimed. Default v20 and
experimental comparison-only v21 identities remain unchanged. Fresh AI sessions
are required for the new policy. The stored analysis/BPMN pair can be reused.

Runtime template inspection found built-in qwen3.5 renderers on the local Qwen3.5
models and glm-4.7 on GLM. The short `{{ .Prompt }}` template alone is therefore
not evidence that system messages were omitted; no lab templates were changed.
The official [Ollama Modelfile reference](https://docs.ollama.com/modelfile)
documents template/system variables. This check did not establish a renderer bug.


## v24 fixed-target Planner / Reviewer trial and retry repair

Artifact `.data/model-probes/draft-review-20261110T071105/`. The frozen Analyst decisions/source for the same three implementation targets were reused; no fresh Analyst, retrieval evaluation or full repository scan occurred. No provisional reference labels were passed to the model. The actual AgentEngine, strict Ollama adapters, fixed RoleRouter, partitioned source reviews, checkpoint store and production 85s HTTP client limit were used. Model 35B, coding, context 32768, output 6144; Planner thinking off, Reviewer thinking on; v24 JSON review. The trial had a six-minute outer bound and did not start FE/BE.

Planner returned a structurally valid three-file draft in 64.82s (36.57s weight loading). Inspection still finds empty-string-only validation instead of whitespace handling and incomplete updates to existing successful submit test calls. The frontend draft explicitly adds submit forwarding, but has redundant-input wording and validation detail gaps. No draft is declared technically correct from format alone.

Both Reviewer calls exhausted the 6144 output budget before final JSON: 78.10s and 75.69s, with resident weights. Session PAUSED, 5 orchestration turns, 3 inference calls, 0 semantic revisions, **no final export**. Total observed trial time 219.22s. This is an incomplete fixed-target trial, not a complete-case accuracy result or a successful pipeline benchmark.

The trial exposed a program defect: the root output-repair feedback was not forwarded into partitioned review calls, allowing an identical failed prompt to repeat. Fixed by forwarding root feedback into every source/contract review and binding its receipt identity to that feedback; changed feedback cannot reuse a stale approval. Parent session counters still charge all attempts. Added tests demonstrate that a failed source review receives the actual repair instruction, that identical feedback reuses its successful receipt, and that changed feedback requires another review. Per-file source findings remain preserved alongside the final contract findings.

These fixes do not increase output/timeout defaults or guarantee that the model will finish its reasoning. A new narrowly scoped retry diagnostic follows, preserving the failed session and counters rather than resetting them.


### Narrow retry diagnostic after forwarding repair

Artifact `.data/model-probes/draft-review-20261110T071931/`. New direct diagnostic,
not a reset/resume of the failed session: one backend proposal from that session,
its frozen matching source, complete process graphs and actual failure feedback.
The file-review instruction was shortened as well, so completion cannot be
attributed to feedback alone. Same 35B/coding/thinking-on/6144 v24 configuration.
An explicit model-load request with no prompt/source restored the warm-weight
condition of the original retry: 37.16s loading, followed by a 64.34s review call
under the ordinary 85s HTTP client limit.

The response passed strict host validation and the model approved the file.
This is **not** semantic proof: empty-string-only validation still omits whitespace
handling, and the finding largely restates that narrow check. No other file,
whole-repository scan, complete final plan or new accuracy metric was produced by
this diagnostic. Original failed checkpoints/counters and user reports were kept.

Closing verification after the retry fix: `go test ./...`, `go vet ./...`,
labdiagnostics compilation/control-definition checks, frontend type/encoding
checks, gofmt and `git diff --check` passed. Review-policy integration tests cover
resume without repeating completed file work, charged budgets, preservation of
cross-file checks and detailed source findings, feedback forwarding, reuse under
identical feedback and invalidation under changed feedback. No FE/BE/browser was
started, no model was imported/downloaded, and no case-study source was edited.

**Remaining:** complete the actual Planner/Reviewer flow under production budgets
and measure technical correctness on an explicit requirement set. The observed
control gains and one valid retry do not justify automatic sign-off or a claim
that arbitrary BPMN changes are solved. Default review settings remain unchanged;
new JSON/reasoning policy is opt-in and requires a fresh AI session.


## Continuation: source-grounded acceptance and coordinated file review

The three case-study source files still match the SHA256 values in the frozen
`current-staff-v2/add-task/source-review.json` when checked on 10 November. Source
was read only. The checks below are an engineering audit prepared after seeing
model outputs, not independent ground truth, training data or model input.

Core acceptance for the reason-before-submit case:

1. Reuse the frontend reason state/input and send its actual value for submit;
   retain existing forwarding for other actions and existing role/state controls.
2. Validate the reason in the backend submit branch before state mutation, keeping
   transaction/lock/version/DRAFT checks and unrelated branches. Do not impose a
   new reason requirement on approve through an unconditional/global guard.
3. Update impacted successful inventory submit test calls that currently supply
   empty text, then add empty-input rejection and unchanged-state checks. Existing
   source includes more submit calls than the two excerpts, so a plan should
   explicitly search/update all affected success fixtures, not claim a complete
   line list from limited excerpts.
4. Reuse existing request field, backend parameter and persistence. No new schema
   or endpoint is justified by this case. New local errors are proposals; an HTTP
   error mapping must be confirmed rather than invented.

The stored requirement says non-empty; BPMN's label says enter and validate.
Rejecting whitespace-only input is sensible hardening, but its exact normalization
policy is not specified in that label. Its omission should be recorded as an open
requirement/hardening gap rather than silently treated as an independently proven
benchmark failure. This does not excuse missed submit forwarding, broken existing
success tests, or changes to unrelated actions.

### Coordinated file context

Per-file review previously supplied only that file's proposal/source. This could
make a test proposal appear responsible for implementing a backend guard, rather
than testing the backend change proposed elsewhere. Added compact
`related_proposals` containing other targets' intended behavior, edits and
assumptions. They contain no citable evidence IDs or source claims. Current source
remains the only authority for observed behavior. Related proposals are included
in receipt bindings so changed companion contracts cannot reuse stale approval.

Local target behavior is retained as a claimed obligation, not implementation
proof; Planner self-summary and claimed current behavior remain omitted from the
experimental source-backed Reviewer input. Partitioned Planner calls can see
already produced companion proposals as unverified future contracts. No extra
agent, source write, ground-truth input, hidden model call or silent truncation is
introduced. Byte, turn, context and evidence guards still apply.


## Next-session continuation and stopping checkpoint

User requested closing work to leave. No model call remains running.

- `roles-20261110T073131`: v24 per-file 35B non-thinking controls, 4/7 expected verdicts; several rejections incorrectly demanded new behavior already exist in old source. Not qualified.
- `roles-20261110T073551`: v25 shorter future-proposal instruction, 35B non-thinking, 4/7 expected verdicts with unreliable rationales. Instruction changes alone did not solve the issue. Generic examples are development guidance, not case labels; this is not a blind test.
- `system-canary-20261110T074039`: two harmless, source-free system-instruction checks both returned the expected system token over a conflicting user token. This narrow diagnostic found no system-message delivery failure; it is not general prompt-injection robustness.
- `roles-20261110T074219`: GLM v25 non-thinking, reasonable backend plus two unrelated-action controls; 1/3 expected verdicts. `roles-20261110T074613`: identical controls with thinking on, 2/3; caught lost forwarding, still accepted a global guard affecting unrelated actions. No automatic model change was made.
- `roles-20261110T074933`: three independent Planner calls, one per supplied target. All produced structurally valid proposals, but technical inspection still found misread argument roles, incomplete test coverage and an inadequate frontend assessment. Per-file formatting does not establish a better plan.
- `roles-20261110T075605`: v26 coordinated context, non-thinking 35B accepted all seven controls (3/7 expected verdicts). All three reasonable proposals were accepted but all four planted defects were missed. This is not a quality improvement; non-thinking review is not qualified.

### v26 actual fixed-target trial

Artifact `draft-review-20261110T075833`: current context coordination, frozen
Analyst decisions, real Planner/Reviewer calls, 35B coding, Planner thinking off,
Reviewer on, output 6144 and production client limits. All three case-source
hashes still match the frozen source review. Planner produced the same weak
three-file draft (64.51s). Backend and test file reviews returned valid approvals
(63.96s, 32.02s), but source inspection still finds the test proposal incorrectly
claims existing empty-reason success calls can remain unchanged. Approval is not
proof. Frontend review exhausted output (80.12s); the repair then hit a program
checkpoint-ID length defect. PAUSED, 6 turns, 0 revisions, no final export.

The newly combined related-contract and feedback suffix exceeded the store's
64-character session ID limit. Fixed by hashing the entire derived name only
when it exceeds the limit, preserving existing safe receipt names and strict
filesystem validation. A regression test uses the actual filesystem checkpoint
store, a maximum-length root ID, related proposals, failed review, repair and
restart restoration. Targeted Go tests pass. This persistence fix has not yet
been re-run against the lab model; no complete plan is claimed.

The remaining work is the one-case Planner/Reviewer completion and source-level
acceptance review. No new scenario is needed. New runtime context and checkpoint
handling are implemented and tested; model quality remains unresolved. User root
reports and Simple-ERP source were preserved. No commit/push or FE/BE startup.


## Successful live recovery before progress presentation

Artifact `.data/model-probes/draft-review-20261110T111937/`. A checksum-verified
copy of the paused `draft-review-20261110T075833` session and child receipts was
resumed under the same source/model/budget binding. The original remains intact.
The model was explicitly loaded with no prompt/source to restore the resident
weight condition. No Planner or Analyst inference was repeated.

Result: **COMPLETE**, consumed turns preserved from **6 to 11**, 0 semantic
revisions, **4 actual Reviewer calls**, canonical JSON and Markdown exported for
one changed task / three supplied implementation targets. Observed run
285.31s, including 36.05s explicit loading.
This verifies the actual lab path through retry, bounded receipt identities,
file reviews, final cross-file check and export. It is not a fresh Analyst or
full repository scan, nor a mapping accuracy measurement.

Files: `plan.md`, `plan.json`, `summary.json`, `inferences.json`, `sessions/`.
The separate `source-audit.md` records post-output assistant inspection. The test
proposal still incorrectly treats existing empty-reason successful calls as
already valid. Those fixtures require correction before implementation. No
model output was rewritten to hide this issue; COMPLETE denotes orchestration
completion and does not establish technical correctness.

The sanitized measured outcome is also shown in Research. No ERP source was
edited/executed and no application server/browser was started.
