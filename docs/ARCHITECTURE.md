# Architecture

## Clean Architecture rule

The dependency direction is transport/composition → application/use cases → ports → domain contracts. Concrete mapping adapters point inward to `mapping.Strategy`; `app.Analyzer` receives a `mapping.Resolver` and does not import BM25 or configuration adapters. `wiring` is the sole composition root that binds those adapters. `evaluation`, `benchmark`, `review`, and the case-agnostic `scenario` mutator are separate use cases and cannot import retrieval/configuration/filesystem adapters. A dependency-rule test parses imports and fails on boundary inversion.

The target repository is passive data. No component imports, compiles, starts, or otherwise executes it.

## Pipeline

1. `ingest.Build` canonicalizes the local root, applies exclusions and limits, skips symlinks/binary/secret/leakage files, and returns sorted relative paths with in-memory content.
2. `bpmn.Parse` tokenizes bounded XML, rejects directives, extracts supported nodes, lanes, data state/requirements, and sequence flows, then builds incoming/outgoing adjacency maps.
3. `diff.Compare` matches stable IDs first. A conservative unique structural fallback can match changed IDs; ties become `UNRESOLVED_MATCH`.
4. `index.Build` dispatches `.go` files to `go/parser` and every other allowed text file to the generic adapter. Adapter failures fall back to text with a warning.
5. The selected `mapping.Strategy` runs configuration-driven, BM25-only, or hybrid retrieval and emits only candidates with evidence.
6. `app.Analyzer` constructs a deterministic report and warnings. `report.JSON` is authoritative; `report.Markdown` is a pure view of the same object.
7. Separately, `experiment.Run` decodes a completed report and hidden ground truth, then invokes pure `evaluation.Evaluate`; there is no reverse edge into inference.
8. `benchmark.Runner` coordinates injected analyzer, file, and prediction-renderer ports. The experiment adapter supplies local filesystem behavior; hidden ground truth is read only after a completed prediction.
9. `review.Reconcile` compares independent annotations in memory and emits an agreed subset plus unresolved conflicts.
10. Before inference, optional experiment tooling applies a hash-locked `scenario.Spec` to BPMN bytes, validates the result with `bpmn.Parse`, and writes a new file through the outward `experiment` adapter. It never edits the baseline or target source repository.
11. The web adapter streams bounded BPMN uploads into a request-scoped temporary directory, invokes the same analyzer, persists the resulting canonical report through the `reportstore.Store` port, and removes uploads before returning.

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
- Application ports/use cases: `mapping`, `app`, `evaluation`, `reportstore`.
- Experiment/review use cases: `benchmark`, `review`, `scenario`.
- Analysis/domain services: `bpmn`, `diff`, `index`.
- Outward adapters: `retrieval`, `configmapping`, `ingest`, `report`, `filestore`, `experiment`, `api`, CLI, Next.js.
- Composition: `wiring` only.

## Local web application

`cmd/bpmn-impact-mapper-server` is the Go process composition boundary. It configures the analyzer, filesystem report store, upload workspace, request limits, HTTP timeouts, and graceful shutdown. The versioned API exposes health, multipart analysis creation, canonical report retrieval, and JSON/Markdown downloads. API errors are bounded and do not return internal filesystem errors.

`web/` is a separate Next.js presentation adapter. During development, a same-origin rewrite proxies `/backend/*` to the Go server, avoiding browser CORS coupling. TypeScript types mirror the canonical report contract but contain no parser, diff, retrieval, scoring, or action-classification logic.

Completed reports are immutable deterministic artifacts stored below a configurable local directory. This MVP intentionally uses no database: single-user local presentation does not need concurrent mutation, query indexes, identity, or transactional workflows. The `reportstore.Store` port permits a later PostgreSQL adapter without changing analysis or HTTP contracts.
