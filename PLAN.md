# BPMN Impact Mapper — Phase 1 Plan

Status: **DONE — Phase 4.0 persisted analysis history verified**

## 2026-10-28 focused retrieval and canvas follow-up

- [DONE] Changed gateway conditions now carry the changed branch text, process name, and immediate neighbors into lexical/semantic queries. Synthetic positive and unrelated negative controls pass; retrieval metadata versions changed. This is query construction, not proof of source dependency.
- [DONE] One read-only Simple-ERP gateway pilot rerun (`outputs/gateway-check/`, ignored) completed: primary `MODIFY_GATEWAY_CONDITION` changed from zero displayed candidates in the older report to ten, with the provisional reference file at rank 4. This is a diagnostic, not an independently measured accuracy gain; other new top candidates still need review.
- [DONE] Canvas Before uses a navigation-enabled read-only viewer with independent zoom/fit controls; a Playwright test verifies zoom and reset without editing the baseline.
- [TODO] Re-run the three frozen staff scenarios after the query change and compare against independently adjudicated labels. Previous reports are not comparable as same-version runs; no precision/recall improvement is claimed yet.
- [DONE] Production Hybrid now abstains from naming a function when its source span has no unique direct changed-ID match or unique two-word label/name match. File ranking, RRF score, and `VERIFY` action stay unchanged; focused synthetic controls cover direct ID, prefix collision, label match, tie, and unsupported function. Method metadata and analysis identity are versioned.
- [DONE] Diagnostic reruns under ignored `outputs/gateway-symbol-check/` and `outputs/pr-symbol-check/` retained the inventory file at rank 4 (RRF 0.9262) and purchasing file at rank 8 (RRF 0.8434), while replacing unsupported `Store.PostAdjustment`/`Store.PostReceipt` assertions with file-level candidates. Input fingerprints differ from historical user reports because the Simple-ERP working tree changed; this is not a controlled accuracy comparison.
- [DONE] The third diagnostic `ADD_TASK` Hybrid rerun initially stopped twice in the local Python embedding worker with Windows access-violation exit `0xc0000005`, then succeeded twice without code/input changes. The two successful canonical JSON files under ignored `outputs/reason-symbol-check/` and `outputs/reason-symbol-check-repeat/` have the same analysis ID and SHA-256. This shows an intermittent local runtime failure, not a deterministic BPMN-input failure. At the successful run, Windows reported roughly 1.8–1.9 GB remaining commit headroom and the Python worker used roughly 0.75 GB working set; those observations do not prove memory exhaustion as the cause. No silent BM25 fallback or ranking change was introduced.
- [TODO] Reproduce or profile the native worker crash under controlled memory pressure before changing batch size, retry policy, or error classification. On failure, keep an explicit error and do not label a fallback result as Hybrid.
- [TODO] Validate this conservative function-name policy against independent multilingual/held-out symbol labels. It reduces unsupported function assertions but can omit the correct function when a BPMN role diagram uses generated IDs or different-language labels.
Last updated: 2026-10-22

## Source-relation ablation probe (2026-10-22)

- [DONE] Add a separate, offline `probe-relations` CLI and pure `internal/relationprobe` component. It requires exact cited frontend HTTP-call-line evidence, follows a unique indexed route to a Go handler and at most one method/interface call, and shows an alternate existing-file order without changing canonical scores/actions or product Hybrid. Synthetic tests cover deterministic path selection and abstention when the source line is not cited or the target file was not retrieved.
- [DONE] Read-only Simple-ERP PDF-task probe found `purchase-request-panel.tsx:31` → `Handler.create` at `handler.go:22` → `Service.CreateRequest` call at `handler.go:96`. The score-free proposal moves existing handler/service files from ranks 14/15 to 3/4 and puts four provisional core files in Top-5. This is **one exposed development case**, not blind accuracy evidence or a validated production ranking improvement. The command explicitly warns that a saved report and current repository snapshot are not cryptographically rebound.
- [DONE] `go test -p 1 ./...`, `go vet -p 1 ./...`, and `git diff --check` pass. The actual Simple-ERP `probe-relations` CLI returns the two cited paths and the score-free Top-5 proposal in about two seconds without re-running embeddings. Keep this as an opt-in CLI diagnostic, not a fifth public mapping method. No BPMN or Simple-ERP changes.
- [TODO] Test relation-aware ordering on multiple frozen scenarios with independent labels and negative controls before considering a versioned production reranker. Record false positives from shared routes, ambiguous links, and unsupported languages; abstain rather than infer missing edges.

## One-scenario method diagnosis and file-level Top-K (2026-10-22)

- [DONE] Run BM25, Semantic, Fusion, and Hybrid on the same Simple-ERP required-PDF task insertion with identical input fingerprint `afe268d3154d34a28084b9bb01434307`. The provisional four-file reference is a development diagnostic, not blind ground truth. At the old artifact-level output, BM25 found one core file in its first 20 artifacts, Semantic three, Fusion two, and Hybrid four; at Top-5, BM25 found zero, Semantic one, Fusion two, and Hybrid two. Results cannot establish general accuracy from one scenario.
- [DONE] Found a comparison defect independent of those labels: BM25/Semantic/Fusion public Top-K consumed slots with multiple functions from the same file, while Hybrid already selected one best function per file. Make all four public reports file-diverse before Top-K, while retaining complete artifact rankings *inside* RRF. Added standalone adapter tests; engine identity is `engine-3.5d-file-level-topk` so prior reports are not silently comparable.
- [DONE] `go test -p 1 ./...`, `go vet -p 1 ./...`, and `git diff --check` pass. Same-input read-only Simple-ERP Hybrid smoke with engine 3.5d has the identical Top-20 file/symbol order and RRF values as 3.5c (20 distinct files, zero warnings, matching input fingerprint). Standalone BM25 and Fusion now each emit 20 distinct files at Top-20. Fusion still places the provisional Postgres store and frontend panel at ranks 3 and 4 but not the handler/service within Top-20; file diversity alone is not an accuracy fix. Reports are ignored under `.data/`; no Simple-ERP file was changed. Do not calibrate lexical/semantic weights from the one provisional reference.
- [TODO] Evaluate source-relation evidence (frontend route → handler → service) as a separately versioned ablation on multiple development scenarios. The current index already extracts conservative links, but promoting linked files from one example could add false positives elsewhere. Hidden scenarios remain necessary for final Recall/Precision claims.

## Semantic passage coverage and ranking boundary (2026-10-22)

- [DONE] Sample bounded head, tail, and middle source passages for Semantic retrieval instead of head/tail only; select a query-overlapping evidence line within the winning passage when available. Version the engine as `engine-3.5c-semantic-middle-passages` and record the passage strategy in method metadata. Unit tests cover a middle-only match and accurate bounded excerpt location.
- [DONE] Keep production Hybrid as BPMN-context BM25 + Semantic RRF. Task IDs, neighbouring IDs, path/function names, and code clues remain evidence, not a hidden third weighted score. For `ADD_TASK`/`ADD_GATEWAY`, missing new IDs in old source are expected. Consider a separately versioned trace/context reranker only after a blinded multi-scenario ablation; action classification remains separate from ranking.
- [DONE] `go test -p 1 ./...`, `go vet -p 1 ./...`, and `git diff --check` pass. A local synthetic Semantic CLI smoke wrote JSON/Markdown under ignored `.data/semantic-passage-smoke`: 12 BPMN changes, zero warnings, and method metadata `passage_version=head-middle-tail-v1`. The command took roughly 15 seconds on this machine; that is an operational smoke, not a controlled latency benchmark. More passages can increase embedding cost. Retrieval accuracy still requires independent ground truth.
- [DONE] Read-only Simple-ERP one-scenario diagnostic (required supporting PDF task) with engine 3.5c: four provisional core file ranks are 1 (Postgres store), 2 (frontend purchase-request panel), 14 (HTTP handler file; displayed function `Handler.Register`), and 15 (application service). Previous 3.5b ranks were 1, 2, 12, and 11 respectively. Provisional file Recall@5 stays 2/4 and Recall@20 stays 4/4; this is **not** an independent accuracy metric because the reference was drafted after mapper inspection. The middle-passage change did not improve this case. Documents, integration tests, historical migrations, and a finance store still outrank two core files. BM25 ranks the HTTP handler function 174 and service 82; Semantic ranks them 19 and 42, so RRF cannot by itself infer the missing create-operation dependency. Do not fit weights or declare a gain from this one case.
- [TODO] Diagnose generalizable retrieval improvements with a separate development scenario before changing production weights. Candidate work: query-role ablation (changed label versus BPMN neighbours), function-level source-relation propagation with explicit path evidence, and displaying implementation versus documentation/test candidates separately. Keep the frozen one-scenario draft out of inference and reserve independent scenarios for final evaluation.

## Production Hybrid simplification (2026-10-22)

- [DONE] Replace the production Hybrid's six-weight code-evidence ranking with BPMN-context BM25 plus Semantic, combined by equal-position RRF. The plain BM25 research baseline uses only changed-element text; the production BM25 branch also uses BPMN neighbours. The old weighted adapter remains for historical regression tests only. Exact IDs, neighbouring IDs, function/path names, and code-pattern hints remain bounded explanatory evidence without separate score contributions. New element IDs do not penalize ranking; retrieved candidates remain `VERIFY` until an independently justified action classifier exists. Engine identity is `engine-3.5b-context-bm25-hybrid`.
- [DONE] Update the web score explanation, Markdown, README, and methodology to distinguish current reports from legacy saved reports. Added lexical-branch and composition-root tests. Go unit suite and local synthetic Hybrid smoke pass; the synthetic scenario had no semantic candidate over threshold, which is recorded rather than silently treated as agreement.
- [DONE] A fresh Simple-ERP L3 task-insertion diagnostic with context BM25 places the core existing files at file ranks 1, 2, 11, and 12 versus 1, 3, 11, and 21 previously. The selected HTTP symbol is `Handler.Register`, not the more relevant `Handler.create`; finance, migrations, and sales still appear. This is not adjudicated accuracy and does not establish superiority. Go `test -p 1 ./...`, `vet -p 1 ./...`, web typecheck/build, six focused Playwright explanation tests, and `git diff --check` pass. Full Playwright runs past the initial UI tests but local embedding capability checks hit `MemoryError` with Next+Chrome active; full E2E is not a passing gate on this machine yet.
- [TODO] Build a method-blind, adjudicated multi-scenario benchmark before claiming that simplified Hybrid improves accuracy. Report Recall@k, Precision@k, MRR, false positives, action accuracy, and latency by change type. Keep any weight calibration on development scenarios only.

## Named task context-anchor query correction (2026-10-22)

- [DONE] The uploaded required-PDF scenario exposed long BPMN documentation acting as generic search noise: only one of four core code files was in the displayed Top-5, while Sales/Finance ranked highly. This is a diagnostic case, not validated ground truth.
- [DONE] Build a language-agnostic bounded context anchor from a phrase shared by changed-task documentation and adjacent BPMN labels, with a shared-neighbor-token fallback. Named add/remove task queries now use the task label plus that anchor instead of the whole instruction paragraph; BM25 retains neighbor labels and structural hints use only the short anchor. No Simple-ERP identifier/path or retrieval weight was hardcoded.
- [DONE] Same-input local smoke moves purchasing frontend 14→3, handler 37→11, service absent→21, Sales store 4→absent from displayed Top-50; candidates 43→26. Finance store remains 14 and historical migration remains 4. Final analysis ID `d3c4a1fc5ad19dfdefed7bfda5d3805c` differs from the previous engine; method metadata records query version on both retrieval branches. `go test -p 1 ./...`, `go vet -p 1 ./...`, gofmt, and `git diff --check` pass. One candidate-tie warning remains. Independent ground truth is still pending; rank movement is not measured accuracy.

## L3 PR attachment scenario reference (2026-10-22)

- [DONE] Keep one isolated `ADD_TASK` scenario with three supporting plain-flow edits; no new gateway. Add a visible `Upload required supporting PDF` task label and explicit mandatory, bounded, atomically persisted PDF requirement in its BPMN documentation. Stable task ID and sequence flows are unchanged.
- [DONE] Record a provisional, source-inspected existing-file impact reference in `docs/ground-truth/pr-supporting-document-reference-draft.md`. The author has seen prior mapper output, so this is not blind ground truth and must not be used to claim accuracy.
- [TODO] Confirm the reference through a separate authorized implementation diff or independent reviewers; freeze the pre-change repository revision and only then export evaluator ground-truth JSON. Keep both draft and final labels outside inference.

## Named task semantic-query focus (2026-10-22)

- [DONE] Confirmed the three zero-candidate sequence-flow edits in the uploaded L3 PR report are `part_of` the added task. Their impacts are intentionally retrieved once under that task; independent flow changes still map independently. No duplicate retrieval was added.
- [DONE] Remove predecessor/successor labels from the semantic query for a named added/removed task so the embedding targets the changed responsibility. Preserve graph neighbors for topology-only and unlabeled changes, and in BM25/code-evidence retrieval. Unit tests, `go test -p 1 ./...`, `go vet -p 1 ./...`, gofmt, and `git diff --check` pass. Same-input read-only Simple-ERP smoke: purchasing frontend 5→4, purchasing service 10→7, HTTP handler 15→12, Sales store 21→27; finance store stays 8, migration `purpose` moves 4→5. Candidate count stays 35 with zero warnings. This is a diagnostic ranking comparison, not measured accuracy.
- [TODO] Do not claim improved precision from changed ranks without independent file-level ground truth. Semantic-only cross-module candidates remain visible as `VERIFY` pending adjudication.

## Structural evidence precision audit (2026-10-22)

- [DONE] Audit the latest L3 PR `ADD_TASK` report: unrelated Sales/Finance SQL earned `DATABASE_TABLE=1` from task words elsewhere on a line, even when its matched table was unrelated.
- [DONE] Require query overlap with the extracted SQL table identifier for `DATABASE_TABLE`; targeted positive/negative unit tests pass. No Simple-ERP term, path, or scoring-weight change was introduced.
- [DONE] Follow-up smoke found that a neighbour-task word still gave an unrelated `products` table full structural credit. Use only changed-element words for structural clues and scale SQL-table strength by identifier-token coverage; preserve neighbour words in lexical/BM25. Focused tests pass.
- [DONE] Follow-up smoke found Sales still earned `AUDIT_ACTION=1` from an unrelated `request` variable even though its quoted action ID was `CREATE_SALES_ORDER`. Restrict audit/route hints to their action ID/path and test unrelated arguments.
- [DONE] Follow-up smoke found `STATE_STATUS=0.7` from an unrelated `DocumentNo` field on the same line as `Status: "DRAFT"`. Limit state/test hints to their matched syntax span; add negative regression tests.
- [DONE] Four read-only Simple-ERP L3 PR smoke runs isolated the SQL, neighbour-query, audit, and status same-line false signals. Final local report under ignored `.data/structured-span-smoke-20261022` moves Sales `Store.CreateOrder` from rank 8 to 21 and removes its structural credit; purchasing frontend moves 6 to 5, service 13 to 10, HTTP handler 21 to 15. Candidate count falls 43 to 35 with no warnings; all remain `VERIFY`. These are diagnostic rank movements, not measured accuracy.
- [DONE] Final `go test -p 1 ./...`, `go vet -p 1 ./...`, `git diff --check`, and gofmt pass. Git emits only existing LF/CRLF conversion warnings. Simple-ERP was read-only; mapper's existing dirty worktree and unrelated edits were preserved. No commit or push.
- [TODO] Independent ground truth remains necessary before calibrating weights or claiming better Precision/Recall. User review should distinguish contextual files from necessary edits.

## Literal HTTP route links for new-task review (2026-10-22)

- [DONE] Add a generic, conservative index relation from a literal JS/TS `api`/`fetch` call (explicit HTTP method) to a unique Go route registration and statically resolved handler. Ambiguous or dynamic routes abstain; synthetic positive, ambiguity, and ordering tests pass.
- [DONE] Let an existing frontend lexical candidate activate only its matching call line or a route line with two distinct query terms. Linked Go handler and one direct Go service call remain `VERIFY`; existing candidates gain located evidence without a score boost. Threshold and false-route-line tests pass.
- [DONE] Full `go test -p 1 ./...` and `go vet -p 1 ./...` pass; two read-only Simple-ERP L3 PR CLI smoke runs completed. The second run shows `Service.CreateRequest` with `HTTP_ROUTE_HANDLER` and `HANDLER_CALLS` evidence, while the HTTP handler itself is not in displayed Top-20. This is a rank visibility limitation, not a failed route link. No Simple-ERP-specific rule was added.
- [DONE] User reran the L3 request-task scenario with a larger Top-K and shared `impact-report-ffb70870f4c15ed001c6949841f06e43.json`: frontend, service, and handler appeared with route evidence, alongside unrelated Sales/Finance candidates. That review triggered the later structural-evidence audit above. At this earlier stage analysis identity was `engine-3.4o-http-route-score`; local smoke artifacts remain ignored under `.data`.

## Comparison and scoring explanation correction (2026-10-22)

- [DONE] Keep the raw four BPMN changes in canonical JSON, but count only independent mapping units in Compare methods, Saved analyses unresolved summaries, and blind reviewer packs. Supporting flow edits are not three failed retrievals. Domain, filestore, and reviewer-pack tests cover this rule.
- [DONE] Explain the two separate Hybrid rankings in the web report: six weighted code-evidence signals (including BM25) versus Semantic cosine ranking, then equal-position RRF. Semantic is not a missing seventh weight. Comparison overlap is explicitly Jaccard set overlap, not accuracy.
- [DONE] Recompute `CANDIDATE_TIE` from displayed fused candidates after file diversity. Upstream code-evidence/semantic ties no longer masquerade as final RRF ties. Fusion regression tests pass.
- [DONE] Verification: `go test -p 1 ./...`, `go vet -p 1 ./...`, web `npm run typecheck`, production `npm run build`, eight focused Playwright report tests, and `git diff --check` pass. The first sandboxed Playwright attempt exited `spawn EPERM`; the same tests passed after local child-process execution was permitted. No Simple-ERP end-to-end rerun or ground-truth accuracy claim was made.
- [TODO] User reviews a fresh Simple-ERP comparison report. Old saved reports remain unchanged; analysis identity is `engine-3.4m-report-unit-tie` for new runs. Independent, adjudicated ground truth is still required for accuracy claims and weight calibration.

## Insert-task retrieval and report grouping (2026-10-22)

- [DONE] Use shared changed-task and neighbor terms to focus path/function evidence for newly inserted tasks without requiring a generated task ID in existing source. Focused retrieval tests and full Go suite passed.
- [DONE] Treat only an unlabelled one-to-two sequence-flow replacement as a supporting detail of `INSERT_TASK`; preserve every raw graph change in JSON while running retrieval once for the primary task. Diff, analyzer, and Markdown tests passed; an earlier L3 PR CLI probe showed one mapped task plus three raw supporting flows.
- [DONE] Select the best-ranked function per file for production Hybrid display Top-K, preserving original RRF scores; leave research methods' artifact-level ranking intact. Fusion unit test covers file selection and unchanged baseline behavior.
- [DONE] Code checks: `go test -p 1 ./...`, `go vet -p 1 ./...`, web `npm run typecheck`, and `git diff --check` passed. Next production build compiled but stopped at TypeScript worker spawn with `EPERM` in this environment; direct typecheck passed.
- [TODO] User will run the L3 PR scenario in the web app and share a new report. Check candidate relevance and omissions against independent truth before claiming accuracy; no new end-to-end result is asserted for the final file-diversity selection.

## Trace evidence and score explanation (2026-10-20)

- DONE: audit found that changed-element IDs alone contribute the direct 45% trace signal; neighbour IDs already contribute query terms but were not labelled as contextual evidence.
- DONE: expose exact neighbouring ID occurrences as `CONTEXT_TRACE_ID`, distinct from direct traces, without changing scores, acceptance thresholds, propagation seeds or actions. Version the annotation through method metadata; regression tests check unchanged scoring/action and ID collision rejection.
- DONE: separate combined ranking scores from individual search signals; show direct/context/unused-check states, scope weights to the code-evidence branch, and align Markdown terminology. Five explanation tests and the Markdown/JSON golden test pass.
- DONE: `go test -p 1 ./...`, `go vet -p 1 ./...`, TypeScript, production frontend build and whitespace checks pass. Go concurrency was limited after an initial Windows paging/memory failure.
- BLOCKED (environment): visual browser verification remains open. After correcting the new browser fixture's history response, system Chrome exited on launch and the locally installed Chromium revisions did not match Playwright's expected revision. No browser was downloaded. The focused `scoring-display.spec.ts` is ready for rerun.
- TODO: calibrate the 45% trace weight only with independent development/held-out cases; do not rescale positive signals to make absent-ID cases appear certain.

## Local semantic timeout follow-up (2026-10-20)

- DONE: measured a cold Hybrid CLI analysis of the Simple-ERP P2P gateway/task scenario at about 2 minutes 48 seconds; the former 2-minute API default was insufficient.
- DONE: raised API and CLI analysis defaults to 10 minutes and the Next.js rewrite proxy to 11 minutes; retained cancellation and finite resource bounds.
- DONE: tested a larger Python embedding batch/thread count, observed worker failure on this machine, and reverted the experiment. Ranking and embedding semantics were not changed.
- TODO: profile embedding throughput across multiple machines/repository sizes before changing batching or claiming a general performance improvement.

## Canvas and scoring audit (2026-10-14)

### Workspace navigation increment

- DONE: top navigation for Mapping, BPMN Canvas, Saved analyses, Guide; full-width inline editor and preserved mounted state across navigation.
- DONE: browser tests cover navigation, canvas-first upload, draft/undo preservation, saved-report navigation and responsive layout. All 10 serial Playwright scenarios and TypeScript checks pass.

