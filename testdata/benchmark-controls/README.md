# Synthetic control pack (not case-study accuracy)

`scenario-matrix.json` freezes four BPMN comparisons over the existing
synthetic `testdata/repository`: task insertion, task removal and rewire,
gateway-branch addition, and a display-label-only negative control. Every
BPMN file has a SHA-256 lock and every comparison has an expected
method-independent input fingerprint. `TestFrozenSyntheticScenarioMatrixAcrossMethods`
runs all four retrieval methods twice and checks taxonomy, fingerprints,
deterministic canonical JSON, relative evidence paths, and the expected
abstention/candidate shape. Its constant test embedder **does not measure
semantic quality**. Candidate presence is not a relevance label.

`manifest.json` is a separate, executable frozen benchmark for the one
fixture-authored negative control. The original flow has an explicit condition;
only its display label changes, and no allowed source artifact refers to the
flow ID or either label. `ground-truth-label-only.json` therefore has an empty
expected set. It lives outside the analyzed repository and is read only after
inference. Run from the mapper root:

```powershell
$env:GOCACHE=(Join-Path (Get-Location) '.gocache')
go run .\cmd\bpmn-impact-mapper benchmark --manifest testdata\benchmark-controls\manifest.json --output outputs\benchmark-flow-label-control
```

The benchmark repeats BM25, Semantic, BM25 + Semantic, and Hybrid twice with
the same input fingerprint. A correct negative produces `0` candidates and
`1/1` correct abstention for each method. Its `unresolved_rate` is also `1`:
that field describes **how often the mapper abstained**, not whether abstention
was right. With zero expected positive files, Recall/MRR and file F1 are not
meaningful accuracy comparisons. No embedding runtime is invoked for this
direct-reference-free case.

This pack is a functional regression control, not independent research ground
truth. The task/gateway matrix currently has no adjudicated file-level truth;
do not report Precision/Recall for those cases. A frozen implementation diff or
blind reviewer reconciliation remains necessary for Simple-ERP evaluation.
