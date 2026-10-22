# BPMN Impact Report

- Analysis ID: `abc123`
- Input fingerprint: `input123`
- Method: `hybrid`
- Repository: `synthetic`
- Revision: `fixture-v1`
- Changes: 1

Retrieval scores order candidates; they are not probabilities of a required change. Code evidence, meaning match and rank agreement are separate measurements. RRF uses ranking positions, so its score can be high without a direct BPMN ID match. A new task or gateway may not have an ID in source yet; the current code-evidence weights are not recalibrated. TRACE_ID links the changed element; CONTEXT_TRACE_ID links a neighbouring element only and adds no separate score.

## 1. ADD_TASK — TASK-VALIDATE

Element: Validate request

Reason: task added

Composite pattern: `INSERT_TASK`

Predecessors: `TASK-CREATE (Create request)`

Successors: `TASK-REGISTER (Register request)`

| Action | Retrieval score | Module | Layer | File | Function / code name |
|---|---:|---|---|---|---|
| ADD | 0.8000 | internal | application | `internal/application/request.go` | `ValidateRequest` |

### Evidence: `internal/application/request.go` / `ValidateRequest`

Code-evidence score: 0.8000. Meaning match: not scored. Rank agreement: not combined.

Score: trace=1.0000, symbol=0.5000, path=0.2500, lexical=0.5000, structured=1.0000, BM25=1.0000, semantic=0.0000, total=0.8000.

- **TRACE_ID** `internal/application/request.go:4`: TASK-VALIDATE

## Warnings

- **EXAMPLE**: synthetic warning
