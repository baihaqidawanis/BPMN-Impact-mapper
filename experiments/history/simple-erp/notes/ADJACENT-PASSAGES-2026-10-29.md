# Positive-case evaluation and adjacent passage experiment

The baseline is production file-level Hybrid from
`outputs/staff-file-fusion-probe/<scenario>/run-2/impact-report.json`.
All five existing Staff cases and references in the consolidated manifest
remain unchanged. The candidate adds up to two query-independent source
passages immediately following the head and preceding the tail, preserving
the original head/tail/middle samples. Each content passage remains at most
600 Unicode characters; duplicate starts are removed. The global text budget
and round-robin allocation remain enforced. Plain Semantic/Fusion keep their
three-sample baseline. Candidate Hybrid records the passage policy in metadata.

Motivation from source inspection: the PR `CreateRequest` price guard occurs
around character 807 of a roughly 4105-character function, in a gap between
the existing sampled regions. This proposes better coverage without a
query-dependent file selector, business-specific rule or reference-driven boost.
More passages also create more chances for unrelated similarities, so keeping
the candidate depends on unchanged-reference ranking and recall results.

Adoption gate: retain the candidate only if positive-case MAP increases,
P@5/R@5 do not decrease, R@10 stays at baseline, and the independent negative
control stays empty. Report MRR and individual rank regressions even if that
gate passes. If it fails, keep the optional tested sampler but remove it from
production wiring. No cutoff, reference or weight is calibrated on these cases.

The CLI probe runs each scenario twice, compares prediction hashes and checks
input fingerprints against baseline. Only after inference is the reference
hash checked and the Go evaluator called. Both candidate and saved baseline
are evaluated with the new positive-ranking evaluator under separate output
directories in `outputs/staff-adjacent-passages/`; old reports are preserved.
The same MAP/R-precision scope must be used on both sides. These exposed
development cases do not establish independent generalization.

The evaluation upgrade adds positive-case MAP/MRR/R-precision/P/R@5/10 and
keeps legacy metrics separately. New tests hand-check a three-reference
example (AP=0.5889, R-precision=0.6667), negative-only undefined metrics,
weighted aggregation, and bounded passage coverage. The browser checks the
new metric rows, existing demo flow and saved-report evaluation.

## Outcome: rejected for production

All five scenarios completed twice with identical hashes and unchanged input
fingerprints/reference hashes. Both variants were evaluated with the same new
evaluator. The independent label-only control remains empty. Primary summaries
over the four positive changes:

| Metric | File-level baseline | Extra passages |
| --- | ---: | ---: |
| MAP | 0.7389 | 0.7327 |
| MRR | 0.8333 | 0.8125 |
| R-precision | 0.5834 | 0.5834 |
| Precision@5 | 0.4500 | 0.4000 |
| Recall@5 | 0.8750 | 0.7917 |
| Recall@10 | 1.0000 | 1.0000 |

Price validation store stays sixth, while its form drops 3 -> 4. The reason
task's store improves 5 -> 4, but confirmation's form drops 5 -> 6, losing a
top-five hit. Better raw text coverage did not improve retrieval on this set.
The adoption gate fails: remove `AdjacentPassages` from production wiring.
The bounded optional sampler and regression test remain for explicit future
experiments, default off; no public mapping method gains an extra UI option.
Do not revise labels or change k to hide this regression.

Detailed derived CSVs are `outputs/staff-adjacent-passages/comparison.csv` and
`summary.csv`. Per-case metrics are rounded before aggregation, so the last
decimal may differ from an unrounded mathematical mean. Changes to metric
scope are separate from retrieval performance; neither reference selection
nor rescoped MAP is claimed as an accuracy improvement.

Verification: Go full suite and vet pass; browser presentation suite 8/8 and
the follow-up R-precision UI case 1/1 pass; TypeScript passes. One ambiguous
legacy coverage locator was scoped to its details panel before the passing
rerun. Final production-wiring smoke verification is recorded in PLAN.md.

Final synthetic four-method benchmark after restoring production:
`f2df7a31a5e8bbfc9718482d73054dd7`, output
`outputs/benchmark-positive-metrics/`. Saved four-method Staff predictions
were re-evaluated without another embedding/inference run under
`outputs/current-method-evaluation/`. Hybrid uses the file-level production
baseline; the other three use unchanged consolidated baseline predictions.
The summaries use the same positive-case scope on both sides:

| Method | MAP | MRR | R-precision | P@5 | R@5 | R@10 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BM25 | 0.2375 | 0.3583 | 0.1250 | 0.1500 | 0.3750 | 0.4583 |
| Semantic | 0.3000 | 0.3500 | 0.1667 | 0.2000 | 0.4167 | 0.6667 |
| Fusion | 0.2202 | 0.2750 | 0.0833 | 0.2000 | 0.4167 | 0.7500 |
| Hybrid | 0.7389 | 0.8333 | 0.5834 | 0.4500 | 0.8750 | 1.0000 |

These are four positive development cases plus one separate empty control,
not a general accuracy claim. The metric addition does not itself improve
mapping. Current production ordering retains the prior file-level improvement;
the price backend still needs investigation at rank six. Any next query or
embedding experiment should preserve these references and add independent
unseen cases before claiming a broad gain. Source trees remain read-only at
232/6 pre-existing dirty entries. No commit or push was performed.
