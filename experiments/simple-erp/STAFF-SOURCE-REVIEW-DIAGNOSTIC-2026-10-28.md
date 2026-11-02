# Staff source-review diagnostic — 2026-10-28

This is **single-review diagnostic evidence**, not independently adjudicated
TA ground truth. The expected files in
`docs/ground-truth/staff-source-review-*-2026-10-28.json` were written from
BPMN and source inspection before the current rankings were opened. This
reviewer also developed the mapper and may have prior methodological exposure.
The hypothetical changes have no implementation diff. The manifest declares
`label_provenance: "single-source-review"` in exported JSON, Markdown, and CSV.

Input: read-only Simple-ERP HEAD `c9344971a573d5c44d9d4e58f23657eda4d18a86`
plus 232 pre-existing dirty-status entries. This is a local snapshot, not a
reproducible Git commit. The manifest fixes both BPMN SHA-256 hashes, both
reference hashes, and each method-independent input fingerprint. The runner
rejected mismatches, compared four methods on the same inputs, and checked
byte-identical reports across two repetitions per method.

Command from the mapper root:

```powershell
$env:GOCACHE = (Join-Path (Get-Location) '.gocache')
go run .\cmd\bpmn-impact-mapper benchmark --manifest experiments\simple-erp\staff-source-review-2026-10-28-manifest.json --output outputs\staff-source-review-2026-10-28
```

Benchmark ID: `52224287eddd2285159b504cb07c5110`. Eight scenario-method
results were reproducible. The generated report is under the ignored
`outputs/staff-source-review-2026-10-28/` folder; its timing is a separate
volatile artifact and should not be used for a fair speed comparison.

| Method | Inventory gateway: store file | Inventory gateway: integration test | Supporting flow candidates | PR label-only candidates |
| --- | ---: | ---: | ---: | ---: |
| BM25 | absent | absent | 10 | 0 |
| Semantic | 5 | 6 | 5 | 0 |
| BM25 + Semantic | 5 | 6 | 10 | 0 |
| Hybrid | 4 | 1 | 10 | 0 |

Ranks are file positions on `MODIFY_GATEWAY_CONDITION` with `top_k=10`. Under
the frozen two-file reference, Hybrid has main-change Recall@5 = 2/2;
Semantic and Fusion have 1/2; BM25 has 0/2. The benchmark's scenario-level
Recall@5 is lower because it averages the gateway with its supporting
`MODIFY_SEQUENCE_FLOW` record, which has an empty independent expected set.
All four methods correctly abstained for the PR branch-label-only negative
control. The inventory supporting flow generated false positives and should
not be read as a separate set of necessary edits.

The prior source-derived inventory reference in
`docs/ground-truth/staff-inventory-nonzero-delta-reference.json` includes the
backend store but does not label the integration test as a required retrieval
target. My frozen reference includes the test because the new submit guard
needs a regression case in the existing integration suite. That is a labeling
disagreement about whether tests count as required changed files; it is not
resolved by the mapper score. Both references remain unchanged. Formal metrics
require a stated file/test policy and separate adjudication.

The BPMN source contains process-level documentation naming an implementation
file. The current parser attaches `documentation` only to a current BPMN
element; it does not attach process-level text to a query element, and the
indexed corpus excludes BPMN specification trees. Future query changes must
preserve this boundary or sanitize such metadata before formal experiments.
No method weights were changed from these diagnostic results.
