# Architecture

The mapper uses a modular Go backend and a separate Next.js presentation layer. It produces impact candidates and optional developer plans; it never executes or edits the analyzed repository.

## Responsibilities and dependencies

| Area | Packages | Responsibility |
|---|---|---|
| Contracts and domain | domain, mapping | BPMN changes, source artifacts, mapping strategy ports |
| Analysis | bpmn, diff, index, retrieval, fusion, semantic | Parse, compare, index and retrieve evidence; embedding runtime is an outward adapter |
| Planning core | planning | Analyst/Planner/Reviewer orchestration, budgets, validation, resumable partitions and canonical plan rendering |
| Agent application service | agentjobs | Async job lifecycle, cancellation, approved source/model preparation and persistence through ports |
| Outward adapters | llm, agenttools, ingest, filestore | Model HTTP transport, read-only source tools, repository ingestion and local storage |
| Delivery | api, web | HTTP endpoints and presentation; no duplicate scoring implementation in the UI |
| Composition | wiring, cmd | Select concrete adapters and wire application services |
| Research | evaluation, benchmark, experiment, experiments/tools | Hidden-reference evaluation and reproducible studies, outside model evidence |

planning depends on domain contracts, not HTTP, Ollama, filesystem storage or concrete source tools. agentjobs receives model/source/store interfaces. llm and filestore point inward to those contracts. RoleRouter selects one configured model per role and serializes calls; it does not contain planning rules or silently replace models.

### Go call-literal facts for source inspection

Source tools v11 enrich `inspect_source` with at most one bounded
`AST_CALL_LITERALS` item from the approved in-memory Go file. It reports a call
name and explicitly 1-based string-literal argument positions near inspected
source spans, for example `3="publish", 5=""`. This is syntax, not inferred
parameter meaning, a resolved dependency or runtime behavior. Original source
excerpts remain alongside it; expressions are not evaluated. Long literal values
are omitted, facts remain within the existing evidence-size/count bounds, and
malformed Go or unsupported languages retain their text fallback.

The change strengthens source evidence for agents; it adds no retrieval method,
ranking weight, embedding model or source execution. Snapshot identity changes,
so start a fresh AI session after restarting the backend. Saved mappings remain
usable when their repository snapshot matches. Fixture tests cover literal
positions, empty strings versus expressions, stable ordering, snapshot isolation,
text fallback and retention through the planning evidence merge. Offline checks
do not establish improved model accuracy; live comparison remains pending.

This is pragmatic Clean Architecture, not a claim of a completely pure ports-and-adapters design. The path-based app facade still invokes ingestion/index services and filesystem compatibility helpers directly. Those helpers are isolated in internal/app/files.go; internal/app/analyzer.go coordinates analysis. Splitting files clarifies responsibilities without pretending the package dependency changed.

Architecture tests enforce allowed internal imports, plus no direct filesystem/network/process/database imports in domain, mapping, planning and agentjobs. Adapter dependency checks cover llm, agenttools and filestore. These tests protect dependency direction; they do not prove runtime correctness.

## Data flow

Repository and before/after BPMN -> secure ingestion -> semantic parser -> conservative graph diff -> source index -> selected retrieval method -> canonical JSON -> Markdown/web.

Optional planning starts from a saved immutable analysis and its BPMN graphs. Analyst inspects source, Planner proposes edits or an evidence-backed NO_ACTION, and Reviewer checks each target and combined consistency. Oversized planning/review turns are partitioned without silently dropping targets. Completed parts are cached against the exact source, draft, feedback and model identity. Additional calls inside a planning/review partition consume the same child-session turn budget. Focused seed groups have separate bounded child sessions: an oversized or capacity-exhausted multi-seed group can be split deterministically into smaller groups, with failed work retained in total counts and checkpoints. Splitting stops at one seed; no candidate is dropped and no approval is inferred from a failed attempt.

Scores are retrieval signals, not probabilities. Model approval is not independent correctness. A relevant file is not automatically a file requiring edits. NO_ACTION assessments are retained for audit and excluded from proposed-edit counts.

## Interfaces and persistence

- api exposes versioned health, analyses, BPMN library, planning/agent sessions, report/plan downloads and research endpoints; it is no longer the historical Phase-1 POST /analyze-only adapter.
- reportstore and planstore define persistence ports; filestore implements them.
- .data stores local analyses, BPMN snapshots, sessions and plans. It is not source code or a cleanup target.
- outputs contains generated results and older local tools. Its go.mod prevents generated Go fixtures becoming application packages; only that boundary file is eligible for version control.
- Deterministic analysis IDs bind logical inputs and configured methodology. Agent identities separately bind source/model/prompt/profile policy; changes require a fresh agent session, not re-uploading intact saved BPMN snapshots.

## Development and diagnostics

Normal tests: go test ./... and go vet ./.... Use scripts/verify-agents.ps1 for bounded offline agent/API/frontend checks. It disables dependency downloads and restores its process environment afterward. No browser is started unless -Browser is explicitly passed.

Lab/checkpoint diagnostic harnesses are excluded from normal compilation with the labdiagnostics build tag. For an explicitly configured experiment, use go test -tags labdiagnostics ./internal/llm or ./internal/api with -run selecting the intended test. Existing per-test environment opt-ins are still required; the tag alone does not contact a model. Keep raw diagnostic payloads in ignored .data, not tracked source.

Active experiments: experiments/simple-erp/current-staff-v2 and baseline-traceability. Historical packages and frozen hashes remain in experiments/history. They are regression/research evidence, not production code and not automatically obsolete.

Source discovery includes bounded declaration evidence from the approved snapshot. Declaration anchors survive later action-window merges, so parameter information is not lost merely because a local branch was inspected. Excerpts remain partial source observations, not proof that unshown behavior is absent.

Research model-choice summaries are saved experiments, not runtime routing or proof of production quality. Structural validity, control verdicts, file-selection metrics and technical plan correctness are separate measurements.
