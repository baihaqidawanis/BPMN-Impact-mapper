# Controlled context study

Development experiment; no web default changes. Existing four positive cases
and one negative control retain their BPMN/reference SHA-256 from the consolidated
manifest. New local source inputs are identified by common per-case fingerprint;
this is a fresh run, not assumed identical to the historical web benchmark.

Variants, fixed before execution:

1. `full`: unchanged production Hybrid.
2. `no-context`: remove ChangeContext from both retrievers' queries; retain the
   original context in the output for inspection. Task meaning/documentation,
   model, file fusion, passage construction and top-k remain unchanged.
3. `no-start-end`: omit only direct neighbour start/end labels from lexical
   context. Semantic and remaining context stay unchanged.
4. `both-top10`: experimental output filter; retain only candidates whose
   recorded ranks are <=10 in both branches, from the original fused top ten.
   Does not refill discarded slots. This tests selective output, not better
   ranking; report reduced output and missed positives as well as fixed-k metrics.

All variants keep file-level RRF, k=60, top ten, conservative function selection,
same local embedder and two identical repetitions. Reference files are read
after predictions. Repetitions check byte-identical canonical predictions;
different variants have distinct method-config identities. Cached execution
time is not a performance benchmark. These are exposed development cases;
improvement here requires validation on new cases before deployment.

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'
go run ./cmd/bpmn-impact-context-study --output outputs/context-study-current --timeout 1h
```

Output directory must be new. `study-summary.json` is incremental; a complete run
has 20 scenario-variant rows and 40 predictions. Partial output is not a complete
study. Do not overwrite the saved web benchmark with these experimental rows.

Decision policy: prefer preserved Recall with improved AP/R-Precision and fewer
unsupported candidates across cases. Reject arbitrary similarity thresholds or
case-specific path/domain exclusions. Four positives cannot establish general
accuracy; retain negative-control diagnostics and unchanged provisional labels.

## Completed run and decision

`outputs/context-study-current`: 20 scenario-variant cells / 40 analyses; all
canonical prediction hashes repeat exactly. `summarize_context_study.py` checked
prediction/evaluation binding and common per-case inputs. Historical web snapshot
was not overwritten. Primary positive-case results:

| Variant | P@5 | R@5 | R@10 | MAP | MRR | R-Precision |
|---|---:|---:|---:|---:|---:|---:|
| full | 0.4500 | 0.8750 | 1.0000 | 0.7389 | 0.8333 | 0.5834 |
| no-context | 0.0500 | 0.1250 | 0.4166 | 0.1009 | 0.1361 | 0.0000 |
| no-start-end | 0.4500 | 0.8750 | 1.0000 | 0.7389 | 0.8333 | 0.5834 |
| both-top10 | 0.4000 | 0.7917 | 0.7917 | 0.6472 | 0.8333 | 0.5834 |

No-start-end produced no improvement in these aggregate metrics. Do not enable
it by default: event labels can contain useful business meaning in other models.
Both-top10 reduced selected candidates across positives from 40 to 19 and raised
selected-candidate precision from 0.25 to 0.4211, but missed expected files:
Recall@10 fell to 0.7917 and MAP fell. Reject it as the production filter.

No-context is a deliberately broad ablation: it also removes BranchTerms, which
may carry the actual changed gateway condition. Its aggregate deficit must not
be attributed solely to neighbour labels or process-name enrichment. On the
task-addition inventory case (no branch terms), removing context reduced AP
0.7556 -> 0.0333 and Recall@10 1 -> 0.3333. The price-label case had no structural
context and remained unchanged. A finer future ablation should keep changed
branch requirements while independently switching process/neighbour enrichment.
This is not evidence that the parser or BM25 inherently understands business
semantics, nor generalisation beyond these exposed cases.

Decision: keep current production Hybrid. Next accuracy work should inspect
query ambiguity and source passage relevance or review candidates with bounded
evidence. Adding a hard rank cutoff is not justified by this study.
