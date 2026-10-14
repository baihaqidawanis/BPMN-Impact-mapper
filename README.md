# BPMN Impact Mapper

BPMN Impact Mapper is an offline, third-party analysis tool that maps semantic BPMN changes to evidence-bearing software impact candidates. It is designed for applications that use BPMN as a specification and traceability source, not necessarily as a runtime workflow engine.

The implemented deterministic core uses the Go standard library. It inventories source without executing it, parses old/new BPMN, computes a conservative graph diff, indexes text plus native Go symbols, exposes four automatic mapping methods, and evaluates completed reports against separately supplied hidden ground truth. Semantic, Fusion, and Graph-Aware Hybrid use a local Python embedding runtime; BM25-only remains Go-only.

## Scope boundary

### Edit BPMN in the web

The web workspace has four top-level views: **Mapping**, **BPMN Canvas**,
**Saved analyses**, and **Guide**. Upload a baseline from Mapping or Canvas,
then edit its independent copy with the bpmn-js palette (task/gateway/flows;
wrench for task type). Select **Use edited BPMN** to return the current version
to Mapping, then run impact analysis. Uploading a separate new BPMN remains
supported. Switching views preserves editor state and undo/redo history. The
Mapping view warns when the canvas contains edits that have not been applied.
Drafts live in browser memory: download before refreshing.
See [scoring audit and editor limitations](docs/SCORING-AUDIT.md).

The mapper recommends artifacts for human review. It does not modify source, generate code, run the analyzed repository, migrate a database, deploy software, or prove that a candidate must change. Unsupported or ambiguous cases are reported as unresolved.

Implemented language support:

- Go: native syntax indexing through `go/parser`;
- other safe text files: generic textual indexing only.

No AST support is claimed for JavaScript, TypeScript, Java, Python, C#, or other languages in Phase 1.

The indexing registry supports ordered `SourceAdapter` and `RelationExtractor` extensions (`internal/index/registry.go`). Add native adapters through the registry without changing retrieval. JSON, Markdown and the web report include per-file source analysis coverage: `TEXT_ONLY`, `AST_SYMBOLS`, limited `STATIC_CALLS`, and `TYPE_RELATIONS`. The Go extractor resolves repository-local free functions, statically typed receiver methods, receiver struct fields, and uniquely implemented interfaces. Parse failures explicitly fall back to text. These labels describe implemented capability, not measured accuracy or a complete call graph. New adapter behavior uses a versioned name so analysis identity remains reproducible.

## Requirements

- Go 1.26.0 (pinned in `go.mod`)
- no database, container, network call, or API key
- Semantic, Fusion, and Hybrid methods: Python 3, `sentence-transformers`, and a locally cached
  `all-MiniLM-L6-v2` model; the analyzer forces offline model loading

## Build and test

```powershell
go test ./...
go vet ./...
go build -o bin/bpmn-impact-mapper.exe ./cmd/bpmn-impact-mapper
```

```sh
go test ./...
go vet ./...
go build -o bin/bpmn-impact-mapper ./cmd/bpmn-impact-mapper
```

## Analyze

PowerShell:

```powershell
go run ./cmd/bpmn-impact-mapper analyze `
  --repo C:\path\to\repository `
  --old-bpmn C:\path\to\old.bpmn `
  --new-bpmn C:\path\to\new.bpmn `
  --output C:\path\to\output `
  --repository-name example `
  --revision snapshot-1 `
  --method hybrid
```

Unix shell:

```sh
go run ./cmd/bpmn-impact-mapper analyze \
  --repo /path/to/repository \
  --old-bpmn /path/to/old.bpmn \
  --new-bpmn /path/to/new.bpmn \
  --output /path/to/output \
  --repository-name example \
  --revision snapshot-1 \
  --method hybrid
