# Evaluation design

## Positive-reference ranking (2026-10-29)

`metrics.positive_ranking` reports MAP, MRR, R-precision and P/R@5/10 only
for changes with nonempty expected-file sets. `positive_changes` states the
denominator. Empty-reference controls retain their abstention evaluation;
they have no defined R-precision and do not dilute positive-case MAP.
Per-change `r_precision` is omitted for empty sets. Benchmark method aggregates
weight these summaries by positive-change count, not by total scenario count.
JSON, evaluation/benchmark Markdown, benchmark CSV and the Research dashboard
expose the new scope. Historical reports without it render n/a.

R-precision uses the top R unique files, where R is that change's reference
file count. AP averages precision at relevant ranks, dividing by all expected
files (missing files contribute zero); MAP averages AP. P@k retains a fixed
k denominator. Legacy top-level MAP/MRR/P@k/R@k continue to include all labeled
atomic changes for compatibility and are labeled separately in reports.
Evaluation IDs use `positive-ranking-v1` in their identity seed. Adding or
rescoping a metric is not a retrieval improvement; compare before/after
predictions using the same evaluator and unchanged references.

For the current four positive Staff cases with ten expected change/file
pairs, P@10 cannot exceed 0.25 and P@5 cannot exceed 0.5. This ceiling follows
from reference counts, not model capability. Do not change labels or shorten
the displayed list merely to inflate a metric. A separately calibrated
variable-length output needs precision/recall measured together.

## Fair comparison

The target retrieval comparison is BM25-only, Semantic-only, BM25 + Semantic RRF, and BPMN-Context-Aware Hybrid Retrieval. Each must receive the same frozen repository snapshot, old/new BPMN pairs, scenario cards, Top-K, and output schema. LLM planning is evaluated separately after retrieval so generation quality cannot hide retrieval errors.

From engine 3.5d, each public method displays one best artifact per file before
Top-K. The evaluator scores file-level ranks, so this avoids giving one method
fewer distinct file slots merely because several functions in one file ranked
highly. Internal RRF branches still rank artifacts before fusion. Do not mix
older artifact-Top-K reports with file-Top-K reports in one comparison.

## Artifact separation

- method input: allowed repository source, old/new BPMN, frozen method configuration;
- baseline configuration: ignores, adapters, mappings, signal weights, and thresholds fixed before evaluation;
- excluded corpus: process/traceability maps, checkpoints, audit/release/implementation plans, reports, ground truth, and implementation diffs;
- hidden ground truth: expected artifact, module/layer, and required action established after scenario implementation;
- evaluation artifacts: predictions, timings, metrics, and reviewer reconciliation.

Stable BPMN task IDs already present in source are valid method input. Ground-truth files, scenario implementation branches/diffs, and generated reports are never retrieval input.

## Ground truth

Use a frozen scenario-change × artifact unit. Prefer two independent reviewers who record file, optional symbol, module/layer, action, evidence, and assumptions before reconciliation. Report agreement (and kappa when sample size supports it) plus unresolved adjudications. Negative controls must be retained; an unobserved or unretrieved artifact is not automatically `NO_ACTION`.

## Metrics

- Recall@1/@3/@5 and Precision@1/@3/@5 on unique paths;
- MRR, and MAP when multiple relevant artifacts have binary relevance;
- file, module, and layer accuracy;
- required-action accuracy on retrieved expected files, plus macro-F1 and a
  confusion matrix where an expected file not retrieved becomes `UNRESOLVED`;
- unresolved/abstention and false-positive rates;
- latency plus evidence/index preparation time;
- repeated-run byte-for-byte reproducibility;
- optional per-layer and per-change-type breakdown.

Recall@k must not be mislabeled as Hit@k. Ranking metrics do not prove implementation correctness.

An empty expected-file set is a labeled negative control. Evaluation now
reports its count, how many such cases returned no candidate, and
`negative_control_accuracy = correct_abstentions / negative_controls` (shown
as n/a in human-readable output when there are none; the typed JSON field is
zero in that case and must be interpreted with `negative_controls`). This is
separate from `unresolved_rate`, which
counts abstention frequency even when abstention was correct. The existing
`false_positive_rate` JSON key is actually FP/(TP+FP), a false discovery
proportion, not statistical FPR; the Markdown/CSV labels say so. An all-negative
fixture can have negative-control accuracy 1 and unresolved rate 1 at the same
time. Recall/MRR/File F1 are not informative when there are no expected
positive files.