- DONE: isolated bpmn-js editor; immutable baseline, explicit edited XML submission, undo/redo, reset, download. Two Playwright editor scenarios pass (edit/analyze/reset and add task/gateway with BPMN-DI export).
- DONE: audit zero-valued signals and RRF candidate truncation; preserve action evidence requirements. See `docs/SCORING-AUDIT.md` and full-pool regression in `internal/fusion/strategy_test.go`.
- DONE: make stable-ID trace matching boundary-aware and add conservative grammatical matching for path/symbol tokens, with collision regressions; fixed weights and action thresholds remain unchanged.
- DONE: expose per-change evaluation diagnostics in the web dashboard (expected files, ranked predictions, RR, AP, and unresolved state) so aggregate Recall/Precision results are actionable.
- DONE: document the post-inference ground-truth protocol, evidence hierarchy, new-task limitation, and human/tool responsibility boundary; expose the non-self-labeling rule in the web guide.
- DONE: implement the pre-LLM agentic planning core as a typed Go state machine: bounded evidence bundle, verifier, per-layer planner, cross-layer reviewer, hallucination/completeness validator, deterministic plan JSON/Markdown, and offline provider compatibility. No network or model call was added.
- DONE: expose a read-only `planning-context.json` preview from immutable reports and link it in the web report, making the future model payload inspectable before provider activation.
- DONE: separate optional reviewer/reconciliation/metrics controls into a dedicated Research workspace so the primary Mapping report remains focused and presentation-ready.
- DONE: add a restrained Planning workspace that visualizes the exact bounded pre-LLM payload, counts candidates/evidence, reports payload size, and exposes evidence IDs without pretending a provider is connected.
- DONE: bind bounded payloads to deterministic context IDs, require provider/model/prompt identity, enforce per-role timeouts, and cover the provider contract with failure-path tests.
- DONE: add an API/UI planning execution boundary plus immutable atomic plan JSON/Markdown storage; the default offline provider creates an explicit `DISABLED` record and never fabricates plan content.
- TODO: add one opt-in OpenAI-compatible provider, adapter-level transient retries, and provider-failure fallback after explicit LLM configuration.
- DONE: editor round trip and analysis, ranking invariance against display top-k, `go test ./...`, `go vet ./...`, TypeScript, production build and diff whitespace checks. Nine browser scenarios verified: eight passed in the full run; one passed targeted rerun after fixing its nested-summary selector and cold-start timeout.
- TODO: frozen independent ground truth for empirical weight calibration; do not inflate missing signals or claim calibrated probabilities.
- TODO: persistent drafts and report-to-canvas highlighting in a subsequent increment.
- BLOCKED: Next/source-map-js advisory upgrade. npm audit identifies fixed ranges,
  but the configured registry returned ETARGET for Next 16.4.0 and 16.3.8 on
  2026-10-14. Keep the tool local until an available compatible patch is tested.
Scope owner: this repository only

## 1. Problem statement and research objective

Software often follows BPMN as a specification without loading BPMN at runtime. A BPMN edit therefore does not directly identify which implementation artifacts require modification or verification. BPMN Impact Mapper is a third-party, mapping-only analysis tool that converts a semantic BPMN graph diff and a static repository index into ranked, evidence-bearing impact candidates.

The research objective is to establish an auditable automatic pipeline comparing BM25-only, Semantic-only, BM25 + Semantic rank fusion, and BPMN-Context-Aware Hybrid Retrieval before evidence-bounded LLM planning. Ranking is candidate retrieval, not causal proof. The tool must abstain when evidence is weak or ambiguous. The legacy configuration adapter is outside the primary product UI and target experiment because it requires manual mapping before inference.

## 2. Scope

Phase 1 includes:

- secure inventory of one local repository without executing it;
- bounded BPMN XML parsing into a typed adjacency graph;
- stable-ID-first graph differencing and explicit ambiguity handling;
- generic textual indexing for supported text files;
- native Go symbol extraction with `go/parser`;
- deterministic BM25 and evidence-aware hybrid candidate ranking;
- canonical deterministic JSON and Markdown rendered from the same object;
- a CLI and a small `net/http`-compatible application boundary;
- a provider interface plus an offline `NoLLMProvider` only;
- synthetic fixtures, unit tests, golden tests, and methodology documentation.

### Explicit non-goals

- no source modification or code generation;
- no workflow execution or BPMN runtime engine;
- no database, migration, deployment, authentication, multi-user service, container, queue, cache, or vector database;
- no ZIP/Git URL ingestion in Phase 1;
- no external API, external LLM, or network access;
- no frontend in Phase 1;
- no claim of framework/language generality beyond the adapters actually implemented;
- no claim that a candidate, score, or confidence proves a required code change.

## 3. Assumptions and terminology

- **Specification-driven application**: software whose developers follow BPMN, while runtime code does not load BPMN.
- **Change**: a typed, evidence-bearing difference between old and new BPMN graphs.
- **Candidate**: an indexed software artifact retrieved for a change.
- **Evidence**: a bounded, locatable observation supporting a candidate; never an invented path or symbol.
- **MODIFY**: strong direct trace evidence plus change semantics that imply an existing implementation likely changes.
- **VERIFY**: relevant artifact with evidence, but modification is not established.
- **UNRESOLVED**: insufficient/ambiguous evidence or a matching ambiguity; an explicit abstention.
- Stable BPMN IDs embedded in source are legitimate traceability evidence.
- Input revisions are supplied metadata in Phase 1; the analyzer does not invoke Git in analyzed repositories.

## 4. Architecture and data flow

```text
CLI / HTTP-compatible handler
        |
        v
application orchestrator ---- cancellation / limits / warnings
   |       |       |
   v       v       v
ingest   BPMN    source adapters
inventory parser  generic + native Go
   |       |       |
   +-------+-------+
           v
   graph diff -> query/evidence terms -> BM25 + exact signals
           |                              |
           +------------------------------+
                          v
               evidence aggregation
                          v
             action / confidence / abstain
                          v
                 canonical report object
                    |              |
                    v              v
             deterministic JSON  Markdown
```

Component boundaries:

- `domain`: dependency-free contracts, enums, graph/index/report models.
- `ingest`: local-path validation, walking, ignores, limits, binary/secret exclusion.
- `bpmn`: safe bounded XML parsing and typed adjacency construction.
- `diff`: deterministic element matching and change taxonomy.
- `index`: adapter selection, textual documents, native Go symbols.
- `retrieval`: tokenization, BM25, evidence aggregation, scoring/action rules.
- `llm`: future provider contract and offline no-op implementation.
- `report`: canonical JSON and Markdown encoders.
- `app`: orchestration and artifact writing; no analyzed-code execution.
- `api`: transport-only request validation around `app`.
- `cmd/bpmn-impact-mapper`: flags, exit codes, and cancellation.

## 5. Proposed repository structure

```text
cmd/bpmn-impact-mapper/main.go
internal/{api,app,bpmn,diff,domain,index,ingest,llm,report,retrieval}/
testdata/{bpmn,repository,golden}/
docs/{ARCHITECTURE,METHODOLOGY,SECURITY,EVALUATION}.md
go.mod
README.md
PLAN.md
```

## 6. Technology decisions and trade-offs

- Go version pinned in `go.mod`; standard library only. This reduces supply-chain and reproducibility risk, at the cost of simpler ranking/parser features.
- `encoding/xml` token streaming with explicit byte/depth/element limits. Go XML does not expand external entities; the parser still rejects directives and unexpected roots.
- adjacency maps plus sorted public projections. Maps make graph lookup cheap; all serialized/iterated outputs are sorted to remove map-order nondeterminism.
- `go/parser` for Go only. Other languages receive textual indexing and must not be labeled AST-supported.
- in-memory BM25 for Phase 1. It is auditable and sufficient for study-sized inputs, but not intended for very large repositories.
- deterministic SHA-256 analysis IDs from normalized logical inputs/configuration, never wall-clock time or absolute paths.

## 7. Domain model

Core objects are `Element`, `Flow`, `Graph`, `Change`, `Artifact`, `Evidence`, `ScoreBreakdown`, `Impact`, and `Report`. Paths stored in output are slash-normalized repository-relative paths. Symbols include kind, name, optional receiver, and bounded line ranges. Warnings are stable codes plus messages.

Required actions are exactly `ADD`, `MODIFY`, `REMOVE`, `VERIFY`, `NO_ACTION`, and `UNRESOLVED`.

## 8. BPMN graph representation

- Nodes are keyed by stable BPMN ID and retain type, label, actor/lane, documentation, data references, incoming IDs, and outgoing IDs.
- Flows are keyed by ID and retain source, target, label, and condition text.
- `Incoming[nodeID]` and `Outgoing[nodeID]` store deterministically sorted flow IDs.
- Supported flow-node types include tasks/events/gateways; unsupported semantic elements generate warnings rather than silent invented semantics.
- Lanes map `flowNodeRef` IDs to actor names.

## 9. Element matching and graph diff

Matching order is stable ID, then type, normalized label, lane/actor, incoming/outgoing signatures, graph neighborhood, and structural similarity. Phase 1 fully implements stable-ID matches and conservative unique structural fallback. Competing equal best fallback matches produce `UNRESOLVED_MATCH`; no forced pairing is allowed.

Taxonomy: `ADD_TASK`, `REMOVE_TASK`, `MODIFY_TASK_LABEL`, `MODIFY_ACTOR_ROLE`, `ADD_GATEWAY`, `REMOVE_GATEWAY`, `MODIFY_GATEWAY_CONDITION`, `MODIFY_GATEWAY_SEMANTICS`, `MODIFY_SEQUENCE_FLOW`, `ADD_DATA_REQUIREMENT`, `REMOVE_DATA_REQUIREMENT`, `ADD_STATE`, `MODIFY_TERMINAL_OUTCOME`, `UNRESOLVED_MATCH`.

Sequence flow comparison covers add/remove, endpoint rewiring, label changes, and condition changes. Gateway condition changes are attributed separately when outgoing conditions differ. Terminal outcome changes compare terminal node labels/types. Data requirements use BPMN data input/output associations and data object references where present.

## 10. Secure repository ingestion

- canonicalize and validate repository/file paths before reading;
- reject roots that are symlinks; skip symlink entries so traversal cannot escape the root;
- enforce configurable per-file, total-byte, and file-count limits;
- honor context cancellation and caller deadlines;
- deterministic lexical file ordering;
- detect binary content using NUL/control-byte sampling;
- exclude secrets (`.env*`, key/certificate/credential patterns), generated/binary files, and configured path patterns;
- default directory exclusions: `.git`, `node_modules`, `vendor`, `dist`, `build`, caches, coverage, and generated output;
- default leakage exclusions: process/traceability maps, checkpoints, release/audit/implementation plans, generated impact reports, and ground-truth artifacts;
- evidence excerpts are line-bounded and length-bounded;
- repository contents are read only and never executed.

ZIP and Git URL sources are extension points only and remain unimplemented.

## 11. Source-indexing adapters

The generic adapter indexes safe text files as one file artifact with path, language by extension, tokens, lines, and structural hints from paths/content. The native Go adapter additionally uses `go/parser` to emit package, type, function, and method symbols with line-bounded text. Adapter output is normalized and sorted. Framework-specific routing is not claimed; textual route strings can still become lexical evidence.

The repository model can label likely layers (`domain`, `application`, `adapter`, `transport`, `persistence`, `frontend`, `test`, `documentation`, `unknown`) through configurable, generic path/content heuristics. These are explainable heuristics, not architecture proof.

## 12. BM25 retrieval

Tokenization lowercases Unicode letters/digits, splits camel/snake/kebab boundaries, and removes no domain-specific terms. Exact stable IDs are matched separately as direct trace evidence; their fragmented pieces are not ordinary lexical query terms for labelled elements. BM25 uses audited constants `k1=1.2`, `b=0.75`, a deterministic document order, and the Robertson IDF variant `ln(1 + (N-df+0.5)/(df+0.5))`. Equal scores tie by relative path then symbol.

Queries prioritize changed labels, state/data terms, and neighboring labels; actors are included for role changes. Unlabelled elements fall back to ID fragments. Flow IDs and generic change-type words do not enter labelled lexical queries. Empty/no-signal queries abstain.

## 13. Evidence, scoring, confidence, and unresolved rules

Evidence types include `TRACE_ID`, `LABEL_TOKEN`, `DOMAIN_TERM`, `PATH`, `SYMBOL`, `API_ROUTE`, `STATE_STATUS`, `DATABASE_TABLE`, `AUDIT_ACTION`, `TEST_NAME`, `BM25`, and `STRUCTURAL`.

Default code-evidence weights (fixed for this experimental baseline):

| Signal | Weight |
|---|---:|
| exact stable trace ID | 0.45 |
| symbol similarity | 0.15 |
| path/module similarity | 0.10 |
| domain/label overlap | 0.10 |
| API/state/table/audit/test structured hints | 0.10 |
| normalized BM25 | 0.10 |

Each component is clipped to `[0,1]`; the code-evidence score is the weighted sum with no missing-signal renormalization. BM25 normalization is `score / maxScore` within a query. Semantic retrieval is a separate ranking, and RRF combines positions rather than these numeric scores. Neither the code-evidence score nor the final ranking score is a calibrated probability.

Rules:

- candidate threshold: `0.20`; lower candidates are omitted;
- high threshold: `0.70` with exact trace evidence permits semantic action mapping to `MODIFY`, `ADD`, or `REMOVE`;
- otherwise relevant candidates are `VERIFY`;
- a top-score tie within `0.01` is retained and warned;
- no candidate above threshold creates an unresolved impact with the query/change reason;
- ambiguous graph match always creates `UNRESOLVED_MATCH` and no confident change action;
- every candidate must contain at least one concrete evidence item; BM25-only evidence is allowed but can yield only `VERIFY`.

## 14. Canonical JSON and Markdown

Canonical JSON is the authoritative report. Struct fields and sorted slices provide stable ordering; indentation is fixed; volatile timestamps and absolute paths are absent. Markdown is a pure render of the in-memory report object that is also serialized to JSON. Tests decode JSON back into the report and verify Markdown equality, preventing a second source of truth.

## 15. CLI and API boundary

CLI:

```text
bpmn-impact-mapper analyze --repo PATH --old-bpmn FILE --new-bpmn FILE --output DIR
```

Optional flags cover revision/name, ignore patterns, limits, top-k, and timeout. It writes `impact-report.json` and `impact-report.md` atomically within the requested output directory.

The HTTP-compatible boundary accepts typed JSON paths/config and returns the canonical report, but no long-running server, upload, authentication, or multi-user persistence is introduced in Phase 1.

## 16. LLM boundary

`Provider` accepts only filtered change/candidate/evidence objects and returns evidence-linked verification. Phase 1 implements only `NoLLMProvider`, which performs no network call and returns an explicit disabled result. Future provider names are reserved conceptually: Groq, OpenAI-compatible, and local-lab. Providers may enrich/rerank/explain but may not invent files/symbols, override absence of evidence, receive the entire repository, or become mandatory.

## 17. Leakage policy

Separate artifacts:

- **method input**: old/new BPMN, allowed repository files, method configuration;
- **baseline configuration**: frozen ignore/scoring/adapter settings created before evaluation;
- **ground truth**: independently reviewed expected impacts, hidden during inference;
- **excluded corpus**: process mappings, traceability coverage, checkpoints, release/audit/implementation plans, generated reports, ground truth, and implementation diffs/branches;
- **evaluation artifacts**: predictions, metrics, latency, reviewer reconciliation.

Stable BPMN IDs already present in implementation are allowed. Ground truth and implementation diffs may be inspected only after inference. Configuration-driven maps must be frozen and their preparation cost reported so they do not become disguised ground truth.

## 18. Security threat model

Threats include path traversal, symlink escape, oversized files/repositories, binary/parser bombs, secret disclosure, malicious XML, cancellation neglect, report path leakage, and accidental execution. Controls are canonical containment checks, no-follow walking policy, strict byte/count/depth limits, secret/binary exclusions, XML directive rejection, context checks, bounded excerpts, relative report paths, deterministic output, no shelling into analyzed repositories, and no network.

Residual risks: text heuristics can index sensitive values in non-obviously named files; XML semantic complexity can exceed supported subsets; TOCTOU changes can occur while local files are read; Windows junction identification differs from Unix symlinks. These limitations must be documented and surfaced as warnings where observable.

## 19. Experiment design

Compare on the same frozen repository snapshot, BPMN baselines/mutations, configuration, output schema, and hidden ground truth:

1. configuration-driven mapping;
2. BM25/lexical retrieval;
3. hybrid retrieval;
4. hybrid plus LLM verification (future phase).

Measure Recall@k, Precision@k, MRR and/or MAP, file-level accuracy, module/layer accuracy, action classification, unresolved rate, false-positive rate, latency, and repeated-run reproducibility. Report evidence-preparation and human-review cost separately. Use scenario-change × artifact as the evaluation unit, include negative controls, stratify by change type/layer, and reconcile ground truth with two independent reviewers when feasible. Conclusions remain bounded to evaluated systems.

## 20. Fixture and ground-truth strategy

- Synthetic BPMN fixtures are authored for this repository and contain task/gateway/flow/actor/data/terminal mutations.
- Synthetic source fixtures use neutral order-processing terms and multiple layers/languages; they do not copy Simple-ERP or POC TA.
- Golden JSON/Markdown capture deterministic public output.
- Evaluation ground truth is stored outside the inference corpus and loaded only by evaluation tooling, not `analyze`.
- Simple-ERP is an external read-only study object, never copied into test fixtures.

## 21. Implementation phases

- **Phase 1 (complete):** deterministic offline ingestion, parse/diff/index/retrieve/report pipeline, CLI/API boundary, tests/docs.
- **Phase 2A (current):** isolated configuration/BM25/hybrid methods and post-inference experiment evaluation.
- **Phase 2B (not authorized):** ZIP/Git ingestion, richer language/framework adapters, frontend, large-scale experiment orchestration.
- **Phase 3 (not authorized):** optional evidence-constrained LLM providers and controlled evaluation.

## 22. Test strategy

- unit tests for path/ignore/binary/symlink/limits/cancellation;
- BPMN parser fixtures and malformed/oversized/directive cases;
- graph diff tests for add/remove/label/actor/gateway condition/flow/data/state/terminal/unresolved;
- Go adapter symbol tests and generic adapter tests;
- BM25 deterministic ranking and tie tests;
- candidate evidence/action/threshold/unresolved tests;
- JSON determinism, path privacy, Markdown-from-object, CLI integration, HTTP handler tests;
- golden end-to-end report tests;
- `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check`.

Tests may compile mapper test code but never build or execute the analyzed fixture repository.

## 23. Acceptance criteria

- [DONE] Go module and documented package boundaries exist (40 new source/doc/fixture files; no external module dependency).
- [DONE] Secure inventory honors ignores, limits, binary/secret exclusion, path containment, symlink policy, ordering, and cancellation (`internal/ingest` tests pass, including Windows symlink-root rejection).
- [DONE] BPMN parser is proven by original synthetic fixtures and same-file smoke analyses of both external Simple-ERP L2 BPMNs.
- [DONE] Graph diff detects add/remove/modify/sequence-flow changes and abstains on ambiguity (`internal/diff` tests pass).
- [DONE] Generic and native Go adapters produce deterministic evidence-bearing artifacts (`internal/index` tests pass).
- [DONE] BM25 ranking and tie behavior are deterministic (`internal/retrieval` tests pass).
- [DONE] Every emitted candidate contains evidence; confidence/action rules are explainable (unit and end-to-end assertions pass).
- [DONE] JSON is valid/deterministic and Markdown is rendered from the same report object (golden and round-trip tests pass; repeated CLI SHA-256 matched).
- [DONE] Reports contain no absolute local paths, API keys, or case-study-specific implementation rules (tests and implementation scans pass).
- [DONE] CLI produces both required artifacts (CLI integration test and synthetic smoke run pass).
- [DONE] `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check` pass on 2026-09-28.
- [DONE] Simple-ERP and POC TA before/after Git status and revisions are unchanged.

## 24. Risks and unresolved decisions

- [DONE] Accepted residual risk: conservative structural matching may reduce recall; false matching is considered worse than abstention.
- [DONE] Accepted residual risk: confidence is a transparent heuristic score, not empirically calibrated probability; calibration awaits labeled data.
- [DONE] Accepted residual risk: generic layer inference may misclassify unconventional repositories; output exposes `unknown` rather than inventing certainty.
- [DONE] Practical parser/index defaults smoke-tested read-only on both Simple-ERP L2 BPMNs after synthetic tests, with old=new and without excluded documents or implementation diffs as input.
- [BLOCKED] None.

## 25. Deviations

Controlled implementation deviation: Phase 1 keeps the six signal weights frozen in code rather than exposing weight overrides. Ignore rules, ingestion limits, top-k, thresholds, and tie delta remain configurable through typed boundaries. Free weight tuning is deferred until an evaluation dataset can constrain normalization and reduce experiment leakage; the implemented fixed formula exactly matches this plan and methodology documentation.

The HTTP boundary is implemented as a tested embeddable handler, not a server command, preserving the Phase 1 non-goal of a deployed multi-user service. Verification used a repository-local Go build cache because the sandbox denied the user-profile cache; this does not affect product behavior or dependencies.

## 26. Execution checklist

- [DONE] Inspect mapper workspace and applicable `AGENTS.md` files (workspace initially empty; no local file applied).
- [DONE] Capture Simple-ERP initial status read-only (`fix/ta-baseline-readiness`, `69724e6`, clean).
- [DONE] Capture POC TA initial status read-only (`main`, `1f09068`, pre-existing dirty files recorded externally in final verification log).
- [DONE] Audit Simple-ERP BPMN/source trace signals without using excluded documents for inference.
- [DONE] Audit POC TA concepts read-only; reject runtime/code-generation concerns from mapper scope.
- [DONE] Check local technology without downloads (Go 1.26.0, Git 2.46.0.windows.1).
- [DONE] Write and self-review this plan for required sections and scope consistency.
- [DONE] Implement Phase 1 packages, CLI/API boundary, fixtures, and documentation.
- [DONE] Run quality gates and end-to-end synthetic verification.
- [DONE] Confirm external repositories are unchanged.
- [DONE] Update evidence-backed statuses and issue final report.

## 27. Phase 2A plan — reproducible method comparison

Authorization: user instruction on 2026-09-28 to proceed with the next step. This tranche intentionally excludes frontend, remote ingestion, and external LLM providers.

### Objective

Make the research methods executable and comparable without weakening the leakage boundary. One inference command must select exactly one method, while a separate evaluation command may read hidden ground truth only after inference has produced a canonical report.

### Scope and boundaries

- `hybrid`: existing weighted exact/structural/BM25 evidence aggregation;
- `bm25`: lexical baseline ranked only by deterministic BM25, always classified `VERIFY`;
- `configuration`: frozen explicit change-selector to artifact rules, validated against the indexed repository and emitted as `CONFIG_MAPPING` evidence;
- `evaluate`: offline report-versus-ground-truth metrics with no repository/BPMN input and no feedback into predictions;
- canonical method metadata in reports and deterministic evaluation JSON/Markdown;
- CLI flags `analyze --method` and `--mapping-config`, plus `evaluate --report --ground-truth --output`;
- synthetic method configuration and hidden-ground-truth fixtures only.