```

Methods:

- `hybrid`: graph-aware method that fuses the deterministic stable-ID, symbol/path/structured, lexical, BM25, and source-relation ranking with Semantic using RRF; deterministic actions are preserved only for candidates independently supported by the graph-aware side;
- `fusion`: BM25 + Semantic rank fusion using deterministic RRF (`k=60`); every candidate action is `VERIFY`;
- `bm25`: deterministic lexical baseline; every candidate action is `VERIFY`;
- `semantic`: offline embedding/cosine baseline; every candidate action is
  `VERIFY`, and semantic similarity is not a probability;

Semantic can be run directly with `--method semantic`. It intentionally excludes
stable BPMN IDs from its embedding query so it measures vocabulary-based
retrieval rather than duplicating exact trace matching. A missing local model is
reported as an error; it never silently falls back to BM25 or downloads a model.

Fusion can be run with `--method fusion`. It combines ranks rather than adding
raw BM25 and cosine values, because those scores are not on a common scale.
Its normalized RRF value is rank consensus, not an accuracy probability.

Hybrid treats a label-only task change conservatively: even a high-scoring exact trace remains `VERIFY`, because changed wording does not by itself prove changed software behavior. Exact trace evidence still raises and explains the candidate ranking.

Topology edits may also carry optional composite context: `INSERT_TASK`, `REMOVE_TASK_WITH_REWIRE`, `ADD_GATEWAY_BRANCH`, `MOVE_TASK`, or `REMOVE_GATEWAY_BRANCH`. Atomic changes remain canonical, while predecessor/successor context contributes deterministic retrieval terms. A composite label explains related BPMN edits; it is not proof that a source change is required.

Required output:

- `impact-report.json` — authoritative canonical artifact;
- `impact-report.md` — view rendered from the same report object.

Useful limits and controls:

- `--ignore pattern` (repeatable, slash-normalized glob);
- `--max-file-bytes`, `--max-total-bytes`, and `--max-files`;
- `--top-k`;
- `--timeout`.

The analyzer never invokes Git in the target repository. Supply `--revision` as frozen experiment metadata.

## Local web application

The presentation MVP uses a Go HTTP server and a Next.js frontend. It has no authentication, database, queue, external API, or source-execution capability.

Start the Go API from the repository root:

```powershell
$env:GOCACHE = (Resolve-Path .gocache).Path
go run ./cmd/bpmn-impact-mapper-server --addr 127.0.0.1:8080 --data-dir .data
```

In another terminal, start Next.js:

```powershell
Set-Location web
npm install
npm run dev
```

Open `http://127.0.0.1:3000`. The browser uploads only the two BPMN documents; the repository is currently a server-local read-only path. The backend writes canonical reports below `.data/analyses/<analysis-id>/` and deletes temporary BPMN uploads after every request.

The **Saved analyses** panel lists deterministic summaries from local immutable reports and can reopen a report after browser refresh without rerunning inference. It intentionally shows no unreliable creation timestamp.

For presentation, select **Load synthetic demo** and then **Run impact analysis**. The preset uses neutral tracked BPMN/source fixtures and is not a Simple-ERP rule, benchmark result, or ground truth. Start the Go server from the repository root so the relative synthetic repository path resolves correctly. Manual repository and BPMN input remains available.

The demo selector also provides focused insert-task, add/remove gateway branch, move-task, and ambiguous-match scenarios. These are deterministic functionality demonstrations, not accuracy evidence.

Select **Compare mapping methods** to run the same input sequentially through BM25, Semantic, BM25 + Semantic Fusion, and Graph-Aware Hybrid. The comparison verifies one shared input fingerprint and shows retrieval counts, unresolved cases, overlap, and per-change top candidates. These are method differences, not accuracy metrics. On the current machine a cold Semantic run can take tens of seconds because the local model starts in a separate process; identical embeddings are reused in memory by the following Fusion and Hybrid runs.

After any report, open **Reviewer annotation** to select expected impacts, correct their expected action, and add missed files manually. The browser downloads a version-1 reviewer JSON compatible with `bpmn-impact-mapper reconcile`. Candidate selections are not automatic ground truth; independently inspect the frozen implementation evidence before export.

Open **Evaluation dashboard** below the report, upload Reviewer A/B JSON, and select **Reconcile reviewers**. Resolve each conflict explicitly by choosing one reviewer, entering corrected expected impacts, or confirming an empty negative case. Confirm the annotations apply to the selected snapshot, then **Evaluate stored reports**. The Go evaluation API loads the already completed reports, requires complete change coverage and matching fingerprints, and computes file-level metrics. Download the evaluation bundle (including reviewers and adjudication decisions), Markdown, and final ground truth. Uploaded reviews are processed only by evaluation, never inference; no review/evaluation files are persisted on the server.

