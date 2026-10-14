# Architecture

## Clean Architecture rule

The dependency direction is transport/composition → application/use cases → ports → domain contracts. Concrete mapping adapters point inward to `mapping.Strategy`; `app.Analyzer` receives a `mapping.Resolver` and does not import BM25 or configuration adapters. `wiring` is the sole composition root that binds those adapters. `evaluation`, `benchmark`, `review`, `reviewpack`, and the case-agnostic `scenario` mutator are separate use cases and cannot import retrieval/configuration/filesystem adapters. A dependency-rule test parses imports and fails on boundary inversion.

The target repository is passive data. No component imports, compiles, starts, or otherwise executes it.

## Pipeline

1. `ingest.Build` canonicalizes the local root, applies exclusions and limits, skips symlinks/binary/secret/leakage files, and returns sorted relative paths with in-memory content.
2. `bpmn.Parse` tokenizes bounded XML, rejects directives, extracts supported nodes, lanes, data state/requirements, and sequence flows, then builds incoming/outgoing adjacency maps.
3. `diff.Compare` matches stable IDs first. A conservative unique structural fallback can match changed IDs; ties become `UNRESOLVED_MATCH`.
4. The indexing registry dispatches `.go` files to `go/parser` and every other allowed text file to the generic adapter. Relation extractors run only over compatible indexed files; adapter failures fall back to text with a warning.
5. The selected `mapping.Strategy` runs configuration-driven compatibility, BM25-only, offline Semantic, BM25 + Semantic RRF fusion, or Graph-Aware Hybrid retrieval and emits only candidates with evidence. The fusion adapter composes strategies through the application port instead of coupling their implementations.
6. `app.Analyzer` constructs a deterministic report and warnings. `report.JSON` is authoritative; `report.Markdown` is a pure view of the same object.
7. Separately, `experiment.Run` decodes a completed report and hidden ground truth, then invokes pure `evaluation.Evaluate`; there is no reverse edge into inference.
8. `benchmark.Runner` coordinates injected analyzer, file, and prediction-renderer ports. The experiment adapter supplies local filesystem behavior; hidden ground truth is read only after a completed prediction.
9. `review.Reconcile` compares independent annotations in memory and emits an agreed subset plus unresolved conflicts.
10. Before inference, optional experiment tooling applies a hash-locked `scenario.Spec` to BPMN bytes, validates the result with `bpmn.Parse`, and writes a new file through the outward `experiment` adapter. It never edits the baseline or target source repository.
11. The web adapter streams bounded BPMN uploads into a request-scoped temporary directory, invokes the same analyzer, persists the resulting canonical report through the `reportstore.Store` port, and removes uploads before returning.
12. Separately and only after a canonical report exists, `planning.Orchestrator`
    packages bounded evidence, invokes verifier/planner/reviewer provider roles,
    validates every returned target against mapper candidates, and renders a
    change plan. Planning never feeds candidates back into inference and never
    modifies repository source.
13. The web may request a planning record for an immutable analysis. The
    orchestrator binds it to the exact context/provider/model/prompt identity,
    then `planstore.Store` persists JSON and Markdown without overwriting a
    different plan. With `NoLLMProvider`, the record is explicitly `DISABLED`.

## Source relation model

`domain.SourceRelation` is adapter-neutral and links two indexed artifact IDs with a relation kind plus repository-relative call-site evidence. The Go extractor emits `CALLS` for free functions, `CALLS_METHOD` for a statically known receiver, and `INTERFACE_DISPATCH` only when one indexed named type uniquely satisfies the supported interface method set. Receiver fields can be followed through selector chains such as `handler.service.Execute()`.

The resolver compares complete supported method names and normalized parameter/result type shapes. It abstains on ambiguous implementations, embedded interfaces, generic or promoted dispatch, unresolved external implementations, unsupported type expressions, and shadowed receiver names. Retrieval expands only one edge from an exact-trace seed, discounts the dependency score, and always labels the propagated candidate `VERIFY`. Thus a static edge is useful retrieval context, not runtime or causality proof.

## Graph model

`domain.Graph` contains stable-ID maps for elements and flows plus `Incoming` and `Outgoing` adjacency maps. Serialized report elements contain sorted incoming/outgoing flow IDs, so no Go map iteration order reaches public output.