Non-goals remain ZIP/Git ingestion, frontend, additional AST adapters, runtime tracing, configurable free-form scoring weights, external LLMs, database, and automatic source changes.

### Configuration mapping contract

Configuration is JSON with a declared version and sorted rules. Each rule selects a BPMN change using optional `change_type` and exact `element_id`, then declares repository-relative `file`, optional `symbol`, `module`, `layer`, action, and rationale. At least one selector is required. Paths must be clean relative slash paths; duplicates and unsupported actions are rejected. A rule is emitted only when its file and optional symbol exist in the current index. Missing targets become warnings, never invented candidates. Configuration files are method input and must be frozen before evaluation; they must not be generated from ground truth.

### BM25 baseline contract

BM25 uses the same tokenization, corpus, query, `k1`, `b`, top-k, and stable ties as hybrid retrieval, but scores are normalized BM25 only. Candidates require positive BM25 evidence and receive `VERIFY`; no exact-trace action inference is allowed. This keeps method differences explicit and avoids presenting lexical ranking as proof.

### Evaluation contract

Ground truth JSON keys each expectation by `change_type` plus element ID and contains expected file, optional symbol/module/layer, and action. Evaluation deduplicates predicted files per change before ranking metrics. It reports:

- Recall@1/@3/@5 and Precision@1/@3/@5;
- MRR and MAP;
- file-level micro precision/recall/F1;
- module, layer, and required-action accuracy over matched expected paths;
- unresolved rate and false-positive rate;
- evaluated change/expectation/prediction counts;
- report-declared latency when available (Phase 2A reports `0` because timing is deliberately excluded from deterministic canonical inference output);
- reproducibility hash supplied from canonical report bytes.

Missing predictions are not converted to `NO_ACTION`. Ground truth never enters `Analyzer`, retrieval packages, analysis IDs, or impact reports.

### Phase 2A acceptance criteria

- [DONE] Phase 1 remains green before Phase 2A changes.
- [DONE] Method enum and canonical report metadata implemented without silent fallback; invalid methods fail validation.
- [DONE] BM25-only retrieval produces deterministic evidence-bearing `VERIFY` candidates (`BM25Only` unit and CLI tests pass).
- [DONE] Configuration mapping validates selectors/paths/actions and never invents missing indexed targets (`configmapping` tests pass).
- [DONE] Evaluator computes hand-checkable ranking/classification/abstention metrics from hidden ground truth (`evaluation` tests pass).
- [DONE] JSON and Markdown evaluation artifacts are deterministic (repeated evaluation SHA-256 `7D98022E275DF567052B674469F7454A11A5B7ADF6CE92A128241652070A478D`).
- [DONE] CLI integration covers hybrid, BM25, configuration, and separate evaluation.
- [DONE] Leakage tests prove ground truth is excluded from inventory and rejected by the inference API; BPMN/config method inputs are dynamically excluded from retrieval.
- [DONE] Documentation explains clean boundaries, preparation-cost, metric definitions, and fairness constraints.
- [DONE] Modular Clean Architecture is enforced by strategy/resolver ports, a dedicated composition root, and import dependency-rule tests.
- [DONE] `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check` pass.
- [DONE] Simple-ERP and POC TA before/after status and revisions remain unchanged.

### Phase 2A verification note

The three-method synthetic smoke benchmark completed successfully. Its configuration fixture intentionally mirrors two synthetic expectations, so the resulting scores validate mechanics only and are not research findings. Real comparative claims remain blocked on a frozen external scenario set, independently reviewed hidden ground truth, and measured preparation cost.

## 28. Phase 2B plan — frozen benchmark and reviewer workflow

Authorization: user instruction on 2026-09-28 to continue after Phase 2A.

### Objective and Clean Architecture boundary

Automate fair repeated comparison without letting orchestration or reviewer data enter the inference core. `benchmark` is an application-level coordinator over the existing analyzer/evaluator ports. Manifest and artifact filesystem concerns remain in outward experiment adapters. Reviewer reconciliation is a pure use case over annotation objects. The `wiring` composition root remains the only package binding concrete mapping adapters.

### Scope

- add a method-independent `input_fingerprint` to canonical reports;
- validate a versioned benchmark manifest with unique scenario IDs and all three methods;
- hash/freeze old BPMN, new BPMN, method configuration, and hidden ground-truth files;
- run every selected method against the same scenario input for at least two repetitions;
- fail on input-fingerprint or repeated canonical-output drift;
- evaluate each completed prediction only after inference;
- aggregate method-level macro metrics, latency, scenario counts, and reproducibility status;
- output deterministic benchmark JSON/Markdown, with volatile timing explicitly separated from deterministic prediction hashes;
- reconcile two independent reviewer annotation files into agreed cases plus explicit disagreements; never auto-resolve conflicts;
- provide `benchmark` and `reconcile` CLI commands and synthetic fixtures/tests.

### Explicit non-goals

- no automatic creation of Simple-ERP mutations or implementation diffs;
- no fabricated case-study ground truth;
- no modification of Simple-ERP or POC TA;
- no frontend, ZIP/Git ingestion, external LLM, database, deployment, or source execution;
- no statistical significance claim from synthetic fixtures.

### Phase 2B acceptance criteria

- [DONE] Canonical reports expose a deterministic method-independent input fingerprint.
- [DONE] Manifest validation rejects duplicate scenarios/methods, missing frozen hashes, unsupported methods, and unsafe repetitions.
- [DONE] Orchestrator enforces identical input fingerprints and canonical output hashes across repetitions; drift has an explicit failing test.
- [DONE] Ground truth is loaded only after each prediction is complete, verified by an ordered port-event test.
- [DONE] Aggregate metrics are deterministic apart from separately labeled `benchmark-timing.json`; repeated report SHA-256 is `0C0A76CEBA9BAC0C3131AEF6886297AA7E505C1A441417DE6234BB2897CF9469`.
- [DONE] Two-reviewer reconciliation emits only exact agreements and preserves all conflicts (synthetic result: one agreement, one disagreement).
- [DONE] CLI integration covers benchmark and reconciliation workflows.
- [DONE] Clean Architecture dependency tests cover `benchmark` and `review` inner packages.
- [DONE] Synthetic benchmark is reproducible with benchmark ID `8dd3cf3109ddca57e9b3c23af07bbb23` and is explicitly documented as non-research evidence.
- [DONE] `gofmt`, `go test ./...`, `go vet ./...`, build, and `git diff --check` pass.
- [DONE] Simple-ERP and POC TA remain unchanged.

### Phase 2B verification note

Historical note: this tranche originally executed three methods. Phase 4.2
supersedes its manifest contract with the current four automatic methods. The
reviewer boundary remains unchanged: a real Simple-ERP benchmark still requires
independently adjudicated ground truth, which the tool does not fabricate.

## 29. Phase 2C plan — auditable case-study pilot preparation

Authorization: user instruction on 2026-09-28 to proceed with work that can be automated.

### Objective and boundary

Prepare reproducible Simple-ERP pilot inputs without writing to or executing the study repository. A generic, specification-driven BPMN mutator belongs to the outward experiment tooling; it is not part of inference and contains no Simple-ERP IDs or business rules. Tracked case-study specifications may name study elements, while generated BPMN and analysis reports remain ignored evaluation artifacts. The pilot demonstrates mechanics only until two independent reviewers adjudicate hidden ground truth.

### Scope

- add a versioned mutation specification with `set_attribute`, `set_child_text`, and `remove_element` operations;
- require a frozen baseline SHA-256 and exactly one application of every operation;
- validate generated BPMN with the production parser and write atomically;
- expose a `mutate` CLI command with explicit baseline, specification, and output paths;
- create six auditable case-study scenario specifications covering label, condition, sequence-flow label, actor, terminal outcome, and removal/rewiring changes;
- generate mutated BPMN only under ignored `outputs/`, then run all three mapping methods against the unchanged Simple-ERP repository;
- retain reports as reviewer evidence, not as final ground truth or research conclusions.

### Explicit non-goals

- no source-code mutation, execution, copy, or write inside Simple-ERP;
- no automatic inference of ground truth from predictions;
- no claim that a synthetic BPMN mutation equals a production requirement;
- no final evaluation metrics before independent reviewer agreement and adjudication;
- no frontend, external LLM, network dependency, database, or Phase 2 product expansion.

### Phase 2C acceptance criteria

- [DONE] Generic mutation engine and CLI are implemented without case-specific rules in Go packages.
- [DONE] Invalid hash, missing/duplicate targets, unsupported operations, malformed output, baseline overwrite, and existing-output overwrite fail closed in tests.
- [DONE] Six specifications are frozen with verified Simple-ERP baseline hashes and generate parser-valid, byte-preserving BPMN.
- [DONE] Every scenario runs with hybrid, BM25, and configuration methods; all 18 reports contain evidence for every emitted candidate.
- [DONE] Generated artifacts contain repository-relative report paths and no embedded absolute local paths.
- [DONE] Pilot documentation separates method input, baseline configuration, excluded corpus, reviewer data, and evaluation artifacts.
- [DONE] Full quality gates pass and Simple-ERP/POC TA status and revisions remain unchanged.

### Phase 2C verification note

Six case-study scenarios produced 18 canonical reports under ignored `outputs/pilots/reports-v2/`. A second complete run was byte-identical for all 18 JSON reports. Detected changes cover `MODIFY_TASK_LABEL`, `MODIFY_GATEWAY_CONDITION`, `MODIFY_SEQUENCE_FLOW`, `MODIFY_ACTOR_ROLE`, `MODIFY_TERMINAL_OUTCOME`, and `REMOVE_TASK`; flow condition/name changes intentionally also alter their gateway's outgoing-condition signature. Five hybrid results retain explicit tie warnings. These are useful uncertainty signals, not failures. No evaluation metric or final ground truth is reported because independent reviewer annotations and adjudication remain human research steps.

Coverage extension: [DONE] `append_xml` is a generic, parser-validated experiment mutation operation. Two additional read-only Simple-ERP P2P specifications generate `ADD_TASK` (`PUR-PR-002A`) and `ADD_GATEWAY` (`GW-P2P-BUDGET`) scenarios, each with the expected related sequence-flow changes. Generated BPMN and reports stay under ignored `outputs/pilots/`; no Simple-ERP file was changed. The remaining data/state and gateway-removal coverage is intentionally deferred until its focused scenario design is reviewed.

## 30. Phase 3 plan — local web presentation MVP

Authorization: user instruction on 2026-09-29 to build a modular Clean Architecture web application using Next.js and Go, explicitly prioritizing a presentable end-to-end mapping flow without authentication.

### Objective and scope

Provide a local, single-user web interface over the verified Go analysis engine. The Go backend remains authoritative for ingestion, BPMN parsing, graph diff, retrieval, scoring, canonical JSON, and Markdown. Next.js is a presentation adapter only and must not duplicate analysis rules.

The MVP accepts a server-local repository path, uploaded old/new BPMN files, method selection, optional configuration path, repository metadata, and top-k. It renders the completed report and provides canonical JSON/Markdown downloads.

### Architecture and boundaries

- `domain`, `mapping`, and existing analysis use cases remain framework-independent;
- an application-level analysis registry port stores and retrieves completed canonical reports;
- a filesystem adapter persists reports under a configurable local data directory using analysis IDs;
- `internal/api` remains the HTTP adapter and exposes versioned health, create-analysis, report-read, and download routes;
- a dedicated Go server command owns configuration, timeouts, graceful shutdown, and composition;
- `web/` is an independent Next.js TypeScript application using a same-origin development rewrite to the Go API;
- uploaded BPMN files use bounded temporary storage and are removed after each request;
- repository source is read only and is never executed.

### Explicit non-goals

- authentication, authorization, accounts, multi-user tenancy, database, Redis, queue, worker service, Docker Compose, and deployment;
- repository ZIP/Git ingestion, source modification, code generation, workflow execution, or external LLM;
- evaluation/ground-truth entry in the inference UI;
- frontend reimplementation of parser, diff, retrieval, scoring, or Markdown generation.

### Technology decision

Use Next.js, React, TypeScript, and CSS Modules with pinned dependencies. Use only Go standard library in the backend. Phase 3 persistence remains filesystem JSON/Markdown because the artifact is immutable and keyed by deterministic analysis ID; a database would add operational complexity without improving the presentation workflow. The registry is a port so a PostgreSQL adapter can be introduced later if concurrent users or searchable history become requirements.

### Phase 3 acceptance criteria

- [DONE] Versioned Go API supports health, multipart analysis, canonical report retrieval, and JSON/Markdown download.
- [DONE] Upload limits, temporary-file cleanup, request timeout, path validation, method/config validation, and safe error responses are tested.
- [DONE] Filesystem report store is deterministic, traversal-safe, atomic, idempotent, and covered by unit tests.
- [DONE] Go server command supports graceful shutdown and configurable address/data directory.
- [DONE] Next.js form runs an end-to-end mapping request without embedding analysis rules.
- [DONE] Results UI presents BPMN changes, impacts, evidence, score breakdown, confidence, unresolved impacts, and warnings.
- [DONE] Responsive presentation styling and empty/loading/error states are implemented.
- [DONE] Frontend type-check/build and backend gofmt/test/vet/build pass.
- [DONE] Simple-ERP and POC TA remain unchanged.

### Phase 3 verification note

The Go suite, vet, both binaries, dependency-boundary checks, and `git diff --check` pass. Next.js 16.3.5 and React 19.2.8 are pinned in the lockfile; TypeScript and the optimized production build pass. A production Next server returned the presentation page with HTTP 200, proxied backend health successfully, and submitted a real multipart BM25 analysis through `/backend` with HTTP 201 and 12 detected changes. A direct API smoke test also verified Markdown download, persisted JSON, and zero remaining temporary uploads. Headless Chrome visual inspection at 1440×1000 confirmed the responsive presentation shell renders correctly. No database or authentication was added because neither contributes to the scoped single-user presentation workflow.

## 31. Phase 3.1 plan — presentation hardening

Authorization: user instruction on 2026-09-29 to proceed with the recommended presentation-readiness improvements.

### Scope

- add an explicit one-click synthetic demo preset using neutral tracked fixtures in the presentation adapter;
- add concise action, confidence, and method interpretation guidance without overstating certainty;
- execute a real browser automation walkthrough from form submission through rendered evidence and report download;
- preserve manual input for arbitrary repositories and BPMN files;
- create a reviewed local Git checkpoint only after all gates pass.

The synthetic preset is presentation data, not a product rule, benchmark result, or case-study ground truth. It must remain outside Go analysis packages.

### Phase 3.1 acceptance criteria

- [DONE] One-click synthetic demo runs without manually selecting files and remains clearly labeled synthetic.
- [DONE] UI explains `MODIFY`, `VERIFY`, `UNRESOLVED`, confidence, and the three methods.
- [DONE] Playwright verifies page load, synthetic submission, result metrics, evidence expansion, and download links.
- [DONE] Existing manual analysis workflow remains functional through a separate Playwright upload test.
- [DONE] Frontend and backend quality gates pass with no external repository changes.
- [DONE] Initial local Git commit records the reviewed prototype without generated reports, data, dependencies, or secrets.

### Phase 3.1 verification note

Two serial Playwright tests pass using system Chrome: the one-click synthetic preset completes a hybrid analysis, renders changes, expands an evidence/score panel, and downloads Markdown; the manual flow uploads both BPMN files and completes a BM25 analysis. The final frontend type-check and optimized build pass. `npm audit` reports zero vulnerabilities after upgrading the test runner to patched Playwright 1.63.0. The Go suite, vet, and both binaries remain green. The result view was visually inspected from a real completed analysis and retains the same canonical backend output contract.

## 32. Phase 3.2 plan — case-study retrieval hardening

Authorization: user instruction on 2026-09-29 to continue with the highest-value improvement after reviewing the one-task Simple-ERP presentation result.

### Scope and decisions

- exclude byte-identical copies of BPMN/configuration method inputs when web uploads make path-based exclusion insufficient;
- exclude repository `.bpmn` process specifications from the software-source retrieval corpus;
- exclude documentation plan trees without excluding similarly named source packages;
- keep `MODIFY_TASK_LABEL` candidates at `VERIFY`, even with a strong exact trace, because wording changes do not prove behavior changes;
- suppress a Go package-level candidate when a more precise accepted symbol from the same file exists;
- preserve distinct matching symbols, deterministic ordering, evidence, score breakdown, and every existing method boundary;
- keep Simple-ERP read only and validate against the existing `PUR-PR-001` controlled mutation.

### Acceptance criteria

- [DONE] Exact copies of uploaded method inputs cannot enter retrieval, with a deterministic warning and regression test.
- [DONE] Repository `.bpmn` specifications cannot rank as source-impact candidates.
- [DONE] Documentation plan trees are excluded while non-documentation `plans` source packages remain indexable.
- [DONE] High-confidence label-only impacts remain `VERIFY` and retain their complete score/evidence.
- [DONE] A matching Go method shadows only the redundant package-level candidate from the same file.
- [DONE] The real one-task Simple-ERP report has no BPMN-input or implementation-plan candidate and ranks `Store.CreateRequest` first as `VERIFY`.
- [DONE] Full formatting, tests, vet, builds, diff checks, deterministic rerun, and external-repository status checks pass.

### Phase 3.2 verification note

All Go packages pass uncached tests and vet, both commands build, frontend type-check passes, and `git diff --check` passes. Two independent hybrid runs over Simple-ERP commit metadata `69724e6` and the controlled `PUR-PR-001` label mutation produced byte-identical canonical JSON (`C469C5400A55F43D98DCA880E22756F260BB33F335248B90D7E673A24E871759`). The hybrid result contains one BPMN change and ranks `backend/internal/purchasing/adapter/postgres/store.go / Store.CreateRequest` first as `VERIFY`; selected BPMN, documentation plans/review findings, and the redundant package symbol are absent. BM25 remains a lexical baseline and ranks `docs/BUSINESS-RULES.md` first, while configuration-driven emits its single frozen `store.go` rule; this difference is expected and useful for the method comparison. Simple-ERP remained clean on `fix/ta-baseline-readiness` at `69724e6`.

## 33. Phase 3.3 plan — focused presentation UI

Authorization: user instruction on 2026-09-30 to refine the presentation UI using the supplied neutral/blue palette and SF Pro system typography without expanding the analysis scope.

### Scope and decisions

- use the palette `#000000`, `#1D1D1F`, `#434344`, `#6E6E73`, `#F5F5F7`, and `#0071E3` with white surfaces;
- use the Apple system/SF Pro font stack without downloading or redistributing font files;
- remove decorative gradients and excessive colored cards in favor of flat hierarchy, whitespace, borders, and one blue accent;
- make candidate rank explicit and set the presentation default to top five while retaining the configurable range;
- preserve API contracts, canonical reports, analysis rules, accessible labels, and both manual and synthetic flows.

### Acceptance criteria

- [DONE] Home, analysis form, interpretation guide, and report view share the restrained visual system.
- [DONE] Candidate rows expose deterministic rank, action, target, confidence, evidence, and score breakdown.
- [DONE] TypeScript type-check and optimized Next.js production build pass.
- [DONE] Both Playwright flows pass serially using system Chrome.
- [DONE] A completed hybrid report is captured and visually inspected at desktop width.
- [DONE] `git diff --check` passes.

### Verification note

`npm run typecheck`, the optimized Next.js build, and two serial Playwright tests pass. The first browser run had a pre-page Chrome process exit while its second test passed; an immediate full rerun and the screenshot-enabled full run each passed both tests, so the launch failure is recorded as environment flakiness rather than an assertion failure. Visual inspection of `.data/ui-report.png` confirms the completed report, responsive two-column workspace, ranked candidates, expanded evidence, and score bars render with the intended neutral/blue hierarchy.

## 34. Recommended next research tranche — reviewer ground truth

- [DONE via Phases 3.5–3.8] Add a review-only workflow that records candidate relevance and expected action without altering inference output.
- [DONE via Phases 3.5–3.8] Export two independent reviewer files and reuse the existing reconciliation command for disagreements.
- [DONE via Phase 3.5] Add a same-input method comparison view for configuration-driven, BM25, and hybrid reports.
- [TODO] Run the frozen Simple-ERP scenarios only after reviewer ground truth is hidden from inference and adjudicated.
- [TODO] Report Recall@k, Precision@k, MRR/MAP, action classification, unresolved rate, false-positive rate, latency, and reproducibility without claiming cross-system generalization.

## 37. Phase 3.4C — extensible indexing contracts

Scope approved by user: introduce ordered SourceAdapter and RelationExtractor contracts, explicit TEXT_ONLY / AST_SYMBOLS / STATIC_CALLS capabilities, generic source relations, and deterministic per-file capability reporting. Go is the only native adapter; malformed Go falls back to text. New adapters register without changing retrieval. Capability does not imply accuracy or a full call graph. Keep existing public Build wrapper and report readers compatible. Type-aware interface dispatch remains subsequent work.

- [DONE] Registry, generic relations, capability reporting and fallback. Analyzer accepts an injected registry; ordered adapter/extractor identity is bound into engine-3.4c analysis IDs.
- [DONE] Mixed-language/custom adapter and failure tests; Go tests and vet, frontend typecheck/production build, 3 Playwright tests and diff check passed. Browser test verifies TypeScript fixture is explicitly TEXT_ONLY. JSON/Markdown round-trip test still passes with capability metadata.

Scope clarification: this phase supplies extensibility and truthful coverage, not new native language parsers or type-aware Go dispatch. Native support remains Go syntax plus limited static calls. Capability labels do not lower/raise confidence by themselves. Files, modules and calls still require evidence; architecture classification remains heuristic.

## 38. Phase 3.4D — conservative type-aware Go relations

Authorization: user instruction on 2026-09-30 to continue the highest-value mapping accuracy work.

### Scope and decisions

- resolve direct Go receiver-method calls when the receiver type is statically available from a method receiver, parameter, explicit local declaration, composite literal, or receiver struct field;
- follow selector chains such as `h.service.Execute()` through indexed struct fields;
- resolve an interface call to a concrete method only when exactly one indexed named type structurally implements the complete supported interface method set;
- compare the complete supported method set with normalized parameter/result type shapes and variadic shape; embedded interfaces and unsupported type expressions abstain;
- emit distinct `CALLS_METHOD` and `INTERFACE_DISPATCH` relations with located call-site evidence;
- keep propagation hybrid-only, one hop, discounted, and always `VERIFY`; static dispatch evidence does not prove runtime execution or a required modification;
- reject ambiguous receiver targets, ambiguous implementations, shadowed package identifiers, and unsupported dynamic/generic dispatch rather than guessing;
- bind changed extractor behavior to a new registry/engine identity and document the remaining static-analysis limits.

