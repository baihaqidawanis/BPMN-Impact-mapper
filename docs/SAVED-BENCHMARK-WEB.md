# Saved benchmark results in Research

Research now shows a dated read-only summary for the completed Staff
development cases. The public JSON is
`web/public/benchmarks/staff-2026-10-29.json`, generated from
`outputs/current-method-evaluation/summary.csv` and cross-checked against the
saved per-case Go evaluation reports. Its source CSV hash is recorded in JSON.
The Hybrid row uses file-level production ranking; other rows use the unchanged
consolidated saved predictions, as described in the snapshot's ranking policy.

Four positive cases supply MAP/MRR/R-precision/P@5/R@5/R@10 averages; the one
independent empty control is reported separately. The source-review references
are provisional. The revision is a dirty-tree label, not a complete immutable
source archive. No speed, Qwen, independent-review or general-accuracy result is
claimed. Displaying this snapshot neither starts analysis nor changes scores
when the current repository changes. New mappings still need their own
reference-based evaluation; the web scenario queue is not a repeated benchmark.

`BenchmarkSummary.tsx` owns presentation and accessible table/definitions;
JSON owns the saved data. Users can download the same JSON. No backend API,
retrieval method, reference file or source repository was modified. There is
no expected-file list in the public snapshot and it is not mapping input.

Verification: all published values match saved positive-case evaluations;
TypeScript and Playwright benchmark-summary/batch tests (4/4) pass. Tests check
downloaded data, dated scope, independence from the selected repository, no
analysis POST and mobile horizontal containment. Desktop screenshot inspected.
The Research empty state is compact and explains how to select a report.

## Retrieval experiments, 30 October

The same Research panel now includes a separate collapsed table for the newer
Hybrid retrieval experiments. `web/public/benchmarks/retrieval-studies-2026-10-30.json`
contains seven per-case rows (two baselines plus five experimental comparisons),
32 repeated analyses, provisional-reference status and source-summary hashes.
These are not new four-method benchmark averages or independent validation.
No experimental variant was adopted. The historical table remains unchanged.

Rebuild from frozen analysis/evaluation artifacts, without running inference:

```powershell
python experiments/simple-erp/publish_retrieval_studies.py
```

The publisher verifies reference/BPMN hashes, repeated prediction/evaluation
bytes, evaluation metrics and matching baseline identities before publication.
Running a new user mapping does not update either table. Live precision/recall
still requires a separately prepared expected-file reference in Evaluation.