See [the ground-truth protocol](docs/GROUND-TRUTH-PROTOCOL.md) before interpreting Recall/Precision. In particular, a new BPMN task has no objectively known affected implementation until a closed reference implementation exists or independent reviewers make a documented architecture judgment.

Production verification:

```powershell
Set-Location web
npm run typecheck
npm run build
npm run test:e2e
```

The Playwright suite uses the installed Chrome browser, starts isolated local Go/Next servers, and covers the one-click presentation demo, manual BPMN upload, composite context, same-input method comparison, reviewer annotation export, and reconciliation/adjudication/evaluation downloads.

API routes:

- `GET /api/v1/health`;
- `GET /api/v1/analyses` (deterministic stored-report summaries);
- `POST /api/v1/analyses` using bounded multipart form data;
- `GET /api/v1/analyses/{analysis-id}`;
- `GET /api/v1/analyses/{analysis-id}/report.json`;
- `GET /api/v1/analyses/{analysis-id}/report.md`.
- `GET /api/v1/analyses/{analysis-id}/planning-context.json` (bounded,
  inspectable pre-LLM evidence only; no model call);
- `POST /api/v1/analyses/{analysis-id}/plans` (creates an immutable planning
  record; returns `DISABLED` while `NoLLMProvider` is configured);
- `GET /api/v1/analyses/{analysis-id}/plans`;
- `GET /api/v1/plans/{plan-id}/change-plan.json`;
- `GET /api/v1/plans/{plan-id}/change-plan.md`;
- `POST /api/v1/reviews/reconcile` (bounded JSON annotations);
- `POST /api/v1/review-packs` (stored report IDs and reviewer identifiers; no ground truth);
- `POST /api/v1/evaluations` (stored report IDs, annotations, explicit conflict decisions).

## Generate a controlled BPMN scenario

The experiment-only mutator creates a new BPMN from a versioned specification. It verifies the frozen baseline SHA-256, requires every operation to match exactly once, validates the result with the production parser, and refuses to overwrite an existing file:

```powershell
go run ./cmd/bpmn-impact-mapper mutate `
  --spec experiments\simple-erp\specs\p2p-task-label.json `
  --baseline C:\Simple-ERP\docs\bpmn\l2-procure-to-pay.bpmn `
  --output outputs\pilots\generated\p2p-task-label.bpmn
```

Supported operations are `set_attribute`, `set_child_text`, and `remove_element`. This command changes BPMN experiment input only; it never changes source code or the analyzed repository.

## Evaluate

Evaluation is a separate post-inference command. Ground truth is never passed to `analyze`:

```powershell
go run ./cmd/bpmn-impact-mapper evaluate `
  --report outputs\hybrid\impact-report.json `
  --ground-truth experiment\ground-truth.json `
  --output outputs\hybrid\evaluation
```

It writes `evaluation-report.json` and `evaluation-report.md` with Recall@1/@3/@5, Precision@1/@3/@5, MRR, MAP, file precision/recall/F1, module/layer/action accuracy, unresolved rate, false-positive rate, counts, and reproducibility hashes.

## Frozen benchmark

Run every method repeatedly against the same locked scenario manifest:

```powershell
go run ./cmd/bpmn-impact-mapper benchmark `
  --manifest testdata\benchmark\manifest.json `
  --output outputs\benchmark
```

The manifest pins SHA-256 for old/new BPMN, configuration, and hidden ground truth. The runner additionally requires a shared method-independent repository/BPMN input fingerprint and byte-identical predictions across repetitions. Deterministic metrics are written to `benchmark-report.json`, `.md`, and spreadsheet-ready `.csv`; volatile wall-clock measurements are isolated in `benchmark-timing.json`.

## Reviewer reconciliation

For a method-blind review, prepare a new immutable pack from completed reports for the same input snapshot:

```powershell
go run ./cmd/bpmn-impact-mapper prepare-review `
  --report outputs\bm25\impact-report.json `
  --report outputs\hybrid\impact-report.json `
  --output outputs\review-pack `
  --reviewer-a reviewer-a `
  --reviewer-b reviewer-b