### Acceptance criteria

- [DONE] Direct receiver, receiver-field, and unique interface dispatch resolve to indexed method artifacts.
- [DONE] Multiple interface implementations, signature mismatches, unsupported embedded interfaces, shadowing, and ambiguous method targets abstain.
- [DONE] Deterministic tests prove one-hop retrieval with located method/interface evidence.
- [DONE] Per-file capabilities distinguish syntax symbols, static free-function calls, and type-aware relations.
- [DONE] Documentation and full Go/frontend quality gates pass; work remained inside this repository.

### Phase 3.4D verification note

Deterministic index tests cover direct receiver calls, composite-literal locals, receiver-field selector chains, cross-package named signature types, and unique interface dispatch in both propagation directions. Negative tests cover multiple implementations, method signature mismatch, embedded interfaces, duplicate targets, and receiver shadowing. Propagated candidates remain one-hop `VERIFY` with distinct located evidence. `go test -count=1 ./...`, `go vet ./...`, both Go command builds, frontend type-check/production build, and all three serial Playwright scenarios pass. `git diff --check` is required as the final workspace gate. Simple-ERP and POC TA were outside the writable workspace and were not accessed or changed in this tranche.

## 36. Phase 3.4B — source call evidence

Implementation tranche: resolve direct Go function calls within the same package and explicit imported module packages from the allowed inventory. Reject local shadowing and ambiguous targets. Receiver/interface dispatch remains deferred until type resolution is implemented; SAME_PACKAGE alone is not impact evidence. Hybrid expands only from accepted exact-trace symbol candidates, one hop, with VERIFY and a separate discounted dependency score. BM25/configuration baselines stay isolated. Version the analysis identity to prevent collisions with older persisted reports. Existing-branch/move semantics and UI filters remain follow-up work; this tranche prioritizes a tested source-relation foundation.

- [DONE] Static call extraction and deterministic one-hop retrieval: calls.go and propagation.go, tested with alias imports, shadowed symbols, duplicate targets and two-hop exclusion.
- [DONE] Negative fixtures, regression tests, and methodology documentation. All Go packages and go vet pass; analysis identity now includes engine version, addressing persisted-report collisions beyond test isolation.
- [DONE] Integration fixture recovers Persist through a resolved call despite absent query terms, repeat output is identical; frontend typecheck and three Playwright workflows pass.
- [DONE via Phase 3.4D] Conservative receiver/interface resolution is implemented; advanced BPMN movement/branch patterns and dedicated UI filtering remain follow-up work.

## 35. Phase 3.4A plan — composite BPMN change context

Authorization: user instruction on 2026-09-30 to continue backend development that improves mapping of task, gateway, and flow changes inside one BPMN process.

### Objective and boundary

Preserve the auditable atomic taxonomy while adding deterministic context for common multi-edit operations. Composite recognition explains how related node and flow edits form an insertion, removal with rewiring, or gateway branch addition. It does not claim behavioral equivalence, execute BPMN, or replace evidence-based candidate retrieval.

### Scope

- enrich BPMN elements with process, parent subprocess, boundary attachment, gateway direction, default flow, and event-definition metadata;
- keep nested subprocess nodes in explicit scope rather than treating every node as context-free;
- attach optional, backward-compatible change context containing pattern, predecessor, successor, and related flow IDs;
- recognize `INSERT_TASK`, `REMOVE_TASK_WITH_REWIRE`, and `ADD_GATEWAY_BRANCH` from graph topology;
- make hybrid and BM25 queries change-aware and include bounded neighbor context;
- add independent synthetic BPMN fixture pairs for all three patterns;
- render composite context in canonical Markdown and the web report without duplicating inference logic.

### Explicit non-goals

- no execution semantics, token simulation, automatic code changes, LLM, database, or framework-specific rules;
- no accuracy-improvement claim before hidden ground-truth evaluation;
- no flattening of several atomic changes into one destructive replacement of the existing taxonomy;
- no changes to Simple-ERP or POC TA.

### Acceptance criteria

- [DONE] Parser preserves process and nested subprocess context plus boundary/event/gateway attributes.
- [DONE] Three composite patterns are deterministic and covered by BPMN fixture tests.
- [DONE] Retrieval queries use pattern and neighboring element context with regression tests.
- [DONE] JSON, Markdown, and web UI expose optional context while old reports remain readable.
- [DONE] Formatting, Go tests/vet/build, frontend type-check/build/E2E, deterministic output, and diff checks pass.

### Phase 3.4A verification note

The parser test covers process ID, parent subprocess, boundary attachment, timer event definition, gateway direction, and default flow. Three independent BPMN fixture pairs produce `INSERT_TASK`, `REMOVE_TASK_WITH_REWIRE`, and `ADD_GATEWAY_BRANCH` while retaining their atomic task/gateway/sequence-flow changes. Two complete CLI runs per scenario produced byte-identical canonical JSON: insert `AB90C536385E987AAE52DE321F3EE2F6A9E48898BA7B24515321C5D9E6EDFF7B`, removal `4572C868A92D1F46120B4BCCFFEA352FF29B3F0AEFC5C1B4C48326896C968CC9`, and branch `E6AB24959A993EE9ED1B0ACE7F6C0A7C5387D5A3669F09B8D79941EF5C48009F`. Contextual retrieval has a regression proving predecessor/successor terms can retrieve a source symbol even when the inserted task label is absent. Structural fallback now refuses cross-process matches.

All Go tests, vet, command builds, frontend type-check, optimized Next.js build, `git diff --check`, and three serial Playwright flows pass. Playwright storage moved from persistent `.data/playwright` to its cleaned `test-results` tree after schema evolution correctly triggered the report store's overwrite protection; no production storage semantics were weakened. The third browser test uploads the insertion fixtures and verifies the rendered `INSERT TASK` context end to end.

## 40. Phase 3.6 — evaluation dashboard

Authorized continuation: upload two annotations after inference, reconcile in Go, explicitly adjudicate conflicts in the browser, and evaluate stored predictions through a separate bounded JSON endpoint. Evaluation requires full change coverage, unique methods and identical input fingerprints. Missing reviewer cases cannot be silently treated as negative cases. Predictions stay immutable; review/evaluation results are downloadable without server persistence. File-level metrics retain existing evaluator semantics; FP/(TP+FP) will be labeled false discovery proportion, not statistical FPR. Legacy annotations have no snapshot binding, so the reviewer must confirm applicability to the selected snapshot.

- [DONE] Separate reconciliation/evaluation endpoints and strict validation.
- [DONE] Conflict decisions, metrics table and JSON/Markdown downloads in modular dashboard.
- [DONE] Backend negative tests and browser workflow proof, documentation and quality gates.

Verification: Go tests and vet pass across all packages; TypeScript and production build pass; all six serial Playwright workflows pass. New E2E uploads two synthetic annotations for three stored methods, detects a conflict, verifies evaluation is disabled until adjudication, checks metrics and downloaded provenance, and invalidates results on decision changes. Backend tests cover malformed/oversized/unknown JSON, duplicate reviewer IDs, unresolved/missing/duplicate decisions, empty coverage, fingerprint mismatch, duplicate methods, deterministic repeat results, unsafe expected paths, and immutable prediction storage. The rendered metric table was visually inspected from a real browser screenshot. No dependencies, database, external services, or changes to study repositories were introduced. Changes remain uncommitted.

Limitations: annotation v1 lacks cryptographic snapshot provenance and requires user confirmation; reviewer independence is a research procedure. Existing file-level metric semantics remain unchanged, including optional metadata handling and empty-case averaging. No latency, statistical FPR, universal accuracy, or cross-system generalization claim is made. The dashboard evaluates already-produced reports; inference never receives annotations.

## 41. Phase 3.7 — blind reviewer pack and snapshot provenance

Authorization: user instruction on 2026-10-01 to continue autonomous development toward a robust, thesis-ready mapping workflow.

Problem: browser-local reviewer export is useful for exploratory review, but displaying ranked candidates can anchor reviewers. Version-1 annotations also do not bind themselves to an immutable analysis snapshot. This tranche adds a separate coordinator workflow; it does not change inference, scoring, or fabricate ground truth.

Scope and decisions:

- add a pure `reviewpack` use case that validates one snapshot across canonical reports, rejects duplicate methods and mismatched repository/fingerprint/change sets, and derives deterministic case order;
- emit two version-1 annotation templates containing BPMN change cases and a deliberately invalid `REVIEW_REQUIRED` sentinel; reviewers replace it with findings or remove it for an explicit negative case, so untouched templates fail closed;
- emit a blind Markdown guide containing BPMN old/new/context information but no candidate, score, confidence, evidence, analysis ID, or method identity;
- emit a coordinator-only manifest binding repository revision, input fingerprint, report SHA-256, analysis IDs, methods, case digest, and deterministic pack ID;
- add `prepare-review` CLI with repeatable `--report`, explicit reviewer IDs, bounded regular-file reads, and refusal to overwrite an existing pack artifact;
- allow `reconcile --manifest` to validate coordinator metadata, exact reviewer case coverage, and the fail-closed sentinel before embedding pack ID/fingerprint/case digest in reconciliation output;
- document that reviewer identity, independence, implementation inspection, and final adjudication remain human research responsibilities.

Acceptance criteria:

- [DONE] Blind artifacts contain no prediction-derived candidates or ranking signals.
- [DONE] Snapshot/method/change mismatches, duplicate methods/analysis IDs/cases, empty change sets, and untouched templates fail closed.
- [DONE] Identical inputs produce byte-identical JSON/Markdown and pack ID regardless of report argument order.
- [DONE] CLI integration, architecture boundary, unit tests, Go formatting/test/vet/build, and diff checks pass.
- [DONE] Documentation explains the coordinator-manifest boundary and remaining human steps without making accuracy claims.
- [DONE] Manifest-bound reconciliation retains snapshot provenance and rejects tampered metadata, incomplete cases, or untouched sentinels; legacy reconciliation remains compatible.

### Phase 3.7 verification note

`prepare-review` accepts one to sixteen bounded, regular, non-symlink canonical report files. The pure use case sorts reports by method, verifies a shared input fingerprint/repository/change set, hashes report bytes and normalized cases, and emits deterministic reviewer templates, a blind guide, and a coordinator-only manifest into a new directory via staging and atomic rename. Tests prove report-argument-order invariance, reject duplicate/mismatched/empty inputs, verify that candidate paths and confidence do not leak into reviewer artifacts, reject untouched sentinel templates, validate manifest integrity and exact completed-case coverage, embed snapshot provenance in manifest-bound reconciliation, and refuse pack overwrite. `go test -count=1 ./...`, `go vet ./...`, both command builds, and `git diff --check` pass. The Go tool emitted non-fatal stat-cache access warnings for the user module cache while using the repository-local build cache; all commands exited successfully. No frontend file, study repository, network service, database, or inference/scoring behavior changed.

Remaining research boundary: version-1 annotation files do not independently embed the pack ID; the binding is enforced when the coordinator supplies the manifest to reconciliation. Independent reviewer identity, blind distribution, implementation inspection, annotation completion, adjudication, and any real accuracy claim remain human-controlled study steps.

## 42. Phase 3.8 — web blind-review pack and robustness automation

Authorization: user instruction on 2026-10-01 to continue autonomous development.

Scope:

- expose the existing pure `reviewpack` use case through a bounded post-inference API that loads immutable stored reports only;
- add a restrained web panel that requests a same-snapshot pack and downloads the manifest, blind guide, and two fail-closed reviewer templates separately;
- never send reviewer findings or ground truth into analysis, retrieval, or scoring;
- add API and browser tests proving method/fingerprint validation and absence of candidates/scores from blind artifacts;
- extend deterministic robustness coverage and batch experiment orchestration only after the web boundary is verified.

Acceptance criteria:

- [DONE] Web-generated pack uses the same Go domain use case as the CLI and contains snapshot provenance.
- [DONE] Invalid report IDs, duplicate methods, mismatched snapshots, malformed JSON, and identical reviewer IDs fail closed.
- [DONE] Browser downloads contain sentinels and no candidate, confidence, evidence, ranking, or score data.
- [DONE] Go, TypeScript, production build, Playwright, and diff gates pass.

### Phase 3.8 verification note

The new post-inference endpoint accepts only immutable analysis IDs, a shared fingerprint, and two distinct reviewer identifiers. It reloads canonical reports, computes their byte hashes, and delegates to `reviewpack.Build`; no candidate-building logic exists in TypeScript. API tests prove deterministic responses, snapshot/method/reviewer validation, sentinel presence, and absence of candidate paths/confidence. The browser test runs all three methods, creates a pack, verifies the response has no `impacts` or `confidence`, downloads Reviewer A JSON, and validates every fail-closed sentinel. Visual inspection of `.data/ui-blind-review.png` confirms the restrained panel and role-separated downloads render coherently in the completed report workspace.

The existing frozen `benchmark` runner already supplies batch scenario × method execution, repetition checks, input hashes, and post-inference ground-truth parsing. A second runner was deliberately not added. Existing synthetic diff/index/retrieval tests already cover add/remove/modify task, actor, gateway condition, sequence flow, data/state/outcome, three composite topology patterns, ambiguous matching, deterministic ranking, and conservative static relations. This tranche adds web-boundary robustness rather than duplicating those suites.

## 43. Phase 3.9 — topology robustness, demo scenarios, and CSV export

Authorization: user instruction on 2026-10-01 to continue all remaining autonomous development items.

- [DONE] Recognize `MOVE_TASK` conservatively from stable task IDs whose predecessor/successor sets change, while retaining atomic `MODIFY_SEQUENCE_FLOW` changes.
- [DONE] Recognize `REMOVE_GATEWAY_BRANCH` only for a removed branching gateway with at least one preserved neighboring element.
- [DONE] Add negative controls proving unchanged topology and condition-only flow edits are not labeled as task moves.
- [DONE] Add focused web presets for insert task, add/remove gateway branch, move task, and ambiguous matching.
- [DONE] Export deterministic spreadsheet-ready benchmark CSV without volatile timing.
- [DONE] Pass Go, frontend, browser, build, and diff quality gates.

### Phase 3.9 verification note

Two new composite contexts preserve the canonical taxonomy rather than inventing replacement change types. `MOVE_TASK` annotates sequence-flow edits only when a stable same-scope task's predecessor or successor ID set changes. `REMOVE_GATEWAY_BRANCH` requires a removed branching gateway and a preserved neighboring element. Fixture tests cover both patterns; negative controls prove identical graphs emit nothing and a condition-only sequence-flow edit remains context-free. The web selector provides comprehensive, insertion, branch addition/removal, move, and deliberately ambiguous scenarios. The ambiguous preset initially sat on a floating-point similarity boundary and correctly produced ordinary add/remove changes; adding equal actor evidence made the ambiguity explicit without weakening the matcher threshold.

Benchmark output now includes deterministic `benchmark-report.csv` with scenario and aggregate rows, four-decimal metrics, CSV escaping, reproducibility metadata, and the accurately named `false_discovery_proportion`; volatile timing remains separate. `go test -count=1 ./...`, `go vet ./...`, TypeScript typecheck, production build, and the final seven-test Playwright suite pass. The targeted corrected-control rerun also passed. `git diff --check` remains the final workspace gate.

## 44. Phase 4.0 — persisted analysis history

Authorization: user redirected final work toward application development rather than presentation guidance on 2026-10-01.

- [DONE] Add an inward report-store listing contract with deterministic summaries and no absolute paths.
- [DONE] Harden stored-report reads against symlink/non-regular/oversized artifacts.
- [DONE] Expose bounded `GET /api/v1/analyses` without invoking inference.
- [DONE] Add a restrained web history panel that reloads immutable reports after browser refresh.
- [DONE] Keep history timestamp-free and deterministically ordered because canonical reports contain no trustworthy creation time.
- [DONE] Correct benchmark Markdown terminology from statistical FP rate to false discovery proportion.

### Phase 4.0 verification note

`reportstore.Store` now lists typed summaries while the filesystem adapter accepts at most 1,000 canonical analysis directories and reads only regular, non-symlink JSON artifacts up to 64 MiB. Summaries contain repository metadata, method, fingerprint, and counts but no local path. Ordering is by deterministic analysis ID rather than filesystem time. `GET /api/v1/analyses` is read-only and never invokes the analyzer. The web history refreshes after a successful analysis and reopens an immutable report after a full browser reload. Browser testing exposed the existing sticky form covering the new lower panel; removing sticky positioning fixed the real pointer-interception defect. The targeted history test and final seven-test browser suite pass, together with all Go tests/vet/builds, TypeScript typecheck, Next production build, and `git diff --check`.

## 39. Phase 3.5 — web method comparison and reviewer artifact

Authorization: user instruction on 2026-09-30 to continue with the evaluation and method-comparison tranche.

### Scope and decisions

- add an opt-in comparison run using byte-identical repository/BPMN/top-k metadata for BM25 and Hybrid, plus Configuration-Driven only when a frozen mapping path is supplied;
- execute method requests sequentially to bound local CPU/memory and verify that all returned reports share one `input_fingerprint`;
- present retrieval counts, unresolved counts, per-change ranked candidates, and overlap against Hybrid without calling those values accuracy metrics;
- keep canonical reports method-specific and downloadable through their existing immutable analysis IDs;
- add a browser-local reviewer form that can select predicted candidates, change expected actions, and add missing expected impacts manually;
- export version-1 reviewer annotation JSON compatible with the existing reconciliation command;
- never send reviewer choices or ground truth to the analysis API, report store, candidate retrieval, scoring, or comparison requests;
- retain single-analysis mode and the current presentation styling.

### Explicit non-goals

- no accuracy/recall claim before independent annotation and reconciliation;
- no automatic ground-truth derivation from ranked candidates;
- no database, reviewer accounts, authentication, adjudication UI, or server-side annotation persistence;
- no Configuration-Driven comparison without an explicitly supplied frozen mapping file.

### Acceptance criteria

- [DONE] The same web input can produce BM25 and Hybrid reports, optionally Configuration-Driven, with a verified shared input fingerprint.
- [DONE] The comparison UI clearly separates retrieval differences from accuracy metrics.
- [DONE] Reviewer annotation export supports selected candidates, manual missing impacts, expected actions, and empty/negative cases.
- [DONE] Annotation data remains browser-local and matches the existing reconciliation schema.
- [DONE] Existing single analysis and all previous browser workflows remain functional.
- [DONE] Type-check, production build, Playwright, Go tests/vet, and diff checks pass.

### Phase 3.5 verification note

The synthetic comparison runs all three methods sequentially with one verified fingerprint and distinct immutable reports; its frozen neutral configuration is a method fixture, not ground truth. Browser tests select a method card and verify the corresponding report. Reviewer E2E selects a predicted file, adds a missing expected file manually, downloads JSON, and verifies the version, reviewer identity, and expected impact structure. A real browser screenshot confirms the restrained comparison UI remains usable within the existing layout. `go test -count=1 ./...`, `go vet ./...`, both Go command builds, frontend type-check/production build, all five serial Playwright workflows, and `git diff --check` pass.

## 45. Proposed Phase 4.1 - semantic retrieval research baseline

Status: IN PROGRESS for an isolated offline prototype authorized by the user on
2026-10-10. Inclusion in the final thesis comparison remains advisor-gated.

- [DONE] Record recent BPMN semantics, BPMN-to-source traceability, and semantic
  requirement-to-code literature in `docs/SEMANTIC_RETRIEVAL.md`.
- [DONE] Define the method boundary, reproducibility metadata, evidence
  requirement, and distinction from RAG/LLM verification.
- [TODO] Obtain advisor approval for a fourth independent experimental method.
- [TODO] Freeze a synonym/no-stable-ID benchmark and independently reviewed
  ground truth.
- [DONE] Select and record the prototype runtime: Python 3.12,
  sentence-transformers 5.2.3, and cached `all-MiniLM-L6-v2`. The canonical
  report records provider/model; environment hashing and cross-machine numeric
  reproducibility remain experiment work.
- [DONE] Implement Semantic Retrieval as an isolated offline mapping adapter,
  CLI/API/web method, and comparison column. It uses bounded embedding input,
  deterministic ranking, `VERIFY` actions, explicit errors, and no silent
  fallback. It remains independently selectable after fusion was added.
- [DONE] Verify unit tests and two byte-identical real offline CLI runs; the
  smoke excluded ground-truth/checkpoint paths and proves reproducible
  functionality on the current machine only.
- [DONE] Remove Configuration-Driven from the primary web workflow and target TA
  method set; retain the existing adapter temporarily only for internal backward
  compatibility with older artifacts.
- [DONE] Implement RRF fusion and enable comparison of BM25-only, Semantic-only, BM25 +
  Semantic Fusion, and BPMN-Context-Aware Hybrid Retrieval on byte-identical inputs without a
  premature generalization claim.

Implementation evidence (2026-10-10): RRF unit tests cover agreement, stable
ties, evidence union, abstention, action safety, and graph-evidence precedence.
Real offline CLI smoke runs produced evidence-bearing `fusion` and `hybrid`
reports for all 12 synthetic changes. `go test -count=1 ./...`, `go vet ./...`,
TypeScript type-check, Next.js production build, `git diff --check`, and all
seven serial Playwright workflows pass. These results prove implementation
mechanics only; comparative accuracy remains blocked on hidden ground truth.

## 46. Phase 4.2 - current benchmark contract and semantic preflight

Authorization: user instruction on 2026-10-14 to continue hardening all
development work that does not require reviewer input or an LLM credential.

- [DONE] Require BM25, Semantic, BM25 + Semantic RRF, and BPMN-Context-Aware Hybrid Retrieval in
  the primary frozen benchmark manifest.
- [DONE] Retain Configuration-Driven only as an optional legacy comparator with
  a mandatory frozen mapping path and SHA-256 when selected.
- [DONE] Remove the obsolete mandatory mapping fixture from the primary
  synthetic benchmark.
- [DONE] Add an offline semantic runtime capability endpoint using the same
  configured embedder as analysis.
- [DONE] Prevent Semantic, Fusion, Hybrid, and method comparison from submitting
  when the local embedding runtime is unavailable; keep BM25 independent.
- [DONE] Bound the public capability response and keep subprocess/runtime detail
  in local logs.
- [DONE] Run full Go, vet, Next.js build, Playwright, and diff quality gates and
  record final evidence.

This tranche does not invent case-study ground truth, recalibrate scores from
synthetic fixtures, download a model, call an external API, or enable an LLM.

