# BPMN Impact Mapper — Phase 1 Plan

Status: **DONE — Phase 3.1 presentation hardening verified**
Last updated: 2026-09-28
Scope owner: this repository only

## 1. Problem statement and research objective

Software often follows BPMN as a specification without loading BPMN at runtime. A BPMN edit therefore does not directly identify which implementation artifacts require modification or verification. BPMN Impact Mapper is a third-party, mapping-only analysis tool that converts a semantic BPMN graph diff and a static repository index into ranked, evidence-bearing impact candidates.

The Phase 1 research objective is to establish a deterministic and auditable baseline for later comparison of configuration-driven mapping, lexical BM25 retrieval, hybrid retrieval, and hybrid retrieval with optional LLM verification. Ranking is candidate retrieval, not causal proof. The tool must abstain when evidence is weak or ambiguous.

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