Phase 2A implements these metrics in `internal/evaluation`. Ranking metrics are macro-averaged by ground-truth case; file precision/recall/F1 are micro-aggregated over unique file paths. Prediction files are deduplicated per change before scoring. Module/layer/action accuracy is evaluated only for expected files that were retrieved. The required-action confusion matrix instead includes every expected file and records a missing retrieval as predicted `UNRESOLVED`; macro-F1 averages only action classes that have ground-truth support. `Precision@k` uses `k` as the denominator even when fewer candidates are returned. Negative-control cases may have an empty expected set and still contribute false positives.

Canonical prediction and ground-truth byte hashes bind every evaluation artifact. The evaluation ID is deterministic from those hashes. Latency remains zero in Phase 2A because volatile timing is deliberately kept out of canonical inference reports; controlled experiment orchestration should record wall time separately in a later tranche.

## Frozen benchmark orchestration

Phase 2B adds a versioned manifest with repository metadata and SHA-256 locks for BPMN pairs and hidden ground truth. Every primary manifest must include BM25, Semantic, BM25 + Semantic RRF, and BPMN-Context-Aware Hybrid Retrieval; repetitions must be between 2 and 10. Configuration-driven mapping is an optional legacy comparator and requires a frozen mapping path plus SHA-256 when selected. The report's method-independent `input_fingerprint` hashes repository metadata, allowed repository contents, both BPMNs, and ingestion configuration. A run fails when fingerprints differ across methods, an optional expected fingerprint does not match, or repeated canonical prediction/evaluation hashes drift.

Ground-truth bytes are parsed only after the corresponding inference call returns. Timing uses a separate non-deterministic artifact and does not affect benchmark IDs or metric reports. Method aggregates are macro means across scenarios; counts remain summed.

`benchmark-comparison.md` and `.csv` present Recall@5, MRR, File F1, and
per-method min/median/mean/max wall-clock latency in one table. Latency remains
explicitly volatile and excluded from prediction hashes and benchmark identity.
`benchmark-ablation.md` and `.csv` use Hybrid as a paired aggregate reference
and report deltas for BM25, Semantic, and Fusion. They describe the frozen
scenarios only and do not establish causal contribution.
The batch runner requires ground truth for every changed element unless a
scenario explicitly declares `truth_scope: "subset"`. Subsets must refer only
to real changed elements, are marked diagnostic in JSON/CSV/Markdown, and are
excluded from method aggregates. At least one complete scenario is required.
For an empty diff, complete truth must also have no cases. This prevents
publishing method aggregates from only the convenient changes.
An optional manifest `label_provenance` value (`fixture-authored`,
`single-source-review`, or `adjudicated`) is copied into the JSON/Markdown/CSV
reports. It is a declared research label, not proof of reviewer independence.
The Staff source-only probe uses `single-source-review`: its expected files
were recorded before opening the current run's ranking, but the assistant is
one reviewer and has worked on this mapper before. The resulting figures are
diagnostic, not final TA accuracy.
The [positive/negative synthetic control pack](../testdata/benchmark-impact/README.md)
exercises task insertion, gateway addition, and abstention across four methods.
Its fixture-authored truth is useful for regression diagnostics, not a blind
Simple-ERP accuracy claim. Supporting sequence-flow edits are counted as
separate negative cases, so aggregate Recall@5 must be read alongside the
per-change rankings and abstention counts.
The current runner is shared-process and sequential: the first embedding method
includes cold model startup while later methods may reuse embeddings. Therefore
these timings diagnose runtime behavior but are not yet a fair standalone
method-speed comparison; a cold/warm isolated protocol is required for that
claim.
The vocabulary-mismatch fixture is a synthetic control for a task whose expected
source uses different wording and no BPMN trace ID; it must not be presented as
independent empirical evidence.

## Independent review

The CLI `prepare-review` workflow reduces anchoring bias by creating reviewer artifacts from one or more completed same-snapshot reports without exposing any predicted file, symbol, method, evidence, rank, confidence, or score. Reviewers receive only a BPMN change guide and an individual annotation template. A coordinator-only manifest binds the repository revision, method-independent input fingerprint, normalized change-set digest, prediction methods, analysis IDs, and report byte hashes. Report argument order does not affect the pack ID or bytes.