Verification evidence (2026-10-14): the frozen four-method manifest completed
two repetitions per method with benchmark ID
`1b74d82408aa04a940abbfe642e8e08b`; all 11 serial Playwright workflows passed
in 2.3 minutes, including real offline Hybrid execution and the unavailable
runtime/BM25 fallback control. Full Go tests, vet, TypeScript type-check, Next.js
production build, and `git diff --check` pass. These remain implementation and
reproducibility checks, not retrieval-accuracy claims.

## 47. Phase 4.3 - research baseline and pre-provider hardening

Authorization: user instruction on 2026-10-14 to implement the six remaining
development steps before discussing evaluation metrics.

- [DONE] Add root `BASELINE-METHODS.md` with the four method contracts,
  limitations, and primary literature; correct the stale Hybrid/Semantic note.
- [DONE] Bind semantic reports and analysis identity to adapter SHA-256, Python
  version, sentence-transformers version, embedding dimension, and loaded model
  artifact SHA-256 without exposing local cache paths.
- [DONE] Add a synthetic vocabulary-mismatch/no-trace-ID benchmark scenario.
- [DONE] Export combined deterministic metrics and separately labeled volatile
  latency summaries as benchmark comparison Markdown and CSV.
- [DONE] Implement an isolated OpenAI-compatible planning provider with strict
  typed JSON, bounded I/O, HTTPS/loopback policy, and mock-server tests; keep
  `NoLLMProvider` as production default.
- [DONE] Prevent provider action escalation, code-block output, unbounded plan
  prose, and `COMPLETE` status while reviewer issues remain.
- [DONE] Complete full quality gates and record the final benchmark
  evidence before metric-design discussion.

This phase does not activate an external model, send repository data, create
case-study ground truth, or claim accuracy from synthetic controls.

Interim benchmark evidence: two scenarios, four methods, and two repetitions
completed with benchmark ID `631994bff5746c42dd955fd828c171cf`.
The vocabulary-mismatch control produced Recall@1 `0.0000` for BM25 and
`1.0000` for Semantic, Fusion, and Hybrid. This demonstrates that the control
separates lexical and semantic behavior; it is deliberately constructed
synthetic evidence and not an accuracy or generalization result. The timing
table is explicitly labeled shared-process/sequential because Fusion bears the
cold model start and later methods reuse cached embeddings.

Final verification evidence (2026-10-14): focused semantic, benchmark, LLM,
and planning tests pass; the full Go suite and `go vet ./...` pass; TypeScript
type-check and the Next.js production build pass; `git diff --check` passes.
The serial Playwright suite passed 10/11 on its first run because the real local
embedding runtime transiently failed its cold preflight under the combined
browser/build workload; the failed evaluation workflow passed on an isolated
rerun (1/1). The separate frozen benchmark completed every real offline
Semantic/Fusion/Hybrid repetition. This is recorded as an environment-sensitive
cold-start limitation, not hidden as an accuracy failure.

## 48. Phase 4.4 - product/research boundary and action evaluation

Authorization: user instruction on 2026-10-14 to retain Hybrid as the product
method while keeping component methods available for research comparison.

- [DONE] Make Product mode the web default and submit only BPMN-Context-Aware Hybrid Retrieval.
- [DONE] Move BM25, Semantic, Fusion, and four-method comparison behind an
  explicit Research mode control without removing the reproducible experiment
  path.
- [DONE] Add required-action macro-F1 and a fixed-label action confusion matrix
  to canonical evaluation JSON, Markdown, API output, and the web dashboard.
- [DONE] Count an expected file that is absent from retrieval as predicted
  `UNRESOLVED` in the confusion matrix; retain existing action accuracy as a
  retrieved-file-only diagnostic to avoid silently changing its meaning.
- [DONE] Add `benchmark-ablation.md` and `.csv` with paired aggregate deltas
  against Hybrid, explicitly labeled descriptive rather than causal.
- [TODO] Add Docker Compose as a separately validated local packaging phase;
  it is intentionally outside this focused commit.

Verification evidence (2026-10-15): `go test -count=1 ./...`, `go vet ./...`,
TypeScript type-check, and production build pass. All 11 serial Playwright
workflows pass after the Product/Research selector update, including BM25
fallback, four-method comparison, canvas editing, immutable evaluation, and
reviewer export. No network provider or LLM is enabled.

## 49. Report explanation correction (2026-10-19)

- [DONE] Explain actual fusion ranks, including one-sided and unavailable ranks,
  rather than asserting every fused candidate was found by both methods.
- [DONE] Label textual data/status clues as suggestions, not verified dependencies.
- [DONE] Isolate nested evidence disclosures from candidate-row layout, wrap long
  source paths/excerpts, and improve keyboard focus and detail readability.
- [DONE] TypeScript typecheck, two report-explanation regression tests, and
  `git diff --check` pass. Worker execution required sandbox escalation.
- [DONE] Browser visual verification of the updated report layout (phase 50).
- [DONE] Audit backend structured-evidence extraction against actual source lines
  and correct cross-line evidence attribution (phase 50).

Existing ranking weights, report JSON, and retrieval algorithms are unchanged.

## 50. Evidence correction and report navigation (2026-10-19)

Authorization: user requested priorities 1-3: accurate structured evidence,
Code/Tests/Docs filtering, and per-file result grouping.

- [DONE] Replace artifact-wide structured labeling with syntax/query co-occurrence
  on the actual displayed line. Bound excerpts and preserve original line offsets.
- [DONE] Reject task-ID-only lines, unrelated SQL elsewhere, generic uppercase IDs,
  test-directory-only hints, comments, and syntax clipped out of excerpts.
- [DONE] Bump engine identity to avoid collisions with immutable historical reports.
- [DONE] Add display-only filters and By file view preserving change/function
  associations, individual actions, scores, original ranks, and unresolved changes.
- [DONE] Keep full JSON/Markdown exports and evaluation inputs unaffected by filters.
- [DONE] Increase report detail readability and verify desktop/mobile layout.

Verification: `go test -p 1 ./internal/...`, `go vet -p 1 ./...`, CLI tests
excluding `TestRunFrozenBenchmark`, TypeScript typecheck, and four focused
Playwright tests pass. Browser checks cover filters, grouped files, retained
unresolved text, unchanged report data, original ranks, nested disclosures, and
mobile overflow. Desktop/mobile screenshots were inspected.

Full `go test ./...` initially failed from Windows memory allocation exhaustion.
A serial retry passed internal packages but the CLI frozen benchmark failed
loading the offline embedding runtime. Full real-model benchmark verification
remains open; no accuracy improvement is claimed. No dependency download or
external model call occurred. Existing user BPMN files were not edited.

Limitations: patterns are conservative single-line textual hints, not full
multi-language parsers; some valid multiline syntax will be missed. Result
categories use layer and path conventions. Newly supported evidence requires a
fresh analysis; reopening an old report only changes its display.

## 51. Local semantic reliability and scenario workflow

Authorization: user requested the next development priorities (semantic stability,
scenario coverage, batch web runs, and explicit method differences).

- [DONE] Bound embedding input and thread use; classify memory and input-limit errors in the API, and retain bounded corpus/query cache entries (semantic and API unit tests).
- [DONE] Add eight reusable synthetic scenarios, including a no-change control, and assert expected BPMN diffs in `internal/diff/catalog_test.go`.
- [DONE] Add sequential, cancellable Research batch analysis with per-run reports/errors and local summary export (browser tests for partial failures and cancellation).
- [DONE] Compare shared and method-exclusive files for identical input fingerprints and Top-K (browser unit test).
- [DONE] Verify `go test -p 1 ./...` (including the frozen offline model case), `go vet -p 1 ./...`, frontend production build/typecheck, 18 browser E2E tests, `gofmt -l`, and `git diff --check`; document limitations.

Design: the existing benchmark runner requires hidden ground truth. The new web
batch uses the existing analysis API for exploratory runs; it must not fabricate
accuracy metrics or silently replace failed Semantic runs with BM25. Evaluation
continues through the existing separate ground-truth workflow. Case-specific
retrieval exclusions are not introduced to hide cross-module false positives.
The batch is an exploratory UI, not the hidden-ground-truth benchmark. A failed
Semantic run remains failed. A cancelled request may still finish on the server
and appear in Saved analyses. Cross-module candidates remain visible for review;
the current generic lexical/context terms can over-retrieve without independent
ground truth for weight tuning.

## 52. Gateway semantics and bounded semantic passages (2026-10-20)

- [DONE] Detect stable-ID gateway type, direction, and default-flow changes as
  `MODIFY_GATEWAY_SEMANTICS`; preserve separate flow/condition changes and
  include explicit old/new values in the reason. Synthetic XOR-to-AND test passes.
- [DONE] Give each source artifact one short semantic passage and a second tail
  passage when embedding budget permits; rank by the best passage and cite its
  bounded excerpt. Long-artifact regression test passes.
- [DONE] Bump engine identity because diff and semantic inputs changed.
- [DONE] Full `go test -p 1 ./...` (including the offline-model frozen case),
  `go vet -p 1 ./...`, `gofmt -l`, and `git diff --check` pass.
- [TODO] Establish hidden scenario ground truth before comparing BM25, Semantic,
  Fusion, and BPMN-context hybrid accuracy or retuning weights.

Limits: the 600-character bound is not a tokenizer guarantee; long middle
sections can be missed. No DPR/model download or LLM is added. Graph topology
alone cannot prove an XOR/AND join type or a code change; BPMN XML semantics
and candidate evidence are still presented for review.

## 53. Add/remove task demo diagram integrity (2026-10-20)

- [DONE] Add BPMN-DI shapes and edge waypoints to the web's synthetic
  `insert-task-old/new.bpmn` pair. Reversing the pair tests removal; neither
  modifies Simple-ERP or its source code.
- [DONE] Verify graph-diff catalog and TypeScript typecheck.
- [DONE] Add browser regression checks for every task and nonzero-length
  sequence-flow path. Both diagrams imported without BPMN warnings and passed
  individually; screenshots show the added user task and connections.
- [DONE] Convert the synthetic add/remove gateway-branch pair to visual-ready
  BPMN-DI, with an explicit XOR split and branch waypoints. Graph-diff catalog
  still detects `ADD_GATEWAY`/`REMOVE_GATEWAY`, task, and flow changes.
- [DONE] Center the canvas when fitting an imported diagram. This prevents a
  compact but tall branch diagram from opening beneath the left tool palette.
  Typecheck and seven focused browser tests (four diagram imports and three
  editor regressions) pass; screenshots were inspected.
- [TODO] Convert the remaining synthetic role, label, move, and unresolved
  scenarios to visual-ready BPMN-DI if they must be presented in external BPMN
  editors. Their mapper graph fixtures remain valid, but do not yet claim
  visual-editor readiness.

One combined Playwright run had a transient Windows worker-process exit; an
isolated rerun of the affected scenario passed. No mapping accuracy claim is
derived from canvas rendering.

## 54. Source-backed P2P gateway/task scenario (2026-10-20)

- [DONE] Keep `C:\Simple-ERP` read-only and derive a new P2P BPMN only under
  `docs/demo-bpmn/`. Its baseline BPMN blob matches `69724e6` and current
  checkout `c928a29`; the rest of the checkout is dirty and must not be
  labeled as a frozen `69724e6` source snapshot.
- [DONE] Add one XOR gateway, one Admin user task, and four changed/new flows,
  with matching lane references, incoming/outgoing references, and BPMN-DI.
- [DONE] Run BM25-only CLI smoke against the current Simple-ERP source:
  exactly `ADD_GATEWAY`, `ADD_TASK`, and four `MODIFY_SEQUENCE_FLOW` changes;
  5 candidates per change at Top-K 5. Output is ignored under `.data/`.
- [DONE] Import the complete new BPMN in the local bpmn-js canvas without
  warnings and assert gateway/task and all four changed edges are rendered.
- [TODO] Obtain independently reviewed ground truth before claiming which
  source files truly need modification or comparing mapping accuracy.

The visual scenario tests importability and the smoke run tests mechanics,
not software correctness or measured retrieval quality. No Simple-ERP source
or POC TA files were changed by this work.

## 55. Corpus and lexical-query correction (2026-10-21)

Motivation: a user-supplied P2P impact report indexed 319 files, including a
large BPMN specification/tooling tree. Fragmented generated IDs and flow labels
also contributed generic query words, placing documentation and unrelated
source artifacts above several purchase-request functions. The initial report
is diagnostic evidence only; it is not labelled ground truth.

- [DONE] Exclude conventional BPMN specification trees, audit/review/test-plan
  and phase-design artifacts, traceability verification scripts, build/test
  output, and lockfiles from prediction input. Inventory tests verify exclusions
  and preserve ordinary business-rule documentation.
- [DONE] Query labels, state/data terms, and named BPMN neighbors without
  generated flow/element ID fragments or change-type words; retain exact
  changed-element IDs as separate `TRACE_ID` evidence. Unlabelled elements
  retain an ID-fragment fallback. Actor terms enter actor-role changes.
- [DONE] Cite the line with the most distinct lexical query terms, or the path
  when the match is path-only. Unit tests cover both excerpt cases.
- [DONE] Bump analysis identity to `engine-3.4i-focused-retrieval-queries`;
  methodology, security policy, and README reflect the changed inference
  inputs. Historical reports remain intact.
- [DONE] `go test -p 1 ./...`, `go vet -p 1 ./...`, and `git diff --check`
  pass. A full offline Hybrid CLI run over the read-only current Simple-ERP
  checkout produced six BPMN changes from 198 indexed files and no runtime
  error. The former report indexed 319 files; the corpora are not identical
  and their ranking scores must not be compared as an accuracy metric.
- [DONE] Separate code-evidence, semantic, and RRF numbers in JSON,
  Markdown, and web UI; label missing direct IDs for new BPMN elements as
  expected without silently changing retrieval weights. Focused Go report,
  fusion and retrieval tests, full `go test -p 1 ./...`, `go vet -p 1 ./...`,
  web typecheck, and `git diff --check` passed.
- [DONE] Repeat the full CLI run byte-for-byte: canonical JSON and Markdown
  SHA-256 hashes match across two full offline Hybrid executions with the
  same inputs/configuration before the final Semantic query correction
  (`F6D0C6E9BAEAB3FB6EC3F590F38840BDF506033DE9FBAB836413981B01BA8EE2`
  for JSON). This hash is not the final engine version's reproducibility proof.
- [DONE] Focus Semantic queries on process labels/neighbor names rather than
  generic change/type/actor boilerplate. Unit tests pass. On the same P2P
  input, the offline smoke run still yields six BPMN changes; the plausible
  purchase-request UI component appears in the larger Top-50 review pool and
  unrelated Sales/Inventory store candidates move below Top-20. This is
  screening evidence, not measured accuracy.
- [DONE] Repeat final-engine Hybrid execution with identical Top-K 50:
  JSON and Markdown are byte-identical (`CFA1708C8499E1B6BB858539E6ADB9E127BF12146A19ACE3064CFE293F094EC5`
  JSON SHA-256). All emitted candidates have evidence, no absolute local path
  appeared, and an old=new BM25 control yielded zero changes/warnings.
- [TODO] Obtain independent, adjudicated hidden ground truth and measure
  Recall@k/Precision@k, cross-module false positives, and omitted
  frontend/API candidates before adjusting signal weights or fusion policy.

Self-review: the new query reduces obvious structural-token noise but does not
prove the top-ranked functions must change. The result still includes some
cross-domain candidates and omits a plausible purchase-request UI component
from Top-20. RRF agreement and its high number are not calibrated confidence.
No Simple-ERP-specific suppression, external model, or automatic code edit was
introduced. Repo architecture remains Go-owned analysis with UI as an adapter.

## Staff P2P diagnostic follow-up (2026-10-27)

- [DONE] Prepare three Staff L3 case-study inputs under
  `experiments/simple-erp/staff-p2p`: inserted task, gateway condition, branch
  label. CLI probes distinguish these changes; the source baseline hash stays
  unchanged. Baseline working tree is not frozen for formal evaluation.
- [DONE] Distinguish a branch-label edit from a changed explicit condition;
  retain label-as-condition behavior when no expression exists. Regression
  tests cover both cases.
- [DONE] Exclude `.generated` and process-model tooling directories under
  `scripts`, using the same generic naming convention as `docs`. The probe
  exposed generator/preview contamination; inventory regression covers it.
- [DONE] Surface sparse semantic support in fusion warnings without changing
  RRF weights or thresholds. Tests check the warning and tie independence.
- [DONE] Diagnose false-positive login/README ranking: the old MiniLM probe
  admitted only two semantic candidates at the configured cutoff, while the
  explicit process name was discarded. Record paired probes with identical
  input fingerprints in the Staff case-study diagnostic report.
- [DONE] Preserve explicit process names through parser and inserted/removed
  task context; include this bounded anchor in Hybrid semantic queries and in
  context BM25. Plain Semantic/BM25 baselines remain unchanged. Unit tests
  cover parser-to-diff propagation and Hybrid-only semantic wiring.
- [DONE] Support E5 query/passage prefixes for the three known multilingual-E5
  model IDs. Expose the input profile in method metadata and warn that its
  cosine threshold needs independent calibration. Default remains MiniLM.
- [DONE] Add an opt-in six-query bilingual model smoke test; cached MiniLM
  and E5 both run offline. These authored controls are not an accuracy claim.
- [DONE] Version analysis identity for explicit-condition diff and process
  context changes so new reports do not silently reuse older analysis IDs.
- [TODO] Freeze source and independent labels before choosing a default model
  or tuning thresholds. Dashboard tests still outrank the PR form in the new
  Staff probe. The process anchor is a generic context correction, not proof
  of improved file-level accuracy or of mandatory code changes.

Scope note: E5 support is an optional local embedding profile, not a new LLM
agent or a model download. No default model, RRF weight, or threshold changed.
Only insertion/removal patterns currently carry the new process-name context.

Verification: full `go test ./...`, final focused retrieval/diff/semantic/app
tests, `go vet ./...`, changed-file `gofmt -l`, and
`git -c core.autocrlf=false diff --check` pass. Both full offline model probes
completed with matching input fingerprints. The baseline BPMN hash is
unchanged. A later Playwright run exercised the real web upload/report/save
path for the inserted Staff task and the two lighter gateway/branch scenarios;
all three passed. No frontend product code changed.

- [DONE] Normalize effective repository size/file limits before computing the
  input fingerprint. HTTP omitted limits and CLI explicitly supplied defaults,
  producing different identities for an equivalent corpus. A focused unit test
  covers omitted/explicit defaults. The Staff gateway browser/CLI control now
  has identical analysis ID, fingerprint, JSON, and Markdown. Engine identity
  bumped to `engine-3.5f-normalized-ingest-identity`.

Final gate after normalization: `go test ./...`, `go vet ./...`, web
`npm run typecheck`, changed-file `gofmt -l`, and
`git -c core.autocrlf=false diff --check` passed. The three Staff browser
scenarios passed; the add-task Hybrid run preceded the identity-only
normalization, while both BM25 browser controls were rerun afterward.

## Staff source-based impact review (2026-10-27)

- [DONE] Inspect Simple-ERP's PR form, API helper, HTTP handler, application
  contract, transaction store, request-size middleware, schema, OpenAPI, and
  direct PR-creation test callers without changing the case-study repository.
  Record a design-specific provisional review under `docs/ground-truth` so it
  cannot enter retrieval. The checkout is dirty; this is not blind truth.
- [DONE] Correct the initial diagnostic interpretation of dashboard E2E: it
  creates a PR as an audit fixture and would need a PDF fixture. Distinguish
  test maintenance from dashboard feature logic. The purchase-order form and
  purchasing route page remain unsupported by this task requirement.
- [TODO] Obtain a frozen implementation diff or blind adjudication before
  treating the provisional file list as evaluation ground truth. The budget
  gateway case also lacks a specified budget value/source.
- [DONE] Make sequence-flow diff reasons describe actual changes to endpoints,
  condition, and label. The Staff label-only control now reports one
  `MODIFY_SEQUENCE_FLOW` with reason `sequence flow label changed`; its noisy
  VERIFY candidate list remains a retrieval issue. Analysis identity bumped
  to `engine-3.5g-flow-reasons`; no retrieval weight or cutoff changed.
- [DONE] Verify this refinement with `go test ./...`, `go vet ./...`,
  `gofmt -l` for touched Go files, and `git -c core.autocrlf=false diff --check`
  on 2026-10-27. All passed. Simple-ERP remained at `c934497` with the same
  232 dirty-status entries and baseline BPMN SHA-256; it was not modified.

## Sequence-flow label eligibility refinement (2026-10-27)

- [DONE] Add one method-independent direct-reference gate for a label-only
  sequence-flow edit with unchanged endpoints and a retained explicit
  condition. Contextual words alone cannot qualify a candidate; exact flow
  ID or sufficiently specific old/new label text can. No direct reference
  means unresolved, **not** automatic `NO_ACTION`. Configuration mapping is
  exempt because its explicit mapping is separate evidence. This differs from
  unrestricted lexical/semantic retrieval only for this narrow case; no BM25,
  semantic, or RRF weight changed. Engine identity is
  `engine-3.5h-flow-label-eligibility`.
- [DONE] Add synthetic tests for no reference, exact label, exact flow ID,
  deceptive ID prefix, changed condition, label-as-condition, endpoint edit,
  and all four public retrieval methods. Focused Go tests passed. The
  Simple-ERP label-only control remains an exposed diagnostic, not training
  data or research ground truth.
- [DONE] Rerun the read-only Staff label control with the same input fingerprint
  `f6095e7ad63be127073ee7be96cec810`: BM25 changed from five VERIFY
  candidates to zero with one explicit unresolved reason; Hybrid also returned
  zero/one. Add-document still produced one ADD_TASK with five top-K candidates;
  the budget-condition case still produced five flow candidates and an
  unresolved gateway-condition record at top-K 5. `go test ./...`,
  `go vet ./...`, changed-file `gofmt -l`, and
  `git -c core.autocrlf=false diff --check` passed. This is regression/control
  evidence, not a measured accuracy gain. Simple-ERP remained read-only at
  `c934497`, 232 dirty-status entries, and the same baseline BPMN SHA-256.
- [TODO] Evaluate the gate's missed true impacts and false positives on a
  frozen, independently labeled multi-repository benchmark before treating it
  as a calibrated precision improvement.

## Frozen synthetic control pack (2026-10-28)

- [DONE] Freeze a four-scenario matrix for task insertion, task removal with
  rewire, gateway-branch addition, and label-only negative control. Each BPMN
  input is SHA-256 locked; each scenario pins the same method-independent
  repository/BPMN fingerprint across BM25, Semantic, Fusion, and Hybrid.
  `TestFrozenSyntheticScenarioMatrixAcrossMethods` passed all 16 method-case
  combinations twice. The constant test embedder checks orchestration only,
  not semantic accuracy.