The parser recognizes task variants, call/subprocess activities, common events, five gateway types, sequence flows, lanes, data object/store references, data associations, conditions, and data states. Unsupported executable BPMN semantics are not simulated.

## Determinism

- inventory, artifacts, changes, impacts, evidence, and warnings use explicit stable sorting;
- BM25 ties use artifact ID;
- JSON uses typed structs and fixed indentation;
- analysis IDs hash logical repository metadata, BPMN bytes, and sorted allowed repository content;
- no timestamp, random value, absolute path, or environment-dependent Git command enters output.

## API boundary

`internal/api.Handler` supports only `POST /analyze`, limits request bodies, rejects unknown fields, honors request cancellation, and returns canonical JSON. Phase 1 intentionally provides no server command, upload, authentication, persistence, or multi-user lifecycle.

## Module boundaries

- Enterprise/domain: `domain`.
- Application ports/use cases: `mapping`, `app`, `evaluation`, `reportstore`,
  `planning`, `planstore`.
- Experiment/review use cases: `benchmark`, `review`, `reviewpack`, `scenario`.
- Optional post-inference planning use case and provider port: `planning`.
- Analysis/domain services: `bpmn`, `diff`, `index`.
- Outward adapters: `retrieval`, `configmapping`, `ingest`, `report`, `filestore`, `experiment`, `api`, CLI, Next.js.
- Optional outward embedding adapter: `semantic`; it invokes a trusted embedded
  Python helper and a locally cached model, never target-repository code.
- Optional outward LLM adapters: `llm`; Phase 1 still supplies only the offline
  provider, which also satisfies the planning port without making network calls.
- Composition: `wiring` only.

## Local web application

`cmd/bpmn-impact-mapper-server` is the Go process composition boundary. It configures the analyzer, filesystem report store, upload workspace, request limits, HTTP timeouts, and graceful shutdown. The versioned API exposes health, multipart analysis creation, canonical report retrieval, and JSON/Markdown downloads. API errors are bounded and do not return internal filesystem errors.

`web/` is a separate Next.js presentation adapter. During development, a same-origin rewrite proxies `/backend/*` to the Go server, avoiding browser CORS coupling. TypeScript types mirror the canonical report contract but contain no parser, diff, retrieval, scoring, or action-classification logic.

Method comparison is orchestration in the presentation adapter: it submits the same bounded form sequentially with different method selectors and rejects reports whose method-independent input fingerprints differ. Each response remains an ordinary immutable canonical report. The comparison computes descriptive set overlap only; research metrics stay in the post-inference evaluation use case.

Reviewer annotation creation remains browser-local export. The separate EvaluationDashboard uploads completed annotations to bounded evaluation routes. `review.Adjudicate` preserves agreements and validates explicit conflict resolutions; `evaluation.ValidateCoverage` enforces complete change coverage. The HTTP adapter loads immutable stored predictions and invokes the existing pure evaluator. It never invokes the analyzer or saves reviewer data. The returned bundle contains metrics, adjudication provenance and Markdown derived from the same Go evaluation objects.

The blind-review panel calls a separate post-inference `/api/v1/review-packs` route with immutable analysis IDs, the shared fingerprint, and reviewer identifiers only. The adapter reloads stored canonical reports and invokes the same pure `reviewpack.Build` use case as the CLI. The browser receives four separately downloadable artifacts; no reviewer finding or ground truth crosses this route. Batch method execution remains the existing frozen `benchmark` use case, which parses ground truth only after each prediction completes; a second batch engine would weaken rather than improve the method boundary.

Completed reports are immutable deterministic artifacts stored below a configurable local directory. This MVP intentionally uses no database: single-user local presentation does not need concurrent mutation, query indexes, identity, or transactional workflows. The `reportstore.Store` port permits a later PostgreSQL adapter without changing analysis or HTTP contracts.

Change plans use a separate filesystem root and `planstore.Store` port. Their
identity includes the analysis context and provider identity, allowing multiple
future model/prompt variants without mutating the canonical impact report.
Writes are atomic and idempotent; a different artifact cannot overwrite an
existing plan ID.

The report-store port also exposes deterministic summaries for history. Filesystem listing accepts only canonical analysis-ID directories and bounded regular, non-symlink JSON files, then sorts by analysis ID. The API and web history panel can reopen immutable reports without rerunning analysis. No creation timestamp is invented because canonical reports do not contain a trustworthy one.
