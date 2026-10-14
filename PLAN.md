# BPMN Impact Mapper — Phase 1 Plan

Status: **DONE — Phase 4.0 persisted analysis history verified**
Last updated: 2026-10-14

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

The research objective is to establish an auditable automatic pipeline comparing BM25-only, Semantic-only, BM25 + Semantic rank fusion, and Graph-Aware Hybrid before evidence-bounded LLM planning. Ranking is candidate retrieval, not causal proof. The tool must abstain when evidence is weak or ambiguous. The legacy configuration adapter is outside the primary product UI and target experiment because it requires manual mapping before inference.

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

Taxonomy: `ADD_TASK`, `REMOVE_TASK`, `MODIFY_TASK_LABEL`, `MODIFY_ACTOR_ROLE`, `ADD_GATEWAY`, `REMOVE_GATEWAY`, `MODIFY_GATEWAY_CONDITION`, `MODIFY_SEQUENCE_FLOW`, `ADD_DATA_REQUIREMENT`, `REMOVE_DATA_REQUIREMENT`, `ADD_STATE`, `MODIFY_TERMINAL_OUTCOME`, `UNRESOLVED_MATCH`.

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

Tokenization lowercases Unicode letters/digits, splits camel/snake/kebab boundaries, preserves stable IDs as normalized tokens, and removes no domain-specific terms. BM25 uses audited constants `k1=1.2`, `b=0.75`, a deterministic document order, and the Robertson IDF variant `ln(1 + (N-df+0.5)/(df+0.5))`. Equal scores tie by relative path then symbol.

Queries contain changed element IDs, labels, actors, flow conditions, data names, and neighboring labels. Empty/no-signal queries abstain.

## 13. Evidence, scoring, confidence, and unresolved rules

Evidence types include `TRACE_ID`, `LABEL_TOKEN`, `DOMAIN_TERM`, `PATH`, `SYMBOL`, `API_ROUTE`, `STATE_STATUS`, `DATABASE_TABLE`, `AUDIT_ACTION`, `TEST_NAME`, `BM25`, and `STRUCTURAL`.

Default score weights (configuration may override, then normalization is documented):

| Signal | Weight |
|---|---:|
| exact stable trace ID | 0.45 |
| symbol similarity | 0.15 |
| path/module similarity | 0.10 |
| domain/label overlap | 0.10 |
| API/state/table/audit/test structured hints | 0.10 |
| normalized BM25 | 0.10 |

Each component is clipped to `[0,1]`; total is their weighted sum divided by active total weight. BM25 normalization is `score / maxScore` within a query. Confidence is the explainable normalized score, not a calibrated probability.

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

The frozen synthetic manifest executes three methods × two repetitions and produces three scenario-method results with shared inputs. Deterministic benchmark output is separated from timing. Reviewer reconciliation intentionally leaves one synthetic conflict unresolved. A real Simple-ERP benchmark still requires user/reviewer-authored mutation scenarios and independently adjudicated ground truth; the tool does not fabricate them.

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
  Semantic Fusion, and Graph-Aware Hybrid on byte-identical inputs without a
  premature generalization claim.

Implementation evidence (2026-10-10): RRF unit tests cover agreement, stable
ties, evidence union, abstention, action safety, and graph-evidence precedence.
Real offline CLI smoke runs produced evidence-bearing `fusion` and `hybrid`
reports for all 12 synthetic changes. `go test -count=1 ./...`, `go vet ./...`,
TypeScript type-check, Next.js production build, `git diff --check`, and all
seven serial Playwright workflows pass. These results prove implementation
mechanics only; comparative accuracy remains blocked on hidden ground truth.