- [DONE] Add a separate executable four-method, two-repetition negative-control
  benchmark with fixture-authored empty expected impacts. It returned one
  correct abstention out of one negative case per method, with identical
  fingerprint and byte-reproducible predictions. The output is a synthetic
  regression result, not Simple-ERP Precision/Recall.
- [DONE] Separate correct negative-control abstentions from the existing
  unresolved/abstention rate in evaluation JSON, Markdown, benchmark JSON,
  Markdown, CSV, and the web evaluation table. Retain the existing
  unresolved-rate definition for compatibility; show negative-control
  accuracy as n/a when no labeled negative case exists. Focused
  evaluation/benchmark tests and web TypeScript check passed.
- [TODO] Obtain independent file-level truth for task/gateway scenarios before
  publishing accuracy metrics; the matrix currently checks taxonomy,
  determinism, and candidate presence only. The Simple-ERP source review is
  still provisional and must not enter retrieval.

Verification for this pack: `go test ./...`, `go vet ./...` with the
repository-local Go cache, changed-file `gofmt -l`, web `npm run typecheck`,
and `git -c core.autocrlf=false diff --check` passed. Simple-ERP remained at
`c934497` with 232 existing status entries and the same Staff PR-create BPMN
SHA-256; it was not modified. No commit or push was made.

## Single-review Simple-ERP diagnostic (2026-10-28)

- [DONE] Inspect current dirty Simple-ERP Staff source and BPMN read-only,
  recording a positive stock-delta rule and a negative PR flow-label case
  before opening this run's candidate rankings. Save complete case references
  outside the indexed repository with source evidence and explicit assumptions.
- [DONE] Freeze the two BPMN pairs, source input fingerprints, and
  reference hashes in a provisional two-scenario manifest. Mark its label
  provenance in benchmark outputs so single-review figures cannot be mistaken
  for independently adjudicated ground truth.
- [DONE] Run four methods twice on the frozen inputs, inspect errors, verify
  reproducibility, and report only diagnostic findings. No method weights are
  to be tuned from these labels.

The source-only review is in `docs/ground-truth/staff-source-review-2026-10-28.md`
with two separate truth JSON files; the benchmark manifest is
`experiments/simple-erp/staff-source-review-2026-10-28-manifest.json`. The
read-only study checkout was HEAD `c9344971a573d5c44d9d4e58f23657eda4d18a86`
plus 232 existing dirty entries. The manifest locks BPMN and truth SHA-256,
both input fingerprints, and `label_provenance: "single-source-review"`.
Benchmark ID `52224287eddd2285159b504cb07c5110` completed all eight
scenario-method cells with two byte-identical repetitions each. For the
inventory gateway's two provisionally required files, Hybrid ranked them 4
and 1; Semantic/Fusion 5 and 6; BM25 missed both on the gateway change. All
methods generated unwanted supporting-flow candidates but abstained for the
label-only negative control. The older source reference names only the store,
whereas this review also names the existing integration test; the disagreement
is retained for later adjudication. See the versioned diagnostic report in
`experiments/simple-erp/`. These observations are not independent TA ground
truth or a basis to recalibrate ranking weights.

Verification: `go test -p 1 ./...` passed after the provenance adapter change.
No Simple-ERP or POC TA file was modified.

## Synthetic positive evaluation controls (2026-10-28)

- [DONE] Author two independent synthetic change scenarios (task
  insertion and gateway branch) with source-owned expected file edits before
  inspecting candidate rankings. Keep expected labels outside indexed repos.
- [DONE] Freeze BPMN hashes and method-independent input fingerprints in a
  four-method benchmark manifest; include a negative control.
- [DONE] Run two repetitions per method, record deterministic metrics and
  latency separately, and document that fixture-authored truth is a functional
  control rather than independent Simple-ERP research evidence.

The new `testdata/benchmark-impact/manifest.json` covers one added user task,
one added exclusive gateway, and a display-label negative control. Its truth
was written from the fixture's intended source edits before candidate ranks
were inspected and is outside the indexed repository. All four methods ran
twice; the deterministic benchmark ID is
`a4b2c567147ed8ae70e778c8770cd54d`. Independent runs before and after the
coverage gate had identical `benchmark-report.json` SHA-256
`2d5c15b04ff8889248d7cfe7649cfaad7b037296eaf19ca01d73221c5a9035b9`.
On the principal task change, BM25 placed the two authored files at ranks 4
and 2; Semantic/Fusion/Hybrid at 3 and 2. For the principal gateway change,
all methods placed the approval function at rank 1; only Hybrid included the
approval test, at rank 6. None of the methods classified the authored MODIFY
actions correctly. Supporting flow changes include negative expected sets,
and the gateway generated extra flow candidates. Do not turn these fixture
observations into tuned weights or Simple-ERP accuracy claims.

The new batch coverage gate exposed two incomplete older fixture truths.
`synthetic-request-change` labels only 2/12 changes, so its manifest now
declares `truth_scope: "subset"`; subset metrics remain visible per scenario
but are excluded from method aggregates. The vocabulary fixture's three
task-insertion support flows now have explicit empty expected sets and an
updated truth hash. An undeclared partial truth fails closed. The default
remains complete coverage, and at least one complete scenario is mandatory.

Verification: `go test -p 1 ./...`, focused uncached benchmark/evaluation
tests, `go vet ./...`, changed-Go `gofmt -l`, and
`git -c core.safecrlf=false diff --check` passed. Benchmark outputs remain
ignored under `outputs/`. No Simple-ERP or POC TA source was changed.

## Planning provider contract hardening (2026-10-28)

- [DONE] Reject malformed/null/trailing completion content and model
  responses explicitly stopped by an output limit before planning validation.
- [DONE] Enforce no redirects even for injected HTTP clients; validate
  endpoint URL components and bound the complete serialized request payload.
- [DONE] Exercise failure paths with loopback HTTP mocks, run Go verification,
  and document the provider boundary. Production remains on NoLLMProvider.

Evidence: `internal/llm/openai_compatible_safety_test.go` covers malformed and
incomplete responses, redirect isolation, endpoint validation, cancellation,
serialized request limits, response limits, and error-body non-disclosure.
`go test ./...` passed (CLI suite 120.793 seconds); `go vet ./...`, changed-Go
`gofmt -l`, and `git -c core.safecrlf=false diff --check` passed. HTTP contract
tests use loopback mocks. This does not establish real-model planning quality
or improve retrieval accuracy; endpoint activation and Qwen evaluation remain
pending. No ranking weights, case-study source, or Git remotes were changed.

## Staff P2P total-confirmation pre-inference reference (2026-10-28)

- [DONE] Create a separate Staff L3 BPMN with one new user review/confirmation
  task after PR line input. It is UI-only and does not introduce attachment,
  backend validation, schema, or API changes. Preserve the original Staff
  baseline as read-only.
- [DONE] Write a source-derived existing-file reference JSON and rationale
  **before** running mapper inference for this scenario. Label the PR form and
  two existing E2E creation flows; assign no independent file label to the
  insertion's connecting sequence-flow edits. Hash the reference and BPMN.
- [DONE] Parse the new BPMN and verify the expected one `ADD_TASK` and two
  `MODIFY_SEQUENCE_FLOW` diff keys using only `testdata/repository` for a
  smoke run; no Simple-ERP candidate ranking was inspected. XML flow endpoints
  and DI elements are present. `git diff --check` passed.
- [DONE] The user ran the frozen v1 input in the web UI and supplied Hybrid
  report `31d5eab5168544c9a75ae2fd1f8ffd3d`. Read-only comparison with the
  prewritten reference found expected files at ranks 1, 2, and 6; the two
  supporting flow records had no candidates.
- [TODO] Freeze the entire dirty source working tree and obtain independent
  implementation/reviewer adjudication before claiming formal TA accuracy.

Follow-up after the user's first Hybrid report: the source-derived reference
matched three existing files at ranks 1, 2, and 6 (Recall@5 = 2/3; file
precision at 20 = 3/20), but the BPMN task documentation explicitly stated
backend/API/database non-impacts. Preserve that exposed v1 input and report;
do not use them as a blind accuracy claim. A v2 BPMN removes the architectural
hint while retaining the same user task and graph. Because the v1 ranking and
reference are already known, v2 is diagnostic, not a new held-out benchmark.

- [DONE] Run read-only v2 BM25 and Hybrid diagnostics against the same
  Simple-ERP corpus (198 indexed files, input fingerprint
  `a9dd4895d4e5d8617767d78eab614b52`). Hybrid's top ranks were unchanged:
  the two E2E files ranked 1 and 2 and the PR form ranked 6. The architectural
  sentence was not the cause of that ranking in this case. BM25 alone placed
  the E2E files at 16 and 18 and missed the PR form within 20. These are
  exposed diagnostic observations, not unbiased accuracy estimates.
- [DONE] Correct the web's misleading static "Five keeps the review focused"
  helper and use a 10-candidate default review cap so an important candidate
  just below five is not silently hidden. The existing Code/Tests/Docs view
  filters remain available; the exported ranking and RRF formula are
  unchanged. This improves visibility, **not** measured retrieval accuracy.
- [TODO] Do not add a test penalty, user-task frontend boost, or RRF weight
  from this one exposed scenario. Both top-ranked tests are genuine impacts
  under the source-derived reference. Collect independently adjudicated
  scenarios across different change types before calibrating ranking.

Verification for this follow-up: v2 XML parsed, the analyzer returned the
expected one task and two supporting flow edits, targeted Go package tests,
web `npm run typecheck`, and `git -c core.autocrlf=false diff --check` passed.
Simple-ERP remained read-only at `c934497` with 232 pre-existing status
entries and the unchanged old Staff BPMN SHA-256.

## Gateway retrieval and duplicate-flow correction (2026-10-28)

- [DONE] Preserve direct gateway branch conditions and labels in plain BM25,
  while keeping process/neighbour expansion exclusive to contextual retrieval.
  Include added and removed branches; do not introduce case-specific terms.
- [DONE] Group same-endpoint condition edits with their gateway change, retain
  atomic JSON records and independently map direct source references. Keep
  rewiring, added/removed flows and display-only edits independent.
- [DONE] Run regression tests and frozen synthetic/Staff diagnostics; keep
  existing labels, weights and prior outputs unchanged. Document both misses
  and improvements. No additional retrieval method or LLM activation planned.

Evidence so far: new `gateway_grouping_test.go`, `gateway_query_test.go`,
application direct-reference tests and existing public-method tests pass.
`go test -p 1 ./...`, `go vet ./...`, web `npm run typecheck`, and formatting
checks pass. An intermediate condition-only query dropped a useful branch
label and degraded Semantic; restore both label and expression as retrieval
input while keeping expression precedence for change detection. The failed
diagnostic is preserved under `outputs/staff-gateway-fix-2026-10-28/`.

Final evidence (closed 2026-10-29): Staff benchmark
`e9e6a54b78a74174efec3530afd436ea` has eight reproducible scenario-method
results over two repetitions. Synthetic benchmark
`73ff0e055d61de0e1c45795bb49c0bb0` has twelve reproducible results over two
repetitions, with unchanged primary expected-file ranks versus the previous
synthetic run. BM25's Staff gateway reference files move from absent to ranks
3/4; production Hybrid stays at 4/1. Fusion has mixed rank changes. Keep this
as exposed development evidence, not formal accuracy validation. Full details:
`experiments/simple-erp/GATEWAY-RETRIEVAL-FIX-2026-10-28.md`.

Follow-up browser verification: report-results and report-explanation
Playwright suites passed 8/8 (35.4 seconds), including fixture report filtering
and desktop/mobile file views. Sandbox process spawning initially failed with
EPERM; the identical local test command passed with process permission.

## Consolidated existing Staff experiments (2026-10-29)

### Follow-up file-ranking experiment

### Positive-case evaluation and passage audit (2026-10-29)

### Pre-presentation query audit and new Staff reference

### UI readability pass

### Saved benchmark summary in Research

### Development-only Research workspace

- [IN PROGRESS] Keep Research available during R&D with a build-time flag to
  hide the workspace and experimental method controls at launch. Normal
  mapping stays on Hybrid. This is UI configuration, not authentication.
- [TODO] Verify enabled/disabled modes and input preservation; document how
  to restart/rebuild and the distinction from an admin permission boundary.

- [DONE] Publish a dated, read-only snapshot of the completed Staff
  evaluation in Research, with metric definitions and a downloadable JSON.
  Identify source/references and distinguish it from current-repository or
  live benchmark results. Do not connect reference data to mapping inference.
- [DONE] Validate the snapshot against saved evaluation artifacts, check
  TypeScript, browser display/download/mobile overflow and no inference calls.

Evidence: published JSON cross-checked against all four methods' saved positive
evaluation reports; TypeScript and four Playwright batch/benchmark tests pass.
Screenshot review confirms the dated table, scope, definitions and download.
The test changes the current repo and verifies the snapshot stays identified
as Simple-ERP, downloads matching JSON and triggers no analysis POST. This is
a saved result viewer, not a live benchmark runner or universal accuracy score.

Research copy clarification: explain the one-pass scenario queue separately
from accuracy evaluation, state that the current web evaluation needs two
reviewer-reference JSONs, and describe expected files in plain language.
Accessible control labels and API contracts are preserved; the CLI benchmark
results are not claimed to be automatically imported by the web.

- [DONE] Simplify workspace copy and panel hierarchy, enlarge report
  text, balance configuration/results width, and improve keyboard/mobile
  readability without changing analysis inputs, ranking or canonical output.
- [DONE] Compare desktop/mobile screenshots and run existing mapping, report,
  evaluation and canvas regression checks appropriate to the changed surfaces.

Evidence: desktop/mobile report screenshots inspected; report grouping/filter
suite 3/3, isolated canvas draft/export/mapping 1/1, isolated evaluation UI
1/1, TypeScript and diff checks pass. Text hierarchy, panel widths, file upload
layout, filename wrapping, focus and reduced-motion styles were improved.
The combined 16-test run was interrupted after browser resource failures and
embedding OSError 1455 (Windows paging file too small). Targeted suites passed
without full-model inference. Do not claim the entire suite or live semantic
runtime passed in this resource-constrained session. No ranking/source changes.

- [DONE] Inspect current PR price query and saved ranking, document
  findings/hypotheses without modifying retrieval or demo inputs.
- [DONE] Prepare a new Staff cancellation-reason requirement with source-only
  reference before mapper inference; retain matching BPMN-DI and validate XML,
  mutation and expected diff. Leave this new case unscored for a future test.

Evidence: `experiments/simple-erp/QUERY-AUDIT-2026-10-29.md` records null
context for the price-label change, retained old/new rule text, lexical
language differences and saved branch ranks. Query reconstruction is labeled
static, not a captured runtime trace. No method or production input changed.
The fresh cancellation case has a three-file source-only reference prepared
before inference, frozen input/source hashes, and inherited DI. The mapper's
Go parser/diff confirmed exactly one label modification and a userTask; XML
checks confirmed identical DI and an exact baseline copy. No mapping was run
for this new case. Original Simple-ERP/POC TA remain read-only. This audit does
not constitute independent adjudication or browser rendering proof.

- [DONE] Add explicitly scoped positive-case MAP/MRR/R-precision and
  P/R@5/10, preserve legacy atomic-change metrics and empty-case abstention.
  Surface definitions and undefined values in reports/web; version evaluation IDs.
- [DONE] Audit missed code passages, test a query-independent bounded passage
  variant, retain it only if unchanged-reference ranking improves without recall
  regression. Record unsuccessful experiments too; no label/weight changes.

Evidence: `positive_ranking` in JSON/Markdown/benchmark CSV and Research UI;
hand-calculated R-precision/AP tests, negative-only undefined tests and
positive-count-weighted aggregation test pass. Evaluation IDs are versioned.
Five Staff cases completed twice with fingerprint/hash checks; the adjacent
sampler failed its gate (MAP .7389 -> .7327, P@5 .45 -> .40, R@5 .875 ->
.7917). Production wiring was restored to three passages; optional sampler
is default-off and retained for explicit experiments. Full Go tests and vet,
TypeScript, browser suite 8/8 plus R-precision test 1/1 pass. Final synthetic
benchmark `f2df7a31a5e8bbfc9718482d73054dd7` verifies restored production with
the new evaluator. See `ADJACENT-PASSAGES-2026-10-29.md` for results and limits.

- [DONE] Test file-level RRF inputs: deduplicate files before assigning
  branch ranks and combine file agreement without asserting function agreement.
  Keep atomic flow eligibility unchanged; record policy metadata. Compare
  unchanged Staff labels and synthetic controls before adopting this variant.

Adopted for production Hybrid after five unchanged Staff cases, two runs each:
P@5 0.40 -> 0.45, R@5 0.7917 -> 0.8750, MRR 0.7083 -> 0.8333, R@10 remains
1.0. Synthetic benchmark `e623fcc6194446583075003112afec83` reproduced all
12 scenario/method results; Hybrid expected-file ranks and the negative
control are unchanged. Full Go tests, vet, gofmt and diff checks passed.
No reference labels, weights or case-study source changed. See
`experiments/simple-erp/FILE-FUSION-2026-10-29.md` for the file-granularity
reasoning and the fixed-denominator precision ceiling.

- [DONE] Assemble the existing positive-price, submit-reason task,
  total-confirmation task, inventory gateway-condition and label-control pairs
  with their existing source-derived references. Freeze current input hashes;
  do not change labels based on predictions.
- [DONE] Run four methods, two repetitions and top ten on the same inputs;
  report primary-change metrics separately from atomic-flow macro metrics.
- [DONE] Record misses and false positives, scope limitations and next fixes.
  These exposed cases remain development diagnostics, not held-out validation.

Manifest: `experiments/simple-erp/staff-consolidated-2026-10-29-manifest.json`.
The initial full batch exceeded the CLI's default ten-minute total deadline;
preserve partial outputs and rerun unchanged inputs in a new output directory
with `--timeout 1h`. The post-inference `summarize-staff.ps1` grouping and
positive-change metrics were cross-checked against the prior frozen
gateway/control run (Hybrid Recall@5=1, MRR=1, one correct control abstention).

Completed benchmark `13a8e66389b53ac0b329e8f26fdddbe1`: 20 reproducible
scenario-method cells / 40 analyses. Primary positive-case Hybrid Recall@5
0.7917, Recall@10 1.0 and Precision@10 0.25. All four independent label-only
controls abstained; all 20 supporting-flow/method records had zero candidates.
Preserve the late PR form/store candidates and other methods' misses in the
diagnostic CSV; do not adjust reference labels. Results and next query/passage
audit are in `experiments/simple-erp/STAFF-CONSOLIDATED-2026-10-29.md`.
The original ten-minute timeout is documented as an execution-budget deviation;
the complete rerun uses identical inputs and an explicit one-hour CLI budget.

## Bounded presentation clarification: recorded BPMN context

- [DONE] Add a collapsible report section showing recorded process name,
  predecessor/successor labels, branch terms and related flow IDs, with explicit
  missing-context fallback. These are not a reconstructed exact query or scores.
- [DONE] Document XOR/AND/OR validation backlog in
  `experiments/simple-erp/PANDUAN-EMPAT-SKENARIO.md`; no new scenario or ranking
  change is claimed.
- [DONE] TypeScript typecheck and scoped `git diff --check` passed.
- [TODO] Browser visual verification of the context disclosure.

Scope: UI/documentation only; Simple-ERP remains unchanged. Query generation,
ranking and saved benchmark results are not changed by this clarification.

## Bounded inventory error audit

- [DONE] Inspect saved inventory report against the unchanged provisional
  reference; record expected-file positions and candidate-level error clues in
  `experiments/simple-erp/INVENTORY-ERROR-AUDIT.md`.
- [TODO] Test generic event-label context ablation with identical ranking/model
  settings. The audit alone does not justify changing defaults or references.

No inference, case-study mutation or ranking change was performed for this audit.

### Opt-in lexical context ablation

- [DONE] Add internal `ContextBM25.OmitStartEndLabels` with explicit metadata;
  it only omits neighbour start/end labels from a copied query context.
- [DONE] Tests verify default behaviour, preserved gateway text, unchanged
  recorded context, nil context, and retained changed-event meaning.
- [DONE] Retrieval/wiring/fusion tests, scoped vet, gofmt and diff check pass.
- [TODO] Evaluate on the fixed multi-case inputs before any default activation.

The option is not wired to CLI/web; production ranking is unchanged. Future
experiment wiring must include this setting in analysis/configuration identity.

## Context ablation and structural Staff scenarios

- [DONE] Add isolated offline study command and resolver variants with distinct
  configuration identities; production uses the existing resolver/defaults.
- [DONE] Run full/no-context/no-start-end/both-top10 on five frozen cases, twice:
  20 cells, 40 byte-reproducible predictions, common per-case fingerprints.
- [DONE] Validate prediction/evaluation hashes and aggregate four positive cases.
  Full MAP=.7389, R@10=1; no-start-end unchanged; both-top10 R@10=.7917,
  MAP=.6472. Retain production defaults. Broad no-context also removes changed
  branch terms; document that attribution limitation in CONTEXT-STUDY.md.
- [DONE] Build seven before/after pairs under structural-staff with complete DI,
  independent provisional union-file references and inspected source hashes.
  Include a business-change gateway removal and a separate notation-only control.
- [DONE] Go parser/diff/DI checks pass for all seven pairs. Six browser cases
  pass together, and the seventh added case passes separately for before/after
  element/edge import. No inference on these new cases yet.
- [DONE] Full `go test -timeout 90s ./...`, full `go vet ./...`, frontend
  typecheck and whitespace check pass. Six report-explanation logic tests pass.
- [TODO] Convert new union references to an explicitly scoped evaluation contract
  before scoring; verify role/AND/OR behaviour beyond XML/diff/import checks.

Simple-ERP and POC TA remain read-only (232 and 6 pre-existing status entries).
Saved benchmark table is unchanged. No Qwen, commit or push in this work.

## Staff modelling convention and combined gateways

- [DONE] Add a separate deterministic BPMN checker: structural warnings
  versus optional research conventions (one start/end per nonempty scope,
  meaningful activity after a join). Never reject valid BPMN for conventions.
- [DONE] Audit Staff read-only; prepare one source-reviewed single-end example
  in Mapper, without overwriting frozen experiment pairs or ERP files.
- [DONE] Test a combined XOR + AND + new-task change, including diagram coverage.
- [DONE] Verify checker, regression tests and documentation. No token simulation,
  automatic semantic rewrite, production ranking or benchmark-score changes.

