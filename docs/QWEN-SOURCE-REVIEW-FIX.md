# Qwen source review: 30 October 2026

## Failure reproduced

The original inventory session stopped after its planner turn. A diagnostic
replay reproduced the exact validation failure: the planner repeated the same
evidence IDs within one file plan. The native Ollama structured grammar did not
enforce JSON Schema `uniqueItems` reliably.

The adapter now treats references as an ordered set. Only identical repetitions
are removed. Unknown IDs, fabricated targets, invalid actions and role violations
are still rejected by the planning engine. Validation feedback now includes the
actual contract error instead of only a generic failure message.

## Repository access

The Go backend reads the approved local repository and builds an immutable source
snapshot. The lab model receives bounded excerpts through the SSH tunnel. A ZIP
upload is unnecessary while backend and repository are on the same laptop.

The source tools can search the allowed repository beyond initial candidates;
three initial candidates are seed evidence, not the full searchable corpus.
Excluded secrets, evaluation references, plans and generated artifacts remain
outside that corpus. The model does not receive every repository file in one
prompt and is not guaranteed to inspect every file in one session.

`read_source` now reads an eight-line window from an approved relative file.
It works with Go and generic textual source such as TSX. `start_line` pages the
file; `query` locates a matching line and centers long-line excerpts around the
match. Each excerpt remains limited to 240 characters, at most eight per result.
`next_line` indicates where to continue. This is textual inspection, not TS AST
support. Go method names such as `Store.Save` are accepted by `read_function`.

The Ollama analyst protocol requires a source-window lookup for candidate files
before final decisions, within the existing budgets. It starts around an actual
evidence location; subsequent searches and window reads remain available.
Later windows may replace older tool evidence within the candidate's eight-item
limit. Initial retrieval evidence is retained. Planner instructions require
proposed behavior changes and distinguish static assertions from executed tests.

## Evidence and limitations

Diagnostic artifacts remain under `outputs/erp-agent-diagnostic-*`, outside
the inference corpus. The failed original user checkpoint is preserved. Replays
use the selected change and source, never the evaluation reference as model input.

Replay `20261030T125814` reproduced duplicate-ID rejection. Replay
`20261030T125935` subsequently reached all three roles and exported a Markdown
draft with `NEEDS_REVIEW`, 13 turns and four tools. Later replays exercise source
windows and preserve their findings rather than claiming independent accuracy.

The latest inventory replay is `outputs/erp-agent-diagnostic-20261030T130819`:
17 turns, eight tools, all three roles, no engine error, and a saved
`change-plan.md` (plan ID `853bcdb5224d8861367878d7ad5fd988`). It is still
`NEEDS_REVIEW`: the final draft remains too focused on verification, and still
fails to establish the existing reason-input behavior reliably. This is a
successful technical recovery, not a successful plan-quality acceptance test.

A saved draft is not an approved implementation plan. The 9B model can still
infer absent behavior from insufficient snippets, propose redundant inputs or
broaden validation beyond the changed action. Reviewer issues must remain visible;
there is no forced approval and no claim that reading snippets proves exhaustive
repository coverage. Source is never executed or modified.

Checks: application Go package tests, vet, window paging/path/exclusion tests,
compressed-line matching, immutable snapshot behavior, Go method lookup,
reference normalization and rejection of fabricated evidence. The three inventory
source-file hashes still match the pre-existing source review.

After updating the backend, start a new Qwen review. Previous checkpoints keep
their original model/source/policy binding and are not silently rewritten.
