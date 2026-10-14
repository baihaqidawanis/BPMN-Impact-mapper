# Scoring audit and canvas increment — 2026-10-13

## Findings

The current deterministic branch uses trace 0.45, symbol 0.15, path 0.10,
lexical 0.10, structured 0.10, BM25 0.10. These weights are manually chosen
heuristics, not learned or calibrated probabilities. They remain unchanged in
this increment so that an interface improvement is not confused with validated
accuracy improvement.

- A newly introduced BPMN ID normally has no occurrence in the baseline code.
  Without trace, the deterministic direct score has a ceiling of 0.55.
- Trace matching is case-insensitive and boundary-aware. It accepts an exact ID
  and an operation suffix such as `TASK-001-create`, but rejects identifier
  collisions such as `TASK-0010`, `XTASK-001`, and `TASK-001_extra`.
- Symbol=0 can mean no matching symbol tokens or no extracted symbol. Generic
  textual adapters do not provide the same coverage as Go's native adapter.
- Path=0 means no matching path tokens. It does not prove irrelevance.
- Symbol and path overlap recognize a small, deterministic set of grammatical
  variants (`request`/`requests`, `purchase`/`purchasing`). This does not apply
  broad stemming or synonyms, and therefore does not turn unrelated shared
  prefixes into evidence. Lexical overlap and BM25 remain unchanged.
- Label/symbol/path overlap saturates at five distinct matching query tokens.
  Thus 1.0 does not imply complete semantic agreement.
- Structured=1 is currently a coarse keyword/category heuristic; it is not proof
  of a matching API route, SQL table, or dependency. This requires future
  field-specific extraction and evaluation.
- Semantic-only and BM25-only do not evaluate all deterministic signals; zero
  fields in their legacy score contract must not be treated as negative evidence.
- RRF uses equal ranking-list weights, k=60, and normalization by 2/61.
  Rank 1 in both lists yields 1.0; rank 1 in one list only yields 0.5. This is
  agreement between retrievers, not probability of a required change.

## Implemented correction

Previously TopK truncated both retrieval lists before fusion. A shared candidate
at position 6 could disappear when the display limit was 5. Both branches now
retain every threshold-qualified artifact before fusion; only final output is
limited. A regression checks that a shared rank-2 candidate wins even at TopK=1
and that its score matches TopK=3. Fusion metadata and analysis identity use
`reciprocal-rank-fusion-v2-full-pool`; historical reports remain untouched.

The web shows a 0–1 retrieval score or rank consensus rather than a percentage
called confidence, and explains zero contributions and the weight limitations.
The JSON confidence field is retained for compatibility. Action rules remain
evidence-based; semantic agreement does not promote VERIFY to MODIFY.

## Recommended next experiment

Freeze cases covering exact trace, absent/new trace, synonym labels, arbitrary
paths, and non-Go textual adapters. Evaluate current fixed weights versus
trace-as-bonus and explicit missing-signal handling using separately reviewed
ground truth. Select weights on a development partition; report held-out
Recall@k, Precision@k, MRR, and action correctness. Never renormalize only the
positive signals: doing so makes weak one-signal candidates look perfect.

No accuracy or cross-language generalization claim is made by this audit.

## Error diagnostics

Repository paths pasted with surrounding whitespace are trimmed. API failures
identify inventory, baseline parsing, new parsing, mapping or cancellation
without leaking filesystem paths or source excerpts. Embedding cancellation
now preserves the context error instead of flattening it into a runtime error.
The historical user-upload Hybrid failure is not conclusively reproduced;
successful Fusion alone never proves a graph-aware defect.

## Canvas workflow

The web uses top navigation for Mapping, BPMN Canvas, Saved analyses and Guide.
Upload a baseline in Mapping or Canvas, edit its independent copy, and select
“Use edited BPMN”. Then run analysis from Mapping. Two-file uploads remain
supported. Existing IDs and BPMN-DI are serialized by bpmn-js. Undo, redo,
zoom, fit, reset, download and reopen are available. Switching views preserves
the mounted editor and undo stack; Mapping warns when edits are not applied.
Backend parser validates submitted XML and references. Import warnings are
displayed; models without diagram coordinates may require layout first.

Drafts are currently memory-only: download before refreshing. Persistent drafts,
report highlighting, and richer pre-submit connectivity validation remain TODO.

Integration reference: https://bpmn.io/toolkit/bpmn-js/walkthrough/

## Verification

Go suite and vet passed; the subsequent cancellation change passed focused
semantic/API tests and vet. TypeScript, isolated production build, and diff
whitespace checks passed. Two browser editor tests cover rename/analyze/reset,
undo/redo/download, and adding task/gateway with exported diagram coordinates.
After the workspace navigation increment, all 10 serial Playwright scenarios
pass in one full run, including canvas-first upload, responsive layout, saved
history navigation, comparison, evaluation and review export. This is
functionality evidence, not mapping accuracy evidence.

## Dependency audit

Installing the pinned bpmn-js 18.16.0 identified two existing dependency
advisories: Next 16.3.5 (critical/high advisories) and source-map-js (high).
The audit suggested Next 16.4.0, but the configured npm registry returned
`ETARGET` for both 16.4.0 and the first non-affected 16.3.8 patch. No partial
upgrade was retained. Follow up when a compatible package is available, then
repeat build and regression checks before exposing the app beyond the local
demonstration environment.