Evidence: `experiments/simple-erp/staff-single-end/README.md` and audit JSON.
32 Staff BPMNs parse; 23 SINGLE_END and 5 ACTIVITY_AFTER_JOIN convention findings,
zero SINGLE_START findings. One normalized inventory baseline and compound mutation
pass the checker. Tests assert all four new gateways and five new tasks, rejection
paths not reaching successful state mutation, deterministic diff and empty self-diff.
Full Go tests and vet pass; CLI hash/error additions pass subsequent scoped tests
and vet; frontend typecheck and one browser import test (both diagrams) pass.
Initial browser launch hit sandbox EPERM; approved local browser rerun passed.
Whitespace checks pass. Case-study status counts remain 232 (ERP) and 6 (POC).

- [TODO] Review remaining Staff convention findings against their own abstraction
  level before normalization; no placeholder activity or mass rewrite.
- [TODO] Prepare independent scoped references and run retrieval evaluation for
  the compound case. Current checks prove parsing/diff/import, not mapping accuracy.

The convention checker is CLI-only. Original and frozen benchmark files remain
unchanged; new example files live exclusively in Mapper. No commit or push.

## Canonical Staff BPMN normalization (explicit user request)

The user now requests all Staff BPMNs fixed. This authorizes replacing only the
canonical Staff BPMN documents in Simple-ERP, superseding read-only for these
documents only. ERP application code, other roles and frozen Mapper experiments
remain untouched. Prepare reviewed files/backups inside Mapper before applying.

- [DONE] Normalize 28 of 32 Staff diagrams, preserving original decisions,
  conditions, roles, calledElement links and terminal outcomes.
- [DONE] Validate all 32 with convention checker and browser import; inspect
  representative diagrams, retain original layout except final merge/activity.
- [DONE] Apply verified replacements with original-hash checks and local backups;
  verify deployed hashes and document exact scope and experiment-baseline caveat.

Evidence: `docs/STAFF-BPMN-NORMALIZATION.md`; prepared and applied checks both
report 32 parsed / zero findings. Final browser run 32/32 passes; typecheck passes.
Preservation verifier covers business-node attributes, conditions, actors,
QName namespace bindings, original-node layout, complete DI and exit isolation.
28 canonical files replaced after approved out-of-workspace write; 32 applied
hashes match reviewed content. 91 outside-scope hashes remain unchanged.
Backup/manifest live under ignored `outputs/staff-normalization-2026-10-29-final`.

Two original baseline byte copies were preserved under frozen-staff-baselines,
and two experiment manifests plus the four-scenario guide now use those paths.
This necessary compatibility change preserves original input hashes/references;
no evaluation scores were recalculated. ERP generators were not changed or run;
their older catalogue can overwrite these canonical document edits if regenerated.

## Workspace cleanup and normalized Staff experiment continuation

- [DONE] Archive loose user uploads/images/reports and root binaries with
  a hash manifest; preserve all data, caches, source and frozen results.
- [DONE] Fix Staff document generator using an isolated copy and apply only
  reviewed generator support files to ERP; application source stays untouched.
- [DONE] Build new single-end Staff cases and source-derived provisional labels
  before prediction, then run four-method evaluation with explicit case scope.
- [DONE] Add one main README entry and experiment guide; verify changed paths,
  generator output, tests and artifact provenance. No commit/push or LLM activation.

Evidence: ten root files archived with post-move SHA checks; source/data/caches
preserved. Root has README, PLAN, Go/Git files and two primary research documents.
Generator isolated build creates 32 convention-clean Staff XML and matching SVG;
counts match manifest, normalizer is byte-idempotent. Two reviewed script files
applied to ERP; canonical BPMNs not regenerated. Ninety out-of-scope hashes and
all 32 Staff canonical hashes still match the previous state.

New current-staff package: four positive pairs and unchanged control, all ten
diagrams convention-clean and browser import tests 10/10 pass. Full Go tests,
vet, frontend typecheck and whitespace checks pass. Run b1142fe45846f7f21e3d279d420df29b:
20 method/case cells, 40 analyses, byte-reproducible. Four selected-anchor Hybrid
P@5 .55, R@5 1, MAP .8472; Fusion MAP .9028. These are partial source-reviewed
diagnostics, not full-scenario or held-out accuracy; complete controls excluded
from positive means. No scoring settings/references changed after inference.
Initial preflight hash mismatch was Windows text-newline output, corrected to
UTF-8 byte writing before inference. Final run complete in a fresh output folder.

Entry: README -> experiments/simple-erp/current-staff/README.md and RESULTS.md.
Outstanding full union/primary-change labelling remains explicit; no claim that
every change in the compound scenario has independently validated ground truth.

## Supervisor and Super Admin convention (explicit user request)

- [DONE] Prepare read-only review and backups for 31 Supervisor and
  49 Super Admin BPMNs; preserve actor lanes, outcomes, guards and calledElement.
- [DONE] Check parser/convention, preservation properties and browser import.
- [DONE] Extend document generator to both roles and validate isolated outputs.
- [DONE] Replace only verified canonical BPMNs and generator support; verify
  hashes and document next baseline traceability work. No source application edits.

Evidence: `docs/ROLE-BPMN-NORMALIZATION.md`. 29 Supervisor and 49 Super Admin
canonical documents replaced after hash checks/backups and 80/80 browser tests.
All 112 canonical role BPMNs now parse with zero convention/structure/review
findings. Role-specific result actors checked for all 78 new result activities;
preservation verifier passes on 31+49 applied files. 120 backend Go/Staff hashes
unchanged. Scoped Go tests and both repository whitespace checks pass. Isolated
generator creates 112 convention-clean, byte-idempotent XML outputs and parseable
SVG previews. Two generator scripts applied with original-byte preconditions.
No app source changes, score updates, new baseline mappings, commit or push.

## Baseline traceability: Staff P2P L3

- [DONE] Review ten normalized Staff P2P L3 processes against source only;
  map every activity with file/function/line evidence and distinguish context.
- [DONE] Freeze BPMN/source hashes and repository identity; build canonical JSON
  and render Markdown from that same object. Keep reference outside inference.
- [DONE] Verify complete activity coverage, evidence, hashes, deterministic
  regeneration and Markdown consistency; document static/provisional limits.

Evidence: `experiments/simple-erp/baseline-traceability/README.md` and generated
JSON/Markdown. Snapshot df5ed08f8e69212904abf9f865a2faee: ten processes / fifty
unique activities, 94 links / 169 source excerpts, 22 BPMN/source/test hashes.
Direct activity links separated from supporting API/service/helper/test context.
Eight related test links are static only; no ERP test or application executed.
Independent exact-line/hash/coverage checks and repeated deterministic --verify
pass. Markdown is rendered from the parsed canonical JSON object. Whitespace
check passes. No production ranking, UI, source ERP, commit or push changes.
Reference remains provisional single-AI source review, outside inference; no
automatic required-change ground truth or full gateway/branch coverage claimed.

## Pilot screenshot capability audit and pre-Qwen hardening

- [DONE] Audit screenshot features against actual parser/diff tests;
  distinguish syntactic support, semantic diff and measured file retrieval.
- [DONE] Surface partial-support diagnostics for unmodelled BPMN semantics in
  canonical reports; do not silently imply a complete analysis for these inputs.
- [DONE] Add synthetic coverage for cycles and collaboration/advanced features,
  verify diagnostics end to end and retain established ranking defaults.
- [DONE] Verify current baseline snapshot and document capability matrix, frozen
  notation decision and remaining full-reference/held-out evaluation work.

Evidence: `docs/BPMN-CAPABILITY-MATRIX.md`. Advanced-feature warnings now reach
canonical JSON/Markdown, CLI checker and web, with conditional policy identity.
Unmodelled multi-instance edits may still have zero detected changes but now
report explicit limitations; no invented impact change or complete-support claim.
Cyclic XOR condition JSON text edit, compensation flag, multi-pool/message flow,
call references and event feature diagnostics covered by synthetic tests.
Full Go test/vet passed before final compensation flag; scoped tests/vet pass
after it. Frontend typecheck and browser partial-result test pass; first browser
attempt failed due to ambiguous status selector, then fixed selector and passed.
Screenshot visually inspected. Baseline traceability --verify and whitespace
checks pass. No ranking weights, model, labels, benchmark scores or ERP files changed.

Remaining pre-Qwen work stays open: label all primary compound changes and
evaluate additional unseen business scenarios. The capability audit does not
replace independent reference validation or prove token execution semantics.

## Compound case: all changed primary nodes

- [DONE] Expand provisional source-derived labels to all seven new
  task/gateway nodes, with explicit rationale and source/BPMN hash binding.
- [DONE] Evaluate unchanged saved predictions against frozen expanded labels;
  enforce all-node coverage and report union file metrics without double counting.
- [DONE] Test scope/deduplication, document exposed retrospective status and
  weaknesses. Preserve historical single-anchor references and ranking settings.

Evidence: `current-staff/combined/NODE-EVALUATION.md`, new node-reference JSON
and rationale; seven node labels cover fifteen node/file pairs, three unique files.
Eleven flow changes are excluded, never labelled negative. Offline node-eval CLI
verifies source/BPMN/reference hashes, original manifest and four saved prediction
hashes/analysis IDs/fingerprints before evaluation. No new inference. Two evaluator
runs produce eight identical artifacts. Scope/deduplication test rejects incomplete
node labels and excludes flow-only candidates. Full Go tests, vet and diff check
pass; CLI positive-reference guard added afterward and scoped checks pass.

Hybrid per-node P@5 .3714, R@5 .8571, MAP .6468. Top-5-per-node file union returns
13 files / 3 expected, precision .2308 and recall 1. Semantic MAP .6603 exceeds
Hybrid here. Hybrid store rank 7 at MasterChecksJoin and CombineValidation exposes
a context weakness; no thresholds, labels or ranking policies adjusted to fit it.
Status remains retrospective exposed development analysis, not independent ground
truth or complete atomic BPMN coverage. Historical single-anchor results retained.

## Opt-in bounded baseline context experiment

- [DONE] For added-node clusters, find nearest existing task labels within
  four directed hops, same process/subprocess, visited-set/caps; record paths.
- [DONE] Use anchors in experimental Hybrid queries only; preserve immediate
  neighbours, source evidence, defaults, and distinct configuration identity.
- [DONE] Verify cycles/scope/bounds/determinism and compare full vs enriched on
  compound, AND, single-task and unchanged cases using frozen labels/repeats.
- [DONE] Audit results and keep production unchanged unless evidence supports
  adoption. No threshold fitting, model switch or label edits after inference.

Evidence: `experiments/simple-erp/BASELINE-CONTEXT-STUDY.md`. 8 cells / 16 analyses
reproduce prediction/evaluation bytes. All four Full predictions match historical
Hybrid exactly. Same input fingerprint/corpus/taxonomy, distinct option identity;
labels and flow paths validated. AND R@5 .8333 -> .9167 / MAP .5879 -> .8155;
compound R@5 .8571 -> .9286 / MAP .6468 -> .7664. Store ranks at join 7 -> 1,
aggregation 7 -> 2 (AND) / 3 (compound). One-task/control unchanged.

Not adopted: expected test file at CheckWarehouse drops 5 -> 7 in both cases,
and only exposed related development scenarios are available. Default remains
one-hop query behaviour; no hidden score/reference tuning. Anchors are query-only
labels with auditable direction/hops/paths; no invented source dependency or ID
contribution. Named-task/gateway controls are opt-in with zero production hops.

Initial parallel full Go test hit compiler OOM during embedding; stopped only
that test session, waited for study completion and reran sequentially with
GOMAXPROCS=2/GOFLAGS=-p=1. Full Go tests pass; scoped tests passed before study.
Final vet/whitespace/baseline snapshot verification are recorded below.

Final `go vet -p 1 ./...`, `git diff --check`, and baseline traceability
`--verify` pass. No frontend/source ERP edits or publication in this experiment.

## P2P transfer check and candidate-selection diagnostic

- [DONE] Freeze new PR-submit clustered checks, source-derived labels and
  complete DI before inference; lock multi-hop4, no tuning on this new case.
- [DONE] Compare full/multi-hop4 twice on P2P and unchanged control.
- [DONE] Implement immutable agreement-based selection as an offline option;
  calibrate only on prior development cases, preserving their top-10 recall.
- [DONE] Evaluate the frozen selection on new P2P; report abstentions, deferred
  files, recall loss and actual selected-file precision. Do not call RRF confidence.
- [DONE] Verify tests/hashes/reproducibility, document adoption or rejection;
  keep production unchanged unless the evidence justifies a concrete policy.

Evidence: `experiments/simple-erp/p2p-transfer/README.md`; 8 P2P/control analyses
reproduce. R@5 .9167 -> 1, MAP .9127 -> .9444. Development-only calibration
rejects filtering for full; multi-hop4 both-ranks<=20 preserves 30 development
hits with 140 -> 99 node/file pairs. Locked-policy P2P transfer preserves 12/12
hits with 60 -> 51 pairs (actual precision .2 -> .2353, no abstained nodes).
Candidate counts vary 7-10; raw/deferred candidates and original actions remain.
Two validation runs/policy bytes verified; provisional labels, not independent
ground truth. All Go tests/vet, TypeScript, four browser imports and BPMN checks
pass. Production context and web selection unchanged; wider validation remains
TODO before adoption. No Qwen, case-study writes, commit or push.

## Frozen selection challenges: removal and condition changes

- [DONE] Freeze Copy PR source-derived removal/condition pairs and labels
  before inference; disclose baseline task refinement and provisional truth.
- [DONE] Run full/multi-hop4 twice and apply existing frozen selection policy;
  report misses, lost hits, actual denominator and unchanged control.
- [DONE] Audit false-positive evidence against source without relabelling to fit
  predictions. Implement general retrieval fixes only if justified by evidence.
- [DONE] Verify artifacts/tests/diagram imports and decide whether selection is
  safe to expose in the web. No source ERP execution/modification or LLM calls.

Evidence: `experiments/simple-erp/selection-challenges/README.md`; 12 reproducible
analyses. Removal R@5=1/MAP=.7; condition R@5=.6667/MAP=.5 with required frontend
outside top10. Context variants identical as no added cluster exists. Frozen
selection changes 30 -> 27 node/file pairs, keeps 6/7 expected pairs, actual
precision .2 -> .2222; no selection-caused misses. Policy and validation repeats
identical. Auditor distinguishes raw misses, filter loss and provisional extras;
three unit tests reject inconsistent partitions and exercise abstention/loss.
Six fixture checker/browser tests pass; five relevant Go package tests pass.
Five reviewed source hashes and original ERP BPMN remain unchanged.

No production retrieval fix adopted: evidence does not establish a general safe
filter; docs/schema/similar workflows can be relevant but not mandatory code
changes. Frontend copy guard is in an encoded tail passage, not an indexing gap.
Keep web/default unchanged. TODO: diagnose cross-layer rank loss before further
retrieval experiments; independent labels remain unavailable.

Wider-pool diagnostic: required frontend appears rank17 (BM25 15 / semantic25)
with exact Copy button guard evidence. Original top10 prefix/source coverage/
change context match; diagnostic revision label differs and input fingerprint
is therefore not treated as identical. Both-ranks<=20 would defer this valid
file on a larger pool. No top50 result substitutes the frozen top10 metrics.
Binding JSON records both report hashes and the limitation. Reject web rollout.

Local resource incident: initial browser startup exited 3221225725; retry after
embedding completed passed 6/6. A concurrent Go repeat process failed allocation;
rerunning after browser exit passed and both validation runs match. No OS memory
settings or unrelated user processes changed. Further checks run sequentially.

## Gateway-condition query focus experiment

- [DONE] Compute per-flow lexical old/new condition differences from
  parsed graphs, only behind an offline option. Retain original full context.
- [DONE] Query gateway label plus changed condition words; fallback to full
  context for operator-only/reordered expressions with no lexical difference.
- [DONE] Compare frozen Copy PR challenges twice without label edits; gate wider
  accuracy runs on improvement. Keep RRF/model/index/defaults unchanged.
- [DONE] Verify immutability/default identity, scope/operator fallback, tests,
  reproducibility and adoption decision before touching production wiring.

Condition-focus initial result rejects adoption: removal MAP .7 -> .85 but
condition-extension MAP .5 -> .3333; required frontend remains outside top10.
All 12 analyses reproduce; full baseline bytes match the previous challenge
study. Do not cherry-pick the improved node or activate this query in production.

## Generic text-window index experiment

- [DONE] Add opt-in generic text windows (full file + max eight contiguous
  windows, minimum 1200 runes); no AST claims, named functions or source rules.
- [DONE] Evaluate full vs generic-windows on frozen challenges twice; existing
  queries/model/RRF/top10 unchanged. Track corpus/index identity change explicitly.
- [DONE] Review adoption gate before expanding accuracy runs. Target challenge
  fails to improve; reject rollout and preserve defaults. Wider runs deferred.

Condition-focus broader accuracy runs cancelled because the prerequisite
challenge already regresses; no benefit to claiming a passing rollout gate.
Generic-window broader accuracy runs likewise not justified by regressed R@5.
Core/default regression remains covered by repeated full baseline byte checks
and full Go gates below. Code-only scope control is completed below.

First generic-window case regresses test-file ranks 5 -> 7, while documentation
windows occupy more leading positions. Before seeing the condition-extension
result, define a code-only scope control: JavaScript/TypeScript/Java/Python/C#
text windows, existing AST Go and document/schema/config adapters unchanged.
Unknown languages retain generic fallback, never an AST claim. This addresses
index granularity rather than adding file/layer ranking weights. Test on the
same frozen condition/control first; adopt nothing if coverage still fails.

Final full test exposed an unrelated test-boundary defect: CLI benchmark unit
test loaded real Python/MiniLM and failed MemoryError. Add a typed analyzer
injection boundary for that CLI command; default unit benchmark exercises real
orchestration with explicitly marked constant test vectors. Preserve a separate
opt-in real-model integration test; never treat fake-vector metrics as quality
evidence. The 32 actual study analyses above use real local MiniLM unchanged.

Closed evidence: 12 condition-focus + 12 generic-window + 8 code-window analyses
reproduce prediction/evaluation bytes. Code-only window result equals full
R@5=.6667/MAP=.5; required frontend still absent top10. Eight full baseline
cells byte-match historical reports; source/BPMN/reference hashes unchanged.
All three experimental variants rejected for production. No new public method,
weights, top-k, source-study changes or Qwen integration.

Final `go test -p 1 -timeout 120s ./...`, `go vet -p 1 ./...`, gofmt and
`git diff --check` pass after fixing the CLI unit/integration dependency split.
New tests cover query immutability/default identity, operator/boolean/structural
fallback, Unicode/window coverage/bounds, language scope and route evidence.
No UI edits or browser rerun needed this step. Real-model opt-in integration
test remains resource-dependent; its initial MemoryError is recorded, not
reported as a passing real-model benchmark. No commit/push.

Next research task: investigate precise source-call/route evidence for missing
frontend callers, especially dynamic URL patterns, with frozen evaluation and
conservative ambiguity handling. No graph database or additional ranking weights
approved/adopted by this experiment.

## Report refresh recovery and current research results

- [DONE] Restore the selected saved analysis from its URL after refresh;
  fetch canonical server data, cancel stale restoration, never rerun inference.
- [DONE] Publish the frozen Oct30 study metrics separately from Oct29 four-method
  snapshot; bind source hashes and label them exposed provisional experiments.
- [DONE] Test refresh/no-POST, missing report, restore/open race and Research data;
  verify frontend types/browser behavior and document persistence boundaries.

Evidence: typed frontend check and five Playwright browser tests pass. New
BM25 fixture report is created through UI, reopened after reload by GET and
matches backend canonical JSON; zero inference POST after reload. Missing ID
has recovery guidance; slow old restore cannot replace a newly opened report.
Desktop restored-report and mobile research screenshots inspected. URL stores
only selected ID; BPMN uploads/drafts/comparison tabs/reviewer inputs remain
session-only as documented. Backend persistence was already present, unchanged.

Research Oct30 snapshot publishes seven per-case rows from 32 real study runs
with source hashes and no adopted variants, separately from the Oct29 four-method
table. Publisher verifies artifact repeats, input hashes and evaluation metrics.
Download/mobile containment tests pass; new user analyses do not auto-calculate
precision/recall without supplied references. No ERP edits, backend behavior
changes, dependency downloads, commit or push. Whitespace/type checks pass.

## Pre-model source-caller context and handoff

- [DONE] Recognize bounded JS/TS URL templates against unique Go route
  patterns, with explicit pattern-relation provenance and ambiguity abstention.
- [DONE] Trace unique two-word function-name seeds through at most three reverse
  static call/route edges to production clients. No ranking or action promotion.
- [DONE] Store at most three related source files per primary change, retain
  relative path/line evidence and safe snippets; render JSON/Markdown/web.
- [DONE] Include related evidence in bounded planning whitelist with no invented
  retrieval score; retain disabled provider/no network and evidence validation.
- [DONE] Verify synthetic ambiguity/negative cases, frozen ERP case/repeats and
  unchanged ranked order/scores; finish Go/frontend gates and readiness report.

Scope: source association is not proof of a required change. No rank-boost
heuristic, new public retrieval method, graph DB, model injection or ERP writes.

Evidence: `docs/PRE-LLM-READINESS.md`; final production case/control ran twice
with real local MiniLM. Prediction/evaluation/context bytes reproduce. Ranked
impacts equal historical baseline exactly; the previously missing frontend now
enters the planning whitelist via three source edges, constructor binding and
actual REJECTED button guard. All three provisional reference files are in the
package; this is context coverage, not 100% retrieval accuracy. Source hashes
remain unchanged. Initial source study predates directed branch v4; kept separate.

Planning receives explicit before/after fields and old/new branch conditions,
bounded warnings and truncation flags. Reviewer approval cannot override partial
or incomplete analysis. Offline `planning-context` export is deterministic and
refuses overwrite. Markdown now recognizes current context-BM25 versions rather
than printing obsolete weight wording. No model was injected or source modified.

All Go tests (including architecture boundaries), vet, frontend types and diff
check pass. Three refresh/history browser cases pass; the new real-API source
connection/planning browser test passes after fixing test-path assertions.
Application enrichment depends on an injected resolver; outer source adapters
are wired at the composition root. No commit/push or new dependencies.
## Baseline traceability: three modules and three roles (2026-10-30)

