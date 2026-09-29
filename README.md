# BPMN Impact Mapper

BPMN Impact Mapper is an offline, third-party analysis tool that maps semantic BPMN changes to evidence-bearing software impact candidates. It is designed for applications that use BPMN as a specification and traceability source, not necessarily as a runtime workflow engine.

The implemented Phase 1/2 core is deterministic and standard-library-only. It inventories source without executing it, parses old/new BPMN, computes a conservative graph diff, indexes text plus native Go symbols, supports three isolated mapping methods, and evaluates completed reports against separately supplied hidden ground truth.

## Scope boundary

The mapper recommends artifacts for human review. It does not modify source, generate code, run the analyzed repository, migrate a database, deploy software, or prove that a candidate must change. Unsupported or ambiguous cases are reported as unresolved.

Implemented language support:

- Go: native syntax indexing through `go/parser`;
- other safe text files: generic textual indexing only.

No AST support is claimed for JavaScript, TypeScript, Java, Python, C#, or other languages in Phase 1.

## Requirements

- Go 1.26.0 (pinned in `go.mod`)
- no database, container, package download, network, or API key

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

- `hybrid`: explainable stable-ID, symbol/path/structured, lexical, and BM25 evidence;
- `bm25`: deterministic lexical baseline; every candidate action is `VERIFY`;
- `configuration`: frozen explicit mapping rules, requiring `--mapping-config FILE`.

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

Open `http://127.0.0.1:3000`. The browser uploads only the two BPMN documents; the repository and optional configuration are server-local paths. The backend writes canonical reports below `.data/analyses/<analysis-id>/` and deletes temporary BPMN uploads after every request.

For presentation, select **Load synthetic demo** and then **Run impact analysis**. The preset uses neutral tracked BPMN/source fixtures and is not a Simple-ERP rule, benchmark result, or ground truth. Start the Go server from the repository root so the relative synthetic repository path resolves correctly. Manual repository and BPMN input remains available.

Production verification:

```powershell
Set-Location web
npm run typecheck
npm run build
npm run test:e2e
```

The Playwright suite uses the installed Chrome browser, starts isolated local Go/Next servers, and covers both the one-click presentation demo and manual BPMN upload workflow.

API routes:

- `GET /api/v1/health`;
- `POST /api/v1/analyses` using bounded multipart form data;
- `GET /api/v1/analyses/{analysis-id}`;
- `GET /api/v1/analyses/{analysis-id}/report.json`;
- `GET /api/v1/analyses/{analysis-id}/report.md`.

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

The manifest pins SHA-256 for old/new BPMN, configuration, and hidden ground truth. The runner additionally requires a shared method-independent repository/BPMN input fingerprint and byte-identical predictions across repetitions. Deterministic metrics are written to `benchmark-report.json`/`.md`; volatile wall-clock measurements are isolated in `benchmark-timing.json`.

## Reviewer reconciliation

```powershell
go run ./cmd/bpmn-impact-mapper reconcile `
  --reviewer-a reviewer-a.json `
  --reviewer-b reviewer-b.json `
  --output outputs\reconciliation
```

Only exact agreements enter `agreed_ground_truth`. Missing cases and differing files, metadata, or actions remain explicit disagreements requiring human adjudication.

## Repository layout

- `cmd/bpmn-impact-mapper`: CLI boundary;
- `internal/app`: analysis orchestration and artifact writing;
- `internal/api`: bounded `net/http`-compatible handler;
- `internal/bpmn`: bounded BPMN XML parser;
- `internal/diff`: semantic graph matching and taxonomy;
- `internal/ingest`: secure local repository inventory;
- `internal/index`: generic text and native Go adapters;
- `internal/retrieval`: deterministic BM25, evidence, scoring, abstention;
- `internal/report`: canonical JSON and Markdown renderer;
- `internal/reportstore`: completed-analysis persistence port;
- `internal/filestore`: atomic local JSON/Markdown report adapter;
- `internal/llm`: provider contract and offline `NoLLMProvider`;
- `internal/mapping`: inward strategy/resolver ports and shared method options;
- `internal/configmapping`, `internal/retrieval`: outward method adapters;
- `internal/evaluation`: pure post-inference evaluation use case;
- `internal/benchmark`: injected-port benchmark orchestration and deterministic aggregation;
- `internal/review`: pure two-reviewer reconciliation;
- `internal/scenario`: case-agnostic, deterministic BPMN mutation use case;
- `internal/experiment`: evaluation and mutation filesystem adapters;
- `internal/wiring`: composition root binding ports to adapters;
- `web`: Next.js/TypeScript presentation adapter;
- `experiments`: tracked case-study protocols and frozen inputs, never product rules or ground truth;
- `testdata`: original synthetic BPMN/source/golden fixtures.

Detailed decisions are in [PLAN.md](PLAN.md), [architecture](docs/ARCHITECTURE.md), [methodology](docs/METHODOLOGY.md), [security](docs/SECURITY.md), and [evaluation](docs/EVALUATION.md).

## Current limitations

- structural element matching is deliberately conservative and can abstain;
- confidence is an explainable heuristic score, not a probability;
- generic layer/framework inference is path/content-based and may return `unknown`;
- BPMN choreography, conversation, full event semantics, and executable expressions are not interpreted;
- local files can change during analysis (TOCTOU); use a frozen snapshot for experiments;
- Current synthetic evaluation verifies mechanics, not research performance or generalization.
- ZIP, Git URL, frontend, external LLM providers, and large-scale experiment orchestration are deferred.
