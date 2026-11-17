# Add-task AI review: saved diagnostic

Analysis: `89a7e1615e6e85aa107dfe9e04f6dbfd`. Agent session:
`25ce84d91e6de0a3d3244e004fa28e9c`. Plan:
`83516040b1309d2943c992ee3861609d`. Model: `qwen3.5:9b-q8_0`.
Session remains **NEEDS_REVIEW**. This measures the exported draft, not an approved result.

Current-before source mapping and the after comparison are documented in
[Staff adjustment submit](../experiments/simple-erp/baseline-traceability/adj-submit-current.md).
The broader historical baseline snapshot had a stale merge gateway for this
process; the focused mapping was refreshed from actual current XML. That does
not change the three-file after reference or retroactively change model output.

## Scope and reference

The positive reference is `experiments/simple-erp/current-staff-v2/add-task/reference.json`:
ADD_TASK / ReasonReview, requiring a non-empty reason before adjustment submit.
It expects the inventory frontend panel, inventory PostgreSQL store and operations
integration test. All three source hashes matched `source-review.json` at inspection.
The two sequence-flow edits are outside labelled scope. This is a provisional
source-derived reference, not independently adjudicated or implementation-diff truth.
It was prepared before inference; do not feed it to agents as an answer key.

## File selection

| Output | Selected | Reference matches | Precision | Recall | F1 |
|---|---:|---:|---:|---:|---:|
| Hybrid displayed Top 5 | 5 | 3 | 60% | 100% | 75% |
| Hybrid displayed Top 10 | 10 | 3 | 30% | 100% | 46.15% |
| Hybrid + AI draft selection | 6 | 3 | 50% | 100% | 66.67% |

Precision = matches / selected unique files; recall = matches / expected files.
F1 is their harmonic mean. Top-k is a display choice, not a confidence-based
automatic selector. Top 5 is not evidence of a superior selection algorithm.
AI retained all expected files while reducing the displayed pool from ten to six.
Domain adjustment, business rules documentation and migration are extra proposals
relative to this reference. They count against this precision, but the limited
reference does not independently prove every extra file irrelevant.

Hybrid's first three files are the three reference files, so AP=1, reciprocal
rank=1 and R-precision=1 for this single query. AI proposals have no declared
relevance ordering. AI AP/MAP, reciprocal rank/MRR and R-precision are unavailable;
neither alphabetical Markdown order nor batch traversal order is a valid ranking.
MAP/MRR across several queries must be computed per query then macro-averaged.
Do not combine this single case with the existing four-case benchmark averages.

## Manual plan quality: 55 / 100

This is an explicit engineering judgement, not a validated rubric or research
accuracy metric. Each aspect has a maximum of 20:

| Aspect | Score | Rationale |
|---|---:|---|
| Structure and requirement coverage | 16 | Useful per-file behavior, steps and acceptance sections |
| Evidence grounding | 12 | Correct main files, but several claims exceed inspected evidence |
| Technical correctness | 8 | Go string/nil error; create/submit confusion; unjustified schema constraint |
| Implementation specificity | 11 | Some useful guards and UI proposals; tests still ambiguous |
| Scope and consistency | 8 | Guard can affect unrelated actions; review issues remain unresolved |

Length alone does not improve this score. The same errors remain blockers:
Go `string` cannot be nil; validate only the required submit action; use the
existing submit endpoint; do not impose schema constraints merely because a
column exists; distinguish successful submit tests from intentional failure tests.
Final combined review exceeded its context or retry budget. This measurement does
not fix the agent pipeline or establish a successful final consistency review.

## Offline reproduction

From the mapper root (one PowerShell command):

```powershell
python scripts/evaluate_agent_selection.py --session .data/agent-sessions/25ce84d91e6de0a3d3244e004fa28e9c.json --report .data/analyses/89a7e1615e6e85aa107dfe9e04f6dbfd/impact-report.json --reference experiments/simple-erp/current-staff-v2/add-task/reference.json --date 2026-11-05 --output web/public/benchmarks/agent-add-task-2026-11-05.json
```

The script checks analysis/fingerprint equality, deduplicates file predictions,
rejects absolute output paths and includes input hashes. It exports no source
excerpts, local repository root or private model responses. The public snapshot
appears in Research under Hybrid + AI review and is downloadable. Opening it
does not call a model and does not evaluate a newly selected repository.

Offline checks cover arithmetic, input mismatch, reference scope, duplicate files,
path restrictions, UI display/download and no inference request. Actual model
improvements require a fresh live run after the agent pipeline is corrected.

Verification: four Python unit tests, TypeScript, UI encoding and diff whitespace checks pass. Three Playwright benchmark cases pass (14.1 seconds), including AI metrics/download and mobile wrapping. No lab model call was made.
