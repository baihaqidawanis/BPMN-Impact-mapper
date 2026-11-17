# Gateway query and supporting-flow correction

This is a diagnostic follow-up to `STAFF-SOURCE-REVIEW-DIAGNOSTIC-2026-10-28.md`.
It uses the same frozen Staff BPMNs, source fingerprints, two-file inventory
reference, negative control, four methods and two repetitions. Labels remain
single-source-review, not independently adjudicated implementation truth.

## Cause and implementation

Plain BM25 deliberately removed process/neighbour context, but this also
removed the conditions of the gateway being changed. BPMN stores those
conditions on outgoing flows, so the remaining gateway label alone could
produce no lexical candidates. BM25 now retains direct branch labels and
conditions. It still excludes process/neighbour expansion. No ranking weights,
RRF constants, model, candidate limit, reference files or case-study source
were changed. No new retrieval method was necessary.

For an existing flow with unchanged endpoints, changing its effective
condition also creates a gateway condition change. The flow is now marked
`part_of` that gateway, remains in canonical JSON, and avoids a duplicate
candidate list. Added/removed edges, changed endpoints, and separate
display-only edits remain independent. A source reference to an exact flow ID
or eligible multiword label is separately verified against matching artifacts.
Both whole-file and function-level results retain this reference evidence.

Changed and deleted branches contribute their old/new terms. Added/removed
gateways contribute their outgoing branch terms. Conditions take precedence
over labels for change detection; both are useful for retrieval. Analysis
metadata records `gateway-condition-flow-grouping-v3-branch-labels` and the
plain BM25 query policy. The web/Markdown supporting-flow text now describes a
generic BPMN change rather than always claiming an inserted task.

## Preserved unsuccessful intermediate run

`outputs/staff-gateway-fix-2026-10-28/` (benchmark
`7f6bfaf683ee643409863ea858d73f76`) retained only conditions, dropping the
explanatory branch label. BM25 found the two reference files at ranks 1/2,
but Semantic regressed from 5/6 to 6/absent within ten. This revealed a loss
of meaningful branch annotation, so the final implementation retains labels
as well as expressions. This is a development diagnostic, not held-out tuning
evidence; the unsuccessful output is preserved rather than overwritten.

## Verification scope

Verification closed on 2026-10-29. Final synthetic benchmark
`73ff0e055d61de0e1c45795bb49c0bb0` is under
`outputs/benchmark-impact-gateway-fix-labels/`; report SHA-256:
`d4081b6ffdba2c78fb19f6c51bfa16ea7c8b4d7915554c4b71ea3625ecee9199`.
All twelve scenario-method results reproduce over two repetitions (24
analyses). Primary expected-file positions match the earlier
`outputs/benchmark-impact-coverage/` run: task form/test ranks are BM25 2/4,
others 2/3; gateway implementation ranks first in every method, while its test
is absent for BM25/Semantic/Fusion and rank six for Hybrid. These unchanged
misses remain visible; the correction does not solve general test retrieval.

Final Staff command:

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.gocache'
go run .\cmd\bpmn-impact-mapper benchmark --manifest experiments\simple-erp\staff-source-review-2026-10-28-manifest.json --output outputs\staff-gateway-fix-labels-2026-10-28
```

Benchmark ID: `e9e6a54b78a74174efec3530afd436ea`; report SHA-256:
`b835882ce913772bf3a361d847b269574ec6a506ec9201431738ad62a909599a`.
All eight scenario-method results reproduce over two repetitions (16 analyses).
Frozen input fingerprints and truth hashes passed validation.

| Method | Store rank before -> after | Integration test before -> after | Supporting flow candidates before -> after |
| --- | --- | --- | --- |
| BM25 | absent -> 3 | absent -> 4 | 10 -> 0 |
| Semantic | 5 -> 5 | 6 -> 6 | 5 -> 0 |
| BM25 + Semantic | 5 -> 9 | 6 -> 3 | 10 -> 0 |
| Hybrid | 4 -> 4 | 1 -> 1 | 10 -> 0 |

The PR display-label negative control remains zero candidates for every
method. Main-gateway Recall@5 is now BM25 2/2, Semantic 1/2, Fusion 1/2,
Hybrid 2/2. These are per-primary-change counts, not the benchmark's macro
Recall@5, which also averages the empty supporting-flow case. At ten, each
method finds both reference files: file precision is only 2/10. Several
non-reference candidates remain, and Fusion's store position got worse.
Grouping reduces duplicate false positives; it does not improve primary
Hybrid ranking. Required-action accuracy remains zero against the reference's
MODIFY labels because retrieval conservatively returns VERIFY. None of these
results justify a claim of general accuracy or weight calibration.

Go full-suite tests, vet, changed-file gofmt, web TypeScript checking and
`git diff --check` pass. Browser interaction was not re-tested for the one-line
supporting-flow wording change in isolation. Follow-up on 2026-10-29:
`npm run test:e2e -- report-results.spec.ts report-explanation.spec.ts`
passed 8/8 tests (35.4 seconds), including browser report filters, file view,
desktop/mobile checks and ranking-explanation/grouping checks. Initial execution
failed with sandbox `spawn EPERM`; the same command passed with local process
permission. This uses fixture reports and does not prove a full real-repository
analysis through the browser. No dependency download or external API was used.

Regression coverage includes condition-only edits, branch-label surrogates,
display-only labels with unchanged explicit expressions, rewiring,
added/removed branches, added/removed gateways, no neighbour expansion in plain
BM25, grouped raw diffs, direct source references, and whole-file selection.
The previous test expecting a duplicate candidate list on a condition flow
now checks the gateway candidate list and the retained supporting flow.

The tests do not prove real-model correctness or browser usability. This
change does not activate an external LLM, modify Simple-ERP/POC TA, or provide
independent ground-truth adjudication. Candidate actions remain `VERIFY`;
retrieval agreement does not prove that source must change.