Scope: extend the source-reviewed Staff P2P reference to P2P, Inventory and
Master Data/Configuration for Staff, Supervisor and Super Admin. Simple-ERP is
read-only. Detailed activity mappings target all 72 current L3 processes; L0-L2
are indexed separately as decomposition/navigation references, not fabricated
direct code mappings. Existing Staff P2P snapshot remains intact.

- [DONE] Inspect current BPMNs, backend operations, frontend controls and
  role guards; record direct links separately from supporting call chains.
- [DONE] Publish nine JSON/Markdown views, source/BPMN hashes and a coverage index.
- [DONE] Verify complete L3 activity coverage, evidence anchors, determinism,
  drift detection and existing Staff P2P reference compatibility.

References are provisional static source review, not executed proof of 1:1
behavior, scenario required-change labels or prediction inputs. Any unsupported
activity/control semantics must remain explicit rather than being guessed.
No retrieval method, application behavior, model runtime or public UI changes.

Evidence: `experiments/simple-erp/baseline-traceability/all-roles/README.md`;
20 artifacts cover 72 L3 processes / 354 activities across all nine cells, plus
40 hierarchy diagrams. SHA-256 binds 112 BPMNs and 34 source/test files. All
nine integrity tests pass (including function bounds, ambiguous functions,
symlink/path escape, dense JSX excerpts, corruption detection and JSON/render
determinism). Rebuild `--verify` passes for both the new package and the original
Staff P2P reference. No ERP tests/source executed or modified. L0-L2 navigation
remains explicitly distinct from direct code mappings; no scope deviation.
## Research freeze before model integration (2026-10-30)

- [DONE] Bind current ERP text files, Mapper policy, BPMNs and labels to hashes.
- [DONE] Register exposed cases as development/regression; independent test set
  remains empty until new requirements are prepared before inference.
- [DONE] Audit scenario labels from requirements/source, without predictions.
- [DONE] Verify repeatability, drift rejection and integrity tests.

Research tooling only; no ERP edits/execution, model calls, label retuning or
new accuracy claims. Freeze is a binding, not an archive/runtime environment.

Evidence: `experiments/simple-erp/research-freeze/README.md` and `freeze.json`.
Ten cases across current-staff, p2p-transfer and selection-challenges retain
their original label sets and subset scopes. Per-file requirement rationale,
source evidence and reusable contracts are recorded without reading reports.
320 eligible Git-listed ERP text files and 168 Mapper/review policy files are
hash-bound; snapshot inventory is explicitly not production ingestion coverage.
Five new integrity tests pass; two full freeze verifications reproduce the
same ID. All nine baseline traceability tests and baseline snapshot verification
also pass. Independent tests remain absent, not relabelled from exposed cases.
No changes to public methods, UI, source ERP, model runtime or historical results.
## Multi-agent backend scaffold (2026-10-30)

- [DONE] Add analyst/planner/reviewer contracts, tool-driven turns and
  bounded revision transitions alongside the existing planning orchestrator.
- [DONE] Add read-only tools over the approved ingestion/index snapshot; source
  discoveries must have real file/function/evidence before entering the bundle.
- [DONE] Persist resumable sessions, identities, budgets and structured events.
- [DONE] Verify with scripted models: discovery, revision, cancellation/resume,
  fabricated evidence, policy/source drift and filesystem persistence.

No network, new dependency/framework, ERP execution/edit, shell tool or public
UI/API activation. Existing disabled-provider behavior stays intact. Eino can
later adapt the model port; a framework is not required for these contracts.
Research freeze remains historical: this development changes policy code and
must be recorded as drift rather than silently regenerating its frozen hashes.

Evidence: docs/MULTI-AGENT-BACKEND.md; targeted planning/agenttools/filestore and
architecture tests pass, including real disk restart and source-tool discovery
with a scripted model. No HTTP/UI activation; live lab tests are recorded below.
All application package tests and vet pass via ./cmd/... ./internal/.... Root
./... discovery hits an inaccessible old temporary directory under outputs;
this environmental limitation is not reported as a passing root command.
## Ollama lab adapter (2026-10-30)

- [DONE] Add opt-in loopback Ollama AgentModel adapter with model-digest
  binding, cloud-model rejection, bounded JSON actions and cancellation.
- [DONE] Add CLI runner over existing planning context and read-only repo tools,
  checkpoint/resume and canonical plan export; production web remains disabled.
- [DONE] Verify protocol/safety with httptest and existing scripted agent tests.
- [DONE] Finish all-three-role synthetic live smoke after user restored tunnel.
  Earlier failures preserved. Schema binds exact evidence IDs, current candidate
  counts and valid revision targets; role instructions align action constraints.

Authorized endpoint: user-established localhost:11435 tunnel to lab Ollama.
Selected model qwen3.5:9b-q8_0. First live input must be synthetic, not ERP
source or evaluation labels. No model download, server changes or cloud calls.

Evidence: docs/OLLAMA-LAB.md; failed live attempts retained under
outputs/lab-agent-smoke-*. Native adapter uses schema-constrained JSON actions,
not Ollama native function calling. No ERP input sent during live tests. Normal
server/web still uses disabled provider. New context-export limits are optional
and preserve existing defaults. No new accuracy claim or historical freeze edits.
Verification: all application tests and vet pass via ./cmd/... ./internal/...
with live lab tests disabled by default. Latest adapter changes also pass the
targeted llm/planning/CLI tests and vet. Live NEEDS_REVIEW run: 10 turns, 2 tools,
2 revisions, 37.07s. Latest COMPLETE smoke: 3 roles/turns, 8.26s; artifacts at
outputs/lab-agent-smoke-20261030T120107. Real CLI exports same canonical plan,
and completed-session replay keeps turn count unchanged. No claim of plan quality:
the final draft is still mostly verification prose, not a sufficiently detailed
change plan. Further planning-quality work and ERP validation precede web wiring.
## Web Qwen integration (2026-10-30)

- [DONE] Add opt-in asynchronous agent jobs, polling, resume and model
  readiness API. One worker, server-owned lifecycle, canonical saved plans.
- [DONE] Add Planning UI for selecting one BPMN change and repository scope,
  showing decisions, progress, review issues, per-file plans and downloads.
- [DONE] Verify API/worker cancellation, persistence, disabled mode and browser
  behavior; run a synthetic live path if the tunnel remains available.

Scope decision: Qwen reviews one explicitly selected change per session, with a
small evidence subset plus read-only source tools. Never imply the whole report
was reviewed. Initial Hybrid ranking remains immutable. No silent source sending
during mapping: user starts review explicitly. No credentials/model URL in browser.
No schema/database/framework additions or ERP writes. Model network errors leave
checkpoint/retry visible; repeated requests cannot launch concurrent workers.

Evidence: docs/OLLAMA-LAB.md. Real Qwen HTTP synthetic run at
outputs/lab-web-smoke-20261030T122959 passed, final NEEDS_REVIEW after 11 turns,
2 tools, 2 revisions (40.85s). Browser Qwen polling/refresh and existing offline
snapshot/demo tests passed. Worker tests cover concurrent-job rejection,
request cancellation, resume, restart and completed-session no-replay. Windows
checkpoint replacement permission errors are retried within one second without
deleting/truncating the previous checkpoint. At this checkpoint no ERP model
experiment had been performed.

## ERP Qwen planner repair (2026-10-30)

- [DONE] Reproduce the inventory planner validation failure: duplicate evidence
  IDs despite schema uniqueItems. Deduplicate identical adapter references while
  preserving unknown-ID rejection and strict engine validation.
- [DONE] Expose the actual validation issue in session feedback.
- [DONE] Add bounded, immutable read_source windows for generic source, long-line
  query matching and paging. Support qualified Go methods in source tools.
- [DONE] Require candidate source inspection through the Ollama protocol within
  existing budgets, and clarify the proposed-change planning contract.
- [DONE] Verify targeted regressions and application package tests/vet. Replay
  the real selected inventory change without sending evaluation labels.
- [TODO] Independently assess the generated draft and subsequent scenario runs;
  NEEDS_REVIEW is not an accuracy or plan-quality pass.

Evidence and limitations: docs/QWEN-SOURCE-REVIEW-FIX.md. No case-study source
changes, dependencies, cloud model calls, source execution or ranking changes.

## Offline agent readiness (2026-11-01)

- [DONE] Separate role-specific prompts and retain source observations
  across revisions to prevent contradictory actions and repeated reads.
- [DONE] Require proposed changes and acceptance checks in agent plans; allow
  one bounded output-contract repair without accepting invented evidence.
- [DONE] Exercise the real Ollama adapter, source tools, persistence and plan
  export against an offline HTTP model double, including failure paths.
- [DONE] Surface the structured plan in the existing Planning view and publish
  one readiness command/report. Live model quality remains a separate gate.

Scope: prepare the integration without requiring a lab connection, new source
scenarios, dependencies, source edits or model-specific ERP rules. Offline
protocol success cannot prove that Qwen will produce a correct plan.

Evidence: docs/AGENT-INTEGRATION-READINESS.md; seven local HTTP contract cases,
role/revision/repair tests, full application-package tests/vet, frontend typecheck
and Qwen browser refresh/export test passed. Script verification produced
outputs/agent-readiness/result.json with passed=true. No live lab calls made.

## Uploaded Qwen plan review (2026-11-02)

- [DONE] Audit the uploaded inventory plan against read-only source.
- [DONE] Replace boilerplate-only initial source windows with bounded
  requirement-focused source inspection, including compressed frontend lines.
- [DONE] Verify offline and document the plan's remaining semantic errors.

Do not rewrite the uploaded model output or tune retrieval scores to this case.
No lab connection, ERP writes or new research scenario is required.

Evidence: docs/QWEN-PLAN-REVIEW-2026-11-02.md. Application package tests/vet,
deterministic span/path tests and offline HTTP agent pipeline pass. Read-only
source inspection confirms both existing reason input and action payload now
fit the bounded evidence package. Actual Qwen plan-quality retest is pending VPN.

## Independent agent search and single-case evaluation (2026-11-02)

- [DONE] Audit stored session lookup history: zero repository searches, eight
  seed-file reads. Do not describe this historical run as exhaustive repo review.
- [DONE] Require a repository search in the Ollama analyst contract before final
  decisions; add distinct-file paging and keep bounded discovery rules visible.
- [DONE] Evaluate the selected ADD_TASK against the provisional hidden reference
  after inference. Both Hybrid top 3 and Qwen's unchanged three-file set score
  P=1/R=1; no equal-budget improvement is demonstrated.
- [DONE] Verify offline source paging and actual HTTP agent search/inspect flow.
- [TODO] Run a fresh lab session and compare independent discoveries, file
  decisions and plan quality. Do not feed this evaluation back into inference.

Evidence: docs/QWEN-CASE-EVALUATION-2026-11-02.md and
outputs/agent-case-evaluation/result.json. No lab inference or ERP source edits.

## Full repository review (2026-11-02)

- [DONE] Add an explicit full mode with deterministic, overlapping source
  chunks covering every ingestion-approved file, not only retrieval candidates.
- [DONE] Persist per-chunk decisions and scan progress; resume against the same
  source/model/change identity without re-sending completed chunks.
- [DONE] Plan/review all non-rejected files in bounded batches and aggregate the
  canonical plan; never silently truncate the finding list to top-k.
- [DONE] Add API/UI mode selection and eligible-file/chunk coverage, with offline
  tests for complete scanning, interruption, resume and source changes.

Scope: one selected BPMN change per full review. Excluded files remain excluded;
100% coverage means valid model decisions for every eligible source chunk, not
100% correctness. No global job deadline by default in full mode; per-call bounds
and server cancellation remain. Lab inference is not required for development.

Evidence: docs/FULL-REPOSITORY-REVIEW.md. Application-package Go tests/vet,
typecheck, actual-adapter offline full/resume API tests and Chrome Planning test
passed. Snapshot-only audit of Simple-ERP: 198 eligible files, 276 chunks, zero
inference calls. Full lab run and accuracy/plan-quality comparison remain pending
VPN. Checkpoints bind source/model/context; full scan is not an accuracy claim.

## Workspace UX refinement (2026-11-02)

- [DONE] Keep compact top navigation and integrate Mapping/Planning as
  stages of one analysis workspace. Preserve Canvas width and existing sessions.
- [DONE] Make the desktop input panel sticky with its own bounded scroll.
- [DONE] Carry repository paths to Planning and restore them per analysis in
  private browser storage, without adding absolute paths to reports or URLs.
- [DONE] Collapse technical context, refine spacing/type hierarchy, verify real
  desktop/mobile UI and mapping-to-plan flow offline. No model/retrieval changes.

Design reference consulted: tasteskill.dev (audit, brief-driven direction and
browser review). No skill or dependency installation; SF/system font stack and
existing CSS Modules remain. A research link may stay feature-flagged; Planning
is no longer a separate destination requiring re-entry of shared input.

Evidence: docs/WORKSPACE-UX-2026-11-02.md; eight Chrome regression tests passed
(2.2 minutes), including desktop sticky panel, auto-path and refresh, mobile
Canvas, evidence downloads, saved report recovery and caller context. Typecheck
and diff check pass. Existing source edits/worktree state preserved; no lab call,
dependency installation, method changes or case-study writes.

## Encoding and Canvas clarity (2026-11-02)

- [DONE] Repair static mojibake in UI text and add an encoding guard.
- [DONE] Separate Before/After into spaced diagram cards while preserving
  zoom/pan, markers, baseline immutability and expanded editor behavior.
- [DONE] Clarify Research batch experiments versus multi-process BPMN import.
  Keep production analysis at one before/after pair; bulk bundle UI is a proposal.

Verification: UI encoding guard and TypeScript passed. Targeted Chrome suite
(editor.spec.ts, batch.spec.ts, workspace-flow.spec.ts) passed; last-run status
is passed with no failed tests. The Canvas screenshot confirms two spaced cards.
No retrieval, scoring, BPMN input or case-study source was changed in this task.

## Local BPMN library (2026-11-02)

- [DONE] Audit canonical role BPMN read-only against the requested single
  start/end and no direct gateway-to-gateway/end conventions. These are local
  research conventions, not universal BPMN validity rules. Record exact findings.
- [DONE] Persist uploaded BPMN and repository/process/role/level/version
  metadata in the backend data directory, through a store port and file adapter.
- [DONE] Library UI: filter, inspect checks, download, and explicitly load a saved
  diagram as baseline or changed version, preserving original XML.
- [DONE] Offline persistence/API/browser verification and documentation.

Scope: one versioned diagram per entry; no auto-import of case-study files, no
source changes, no LLM calls, no automatic cross-level propagation. Relationships
and bulk multi-process analysis remain follow-up work.

Evidence: docs/BPMN-LIBRARY.md and docs/ROLE-BPMN-AUDIT-2026-11-02.md.
112 role diagrams audited; all hashes unchanged on repeat read. The stricter
gateway-to-gateway prohibition has 158 findings in 87 diagrams, so the old
normalization pass is explicitly qualified. No case-study normalization done.
Go application tests/vet passed; library API/store tests passed; Chrome library
persistence/Canvas/mobile and existing workspace flow passed. Initial browser
fixture had no diagram coordinates; switched to the existing DI-enabled fixture
and exposed a missing-layout warning rather than modifying XML automatically.
Typecheck, UTF-8 guard and diff checks pass. No lab inference or download.

## Reviewed canonical BPMN gateway cleanup (2026-11-02)

User explicitly authorizes BPMN document changes in Simple-ERP; application source
remains read-only. Prepare all edits/backups inside Mapper before bounded apply.
- [DONE] Prepare removal of redundant XOR result merges where the existing result
  activity can receive alternative incoming flows; preserve conditions/actors.
- [DONE] Prepare Staff AP: expose inspection of existing finance result/status
  before the end; no new mutation, approval or API request.
- [DONE] Verify graph equivalence modulo removed merges and the explicit AP
  observation, XML/DI integrity, idempotence and browser import.
- [DONE] Apply only reviewed canonical BPMNs after hash preflight; verify code
  hashes unchanged and publish correction to the earlier audit classification.

Prepared results: 87 changes, 86 redundant XOR result merges removed and one
Staff AP UI observation task added. Strict audit: 112/112, zero findings. Chrome
import/shape/edge tests: 112 passed (3.4m). Graph equivalence verifier passed
for all diagrams; no source or canonical BPMN changed. 179 source hashes and
ERP git status unchanged. Automatic approval review rejected canonical apply
because the previous read-only boundary and this specific 87-file merge-removal
strategy need explicit user approval. Do not retry/bypass before clarification.
Backups/prepared files and manifest: outputs/gateway-activity-2026-11-02/.

Canonical apply subsequently approved explicitly by the user and completed.
All 112 canonical hashes match prepared versions; 87 BPMNs changed. Re-audit of
actual Simple-ERP role files: zero strict convention findings. All 179 captured
source hashes unchanged. Earlier BLOCKED note is historical and resolved.
Post-apply audit: docs/ROLE-BPMN-POST-CLEANUP-2026-11-02.json.

## Library hierarchy navigation (2026-11-02)

- [DONE] Explicit activity-to-diagram links, bound to immutable library
  version IDs; same repository and increasing level, validated activity IDs.
- [DONE] Read-only diagram viewer, activity selection, detail navigation and
  breadcrumb return; explicit use-as-baseline reuses existing workspace flow.
- [DONE] Persistence/API/browser checks and documentation. No BPMN XML/source
  mutations, no Qwen calls, no automatic inference of hierarchy links.
Bulk decision: keep analysis one before/after pair. Bulk library imports are a
follow-up, distinct from cross-process analysis; not implemented in this task.

Evidence: Go API validation/persistence/relink/unlink tests and existing library
store/architecture tests pass; vet passes. Browser hierarchy test passes (11s):
activity click, zoom, saved relation, read-only child, breadcrumb, refresh
persistence, mobile width and unlink. Existing library/Canvas regression passes.
Typecheck and encoding guard pass. Initial browser test exposed ambiguous label
association; replaced the wrapped detail selector label with explicit htmlFor.
No case-study, retrieval method, Qwen or XML mutations in this increment.
Guide: docs/BPMN-LIBRARY.md. Bulk imports and bulk analysis are not implemented.

Viewer resize follow-up: ResizeObserver refits after width changes and hidden-tab
return. Browser check now asserts the selected activity fits inside the mobile
viewer, in addition to checking page overflow.

## Application navigation and result UX (2026-11-02)

- [DONE] Responsive sidebar/topbar and Overview using actual saved data;
  no fabricated recency/accuracy metrics. Preserve report URL restoration.
- [DONE] Integrate Canvas under Analysis; retain input/draft/session state across
  navigation and carry repository to Planning. Explicit new-analysis action.
- [DONE] Refine result hierarchy and density; retain all evidence, scoring and
  independent-change grouping. No retrieval or research-method changes.
- [DONE] Desktop/mobile browser verification, keyboard navigation, downloads,
  saved report recovery and library links. No lab calls or case-study writes.

UI evidence: docs/WORKSPACE-REDESIGN-2026-11-02.md. Initial targeted suite passed
11/12; duplicate Canvas button naming was corrected. Follow-up passed all seven
Overview/Canvas/workspace tests, including desktop collapse and mobile X/Escape.
Library, hierarchy and three saved-report restoration tests passed in the initial
suite. Mock Qwen recovery test passed without duplicate inference start. Final
Overview visual run passed after icon/menu fixes and hiding the Next dev indicator.
Typecheck, UTF-8 guard and diff checks pass. No real lab calls, dependency install
or analysis-method changes. Existing unrelated worktree changes preserved.

## Library/History findability (2026-11-02)

- [DONE] Library filters for repository, role and level alongside free text;
  clear filters and exact visible counts, preserving all saved versions/links.
- [DONE] History full-page list, text/method filters, refresh and clear empty/error
  states; no fabricated timestamps or changes to saved reports.
- [DONE] Targeted browser and TypeScript/encoding/whitespace verification.
No retrieval, persistence schema, LLM calls or case-study writes.

Evidence: combined filter/reset/empty/refresh-error/mobile browser case and
three saved-report restoration cases passed. Library save/refresh/Canvas case
passed after the filter count was changed to an aria-live announcement instead
of sharing the storage-notice status role. Typecheck, encoding and diff checks
passed. No backend/schema changes, source edits, dependency install or lab call.

## Final visual tone pass

[DONE] CSS-only readability/interaction polish: 14px body base with 1.5 line
height, white topbar, restrained hover/focus states, consistent metadata and
larger mobile navigation targets. Original SF/system font and palette preserved.
No function, schema or score change. Overview/analysis/mobile browser test passes
(8.5s), mobile screenshot inspected; TypeScript, encoding and diff checks pass.

## Pre-Qwen handoff completion (2026-11-02)

- [DONE] Rebase five Staff Inventory scenario pairs onto the actual current
  canonical BPMN. Fix stale result_merge targets and direct XOR-to-AND link.
  Preserve historical pairs/reports; keep provisional references frozen before
  inference and clearly separate from production inputs.
- [DONE] Extract UI analysis/comparison use case into a dependency-injected
  module; keep API, browser private storage and React rendering as adapters.
- [DONE] Snapshot/hash/source/convention/DI checks, repeatable BM25 smoke runs,
  relevant architecture/unit/browser checks and one clear handoff guide.
No new retrieval method, no lab inference, no source-code changes to Simple-ERP.
Full semantic/Qwen quality validation remains for the user's connected run.

Evidence: current-staff-v2 snapshot and verifier pass for 10 BPMNs; all five pairs
and provisional references rebuild byte-identically. BM25 ran twice per case,
identical JSON, expected primary anchor and evidence-bearing matches; unchanged
control has zero edits/candidates. 10 logic tests and 12 browser tests pass
(all diagrams, workspace auto-path/refresh, Qwen mocked checkpoint recovery).
Architecture/parser/diff tests pass. No semantic/Qwen real-model accuracy claim.
Read docs/PRE-QWEN-HANDOFF.md for the single current test entry point. Source
review is provisional, not independent ground truth. ERP files not modified.
The default tempfile scratch ACL failed under Windows sandbox; ordinary ignored
outputs/pre-qwen-ready/rebuilt was used for the reproducibility check instead.

## Workspace visual redesign

- [DONE] Charcoal navigation, softer neutral surfaces, clearer typography and form hierarchy.
- [DONE] Verify desktop/mobile navigation, analysis, restored reports and Planning handoff in Chrome.
Scope: frontend presentation only; preserve retrieval, saved data and model opt-in.

Evidence: eight Chrome tests passed (overview, workspace handoff, filters, five editor cases); desktop screenshots inspected. TypeScript and encoding checks passed. No real lab inference run in this UI pass.