```

Give each reviewer only their JSON template plus `review-guide.md`; keep `coordinator-manifest.json` with the researcher. The guide contains BPMN changes and repository revision but no method, candidate, evidence, ranking, or score. Each template starts with an intentionally invalid `REVIEW_REQUIRED` sentinel. Replace it with reviewed expected impacts, or remove it and use `[]` only for an explicit negative judgment. An untouched template fails reconciliation. The output directory must not already exist, preventing accidental replacement of a frozen pack.

```powershell
go run ./cmd/bpmn-impact-mapper reconcile `
  --reviewer-a outputs\review-pack\reviewer-a.json `
  --reviewer-b outputs\review-pack\reviewer-b.json `
  --manifest outputs\review-pack\coordinator-manifest.json `
  --output outputs\reconciliation
```

With `--manifest`, reconciliation verifies the pack ID, case digest, canonical report metadata, and exact completed-case coverage before writing snapshot provenance into JSON/Markdown. The flag remains optional for compatibility with legacy annotations. Only exact agreements enter `agreed_ground_truth`; differing files, metadata, or actions remain explicit disagreements requiring human adjudication.

## Repository layout

- `cmd/bpmn-impact-mapper`: CLI boundary;
- `internal/app`: analysis orchestration and artifact writing;
- `internal/api`: bounded `net/http`-compatible handler;
- `internal/bpmn`: bounded BPMN XML parser;
- `internal/diff`: semantic graph matching and taxonomy;
- `internal/ingest`: secure local repository inventory;
- `internal/index`: generic text and native Go adapters;
- `internal/retrieval`: deterministic BM25, evidence, scoring, abstention;
- `internal/planning`: bounded post-inference verifier/planner/reviewer contracts,
  deterministic context identity, timed orchestration, validation, and canonical
  change-plan rendering;
- `internal/planstore`: immutable change-plan persistence port;
- `internal/report`: canonical JSON and Markdown renderer;
- `internal/reportstore`: completed-analysis persistence port;
- `internal/filestore`: atomic local JSON/Markdown report adapter;
- `internal/llm`: provider contract and offline `NoLLMProvider`;
- `internal/mapping`: inward strategy/resolver ports and shared method options;
- `internal/configmapping`, `internal/retrieval`: outward method adapters;
- `internal/evaluation`: pure post-inference evaluation use case;
- `internal/benchmark`: injected-port benchmark orchestration and deterministic aggregation;
- `internal/review`: pure two-reviewer reconciliation;
- `internal/reviewpack`: pure blind-review pack validation and rendering;
- `internal/scenario`: case-agnostic, deterministic BPMN mutation use case;
- `internal/experiment`: evaluation, review-pack, benchmark, reconciliation, and mutation filesystem adapters;
- `internal/wiring`: composition root binding ports to adapters;
- `web`: Next.js/TypeScript presentation adapter;
- `experiments`: tracked case-study protocols and frozen inputs, never product rules or ground truth;
- `testdata`: original synthetic BPMN/source/golden fixtures.

Detailed decisions are in [PLAN.md](PLAN.md), [architecture](docs/ARCHITECTURE.md), [methodology](docs/METHODOLOGY.md), [semantic retrieval research proposal](docs/SEMANTIC_RETRIEVAL.md), [security](docs/SECURITY.md), and [evaluation](docs/EVALUATION.md).

## Current limitations

- structural element matching is deliberately conservative and can abstain;
- confidence is an explainable heuristic score, not a probability;
- generic layer/framework inference is path/content-based and may return `unknown`;
- Go type relations abstain on multiple interface implementations, embedded interfaces, generic dispatch, promoted methods, unsupported type expressions, and values whose static type cannot be derived locally;
- BPMN choreography, conversation, full event semantics, and executable expressions are not interpreted;
- local files can change during analysis (TOCTOU); use a frozen snapshot for experiments;
- Current synthetic evaluation verifies mechanics, not research performance or generalization.
- ZIP/Git URL ingestion, additional native language adapters, external LLM providers, and large-scale experiment orchestration are deferred.