The web blind-review panel uses the identical Go use case against immutable stored reports. Its endpoint accepts only report IDs, their shared fingerprint, and two reviewer identifiers; it does not accept annotations, expected impacts, or ground truth. The browser downloads both templates, the guide, and the coordinator manifest separately so distribution roles stay explicit.

Templates fail closed: every case initially contains a `REVIEW_REQUIRED` entry with an invalid empty action. A reviewer must replace it with independently established expected impacts, or explicitly remove it and use an empty list for a negative case. Therefore, two untouched templates cannot silently reconcile into false negative ground truth. The pack directory is write-once at CLI level. Passing its coordinator manifest to `reconcile --manifest` validates the pack ID, case digest, canonical prediction metadata, and exact case coverage for both completed annotations; the resulting JSON and Markdown retain pack ID, input fingerprint, and case digest. The manifest is provenance metadata, not encryption, a digital signature, or proof that the people were independent; the coordinator must keep it separate from reviewers and retain completed files with the research archive.

The web dashboard exposes reconciliation and evaluation after inference. It accepts two version-1 annotation files (512 KiB each in the browser; 2 MiB total API body), recomputes reconciliation on evaluation, and requires exactly one explicit resolution for every disagreement. Missing reviewer cases remain conflicts. Final truth must cover every change exactly once; additional or missing cases reject the evaluation. The server reads at most three immutable reports with unique methods and one input fingerprint. Changing reviewer files or decisions invalidates displayed results.

Version-1 annotations do not encode a snapshot fingerprint. The user must confirm they apply to the selected snapshot; matching case IDs alone cannot prove provenance or reviewer independence. Distinct reviewer IDs are a validation rule, not evidence that two people reviewed the scenario. The downloaded bundle retains both annotations and decisions for audit. It is not persisted by the API.

The legacy JSON field `false_positive_rate` is FP/(TP+FP), so the dashboard labels it **false discovery proportion**. A true-negative universe is unavailable; statistical FPR and file-level accuracy are therefore not displayed. Module/layer accuracy counts unspecified expected metadata as matching under the existing evaluator; the dashboard discloses this limitation. Precision@k retains denominator k; negative expected sets contribute zero to recall/MRR/MAP macro averages. These are file-level metrics: multiple symbols in one file are deduplicated, and the first prediction supplies its action. No new latency measurement is made by this dashboard.

Two versioned reviewer annotations use the same case/expected-impact contract. Reconciliation requires distinct reviewer identifiers, canonicalizes expected-impact order, and accepts a case only when both reviewers provide equal impacts and actions. All missing or conflicting cases remain in `disagreements`. The dashboard supports explicit human decisions through `review.Adjudicate`; it never silently chooses a reviewer.

## Phase 1 fixtures

`testdata` is synthetic and neutral. It exercises task, actor, gateway, flow, data, state, and terminal changes against Go and TypeScript-like text without copying either external study repository. External case-study results must be reported separately and cannot establish universal generalization.

The included four-method run is only a functional smoke benchmark. Its synthetic ground truth is fixture-authored and must not be reported as independent case-study evidence.

The additional [synthetic control pack](../testdata/benchmark-controls/README.md)
locks four task/gateway/flow comparison pairs for cross-method taxonomy and
reproducibility tests. Its executable benchmark contains one fixture-authored
label-only negative control. The matrix uses a constant test embedder, so it
cannot evaluate semantic retrieval quality or file-level accuracy for task and
gateway scenarios.

## Case-study pilot inputs

Tracked files under `experiments/` are frozen scenario and baseline-configuration inputs, not product behavior or ground truth. Generated BPMN and prediction reports remain ignored artifacts under `outputs/`. A mutation specification binds to the original BPMN SHA-256 and must apply each declared operation exactly once. The generated document must pass the same bounded parser used by inference.

Pilot predictions can validate execution, taxonomy coverage, evidence presence, path hygiene, and reproducibility automatically. Relevance labels, required-action truth, completeness, inter-reviewer agreement, and adjudication remain manual research steps. No metric from a case-study pilot is valid until the hidden ground truth has been independently reviewed and reconciled.
