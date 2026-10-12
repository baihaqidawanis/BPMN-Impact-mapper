# BPMN Impact Report

- Analysis ID: `abc123`
- Input fingerprint: `input123`
- Method: `hybrid`
- Repository: `synthetic`
- Revision: `fixture-v1`
- Changes: 1

## 1. ADD_TASK — TASK-VALIDATE

Element: Validate request

Reason: task added

Composite pattern: `INSERT_TASK`

Predecessors: `TASK-CREATE (Create request)`

Successors: `TASK-REGISTER (Register request)`

| Action | Confidence | Module | Layer | File | Symbol |
|---|---:|---|---|---|---|
| ADD | 0.8000 | internal | application | `internal/application/request.go` | `ValidateRequest` |

### Evidence: `internal/application/request.go` / `ValidateRequest`

Score: trace=1.0000, symbol=0.5000, path=0.2500, lexical=0.5000, structured=1.0000, BM25=1.0000, semantic=0.0000, total=0.8000.

- **TRACE_ID** `internal/application/request.go:4`: TASK-VALIDATE

## Warnings

- **EXAMPLE**: synthetic warning
