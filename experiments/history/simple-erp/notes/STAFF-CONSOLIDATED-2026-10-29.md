# Consolidated Staff development benchmark

This combines five existing Simple-ERP Staff scenarios without changing their
BPMNs or reference labels: positive estimated PR price, mandatory inventory
submit reason, explicit PR total confirmation (v2), nonzero inventory delta,
and a display-label-only negative control. Four are positive requirements;
the last has an empty expected-file set. Supporting flow cases stay in the
canonical evaluation but are identified separately in the diagnostic CSV.

This is an exposed development set. References were authored from source by
one assistant, and are not implementation-verified or independently adjudicated.
The total-confirmation v2 query was sanitized after v1 had already been seen;
it is not a new blind case. The price reference evaluates existing files only:
a future new migration and treatment of existing zero-valued records are
outside its labeled targets. Unlabeled candidates are not automatically proven
irrelevant. No reference was relabeled in response to this benchmark.

## Run

From the mapper repository root:

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.gocache'
go run .\cmd\bpmn-impact-mapper benchmark --manifest experiments\simple-erp\staff-consolidated-2026-10-29-manifest.json --output outputs\staff-consolidated-2026-10-29-full --timeout 1h
.\experiments\simple-erp\summarize-staff.ps1 -OutputDirectory outputs\staff-consolidated-2026-10-29-full
```

The manifest records BPMN/reference SHA-256, top ten, four retrieval methods,
two repetitions and `single-source-review` provenance. The benchmark checks
the same per-scenario input fingerprint across methods. It reads reference
labels after inference. The Simple-ERP dirty tree is identified as
`c934497-dirty`; a commit alone does not freeze those local changes. At setup,
the five PR/inventory source files checked against the previous references
still had the same SHA-256 values. Never infer that all future local trees
with this revision label are equivalent.

The initial run in `outputs/staff-consolidated-2026-10-29/` reached the CLI's
default ten-minute whole-benchmark deadline during confirmation-task Fusion.
Its partial outputs are retained and are not counted as a complete benchmark.
The full rerun uses the identical manifest with an explicit one-hour deadline;
the timeout setting changes no retrieval method or reference label.

## Reading outputs

- `benchmark-report.*`: existing canonical aggregate over all atomic changes.
- `benchmark-timing.*`: observed run timing, including model startup/cache;
  sequential timing is not a fair standalone method-speed comparison.
- `change-diagnostics.csv`: each change, expected-file ranks (zero means
  absent), missed files, extra files and positive/supporting/empty grouping.
- `positive-change-summary.csv`: per-method macro averages over positive
  reference cases only, plus separate independent empty-case abstentions.

The summary script only reads completed prediction/evaluation artifacts,
checks their IDs/prediction hashes and writes supplementary CSVs. It does not
read source or feed labels back to inference. Precision uses a fixed denominator
of 5/10, matching the existing Go evaluator; recall divides hits by the case's
expected-file count. MRR averages existing per-change reciprocal ranks.
Empty cases have no positive-case Recall/MRR; they are counted separately.
These supplements do not replace the canonical metrics or redefine labels.

Added-gateway and removed-task fixtures remain in the separate synthetic
benchmark. They are not represented as real Staff source-reviewed cases here.
This batch changes no model, score weight, retrieval algorithm, or source in
Simple-ERP. Any next retrieval fix must be justified by error patterns and then
tested on separate cases rather than silently tuning this exposed set.

## Completed results

Benchmark ID `13a8e66389b53ac0b329e8f26fdddbe1`; canonical report SHA-256
`3a7bb46b6e500c085255d5bbc49f998be927b25218ec6d8dc9c0b9500905b190`.
All 20 scenario-method cells reproduced across two repetitions (40 analyses).
The full-coverage and frozen BPMN/reference checks passed. Positive-change
macro averages below use four positive requirements; the independent empty
control and five supporting flow records are excluded from these averages.

| Method | Precision@5 | Recall@5 | Precision@10 | Recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: |
| BM25 | 0.1500 | 0.3750 | 0.1000 | 0.4583 | 0.3583 |
| Semantic | 0.2000 | 0.4167 | 0.1500 | 0.6667 | 0.3500 |
| BM25 + Semantic | 0.2000 | 0.4167 | 0.1750 | 0.7500 | 0.2750 |
| Hybrid | 0.4000 | 0.7917 | 0.2500 | 1.0000 | 0.7083 |

All four methods abstained on the independent label-only control. Hybrid
retrieved all ten positive change/file reference pairs within ten, but 30 of
its 40 positive-case candidate slots are outside the reference. Recall@10=1
is not 100% mapping accuracy.

Metric interpretation: because these four positive cases have ten total
expected file/change pairs, 0.25 is also the maximum achievable fixed-denominator
Precision@10 for this reference. Increasing relevant items' positions can
improve P@5/R@5/MRR, but cannot raise P@10 once all expected files are within
ten. Precision among an adaptively returned set would be a different measure.

No method generates proven MODIFY decisions:
retrieval candidates remain VERIFY, so required-action accuracy against the
MODIFY reference labels is zero and must not be merged with retrieval recall.

### File misses and ordering

| Scenario | BM25 expected-file ranks | Semantic ranks | Fusion ranks | Hybrid ranks |
| --- | --- | --- | --- | --- |
| Inventory required reason: store / integration / form | absent / absent / 10 | absent / absent / absent | absent / absent / 10 | 5 / 2 / 3 |
| Inventory nonzero delta: store / integration | 3 / 4 | 5 / 6 | 9 / 3 | 4 / 1 |
| PR confirmation: dashboard E2E / operations E2E / form | absent / absent / absent | 1 / 2 / absent | 3 / 4 / absent | 2 / 1 / 6 |
| PR positive price: store / form | absent / 1 | 6 / 5 | 8 / 3 | 8 / 3 |

The reason-task case supports retaining BPMN context: Hybrid finds the three
targets where the other methods miss most/all. Confirmation reveals remaining
ordering trouble: the form is sixth despite being a primary feature target.
Positive-price validation exposes cross-layer retrieval trouble: the backend
store is absent for BM25 and eighth for Hybrid/Fusion. Fusion does not improve
every individual rank, and there is no basis for choosing RRF/model weights
from these few exposed cases.

Next development focus: inspect these missed/late candidates' actual queries
and indexed passages, separating lexical-language mismatch from context or
passage loss. Test any candidate improvement as a versioned experiment against
this unchanged reference and separate held-out scenarios. Keep test candidates
when they are genuine targets; do not boost forms or penalize tests just to
improve these known ranks. User-created Canvas experiments can complement this
batch; keep their reference preparation separate from mapper predictions.

The post-inference summary was cross-checked against the previous frozen
gateway/control run (known Recall@5/MRR and grouping counts). `git diff --check`
passed. No production-engine change was made in this consolidation, so the Go
suite was not needlessly repeated. Simple-ERP/POC TA were read-only throughout;
their pre-existing dirty entries remain 232/6. No commit or push was performed.
