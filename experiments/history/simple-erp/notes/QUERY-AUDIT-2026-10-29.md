# Pre-presentation query audit

Scope: read-only inspection of the current price-validation query code, source
and saved file-level Hybrid report `c608e159775b519ff2d5364229391183`.
No ranking, model, threshold, runtime or demo input is changed in this audit.

## Observed

- The change is `MODIFY_TASK_LABEL` on `role_staff_l3_pr_create_check`.
  Its `context` is null. Current diff annotation provides process context for
  selected topology/gateway changes, not this label modification, so Hybrid's
  process-name enrichment is not applied to this case.
- The lexical query retains both `nonnegatif` and the new `lebih dari nol`.
  Tokenization does not translate `harga` into `price`, `qty` into `quantity`,
  or `produk` into `product`. The source guard is `l.UnitPrice < 0` and the UI
  guard is `unitPrice < 0`; their intended change is stricter zero rejection.
  This lexical difference is observable; its contribution to model ranking
  has not been isolated experimentally.
- Both old and new labels and the same documentation enter the semantic
  query. The repeated text includes the navigation reference
  `Setelah kontrol akses HTTP; lihat l3-kontrol-request.` This is broader than
  the changed price requirement. It is not evidence of an implementation
  dependency or a proof that these words caused the ranking problem.
- In the saved report, the form is lexical rank 1 / semantic rank 5 and final
  rank 3. The purchasing backend is lexical rank 12 / semantic rank 6 and
  final rank 6. E2E creation flows are final ranks 1 and 2. They share process
  vocabulary but are outside this scenario's existing-file reference: their
  current valid-price flows need not change to enforce the new zero boundary.
- The earlier adjacent-passage experiment already failed to improve this
  backend position and regressed other ranks; it remains off in production.

A temporary ignored Go diagnostic reconstructed the query from these element
fields using the current tokenizer and confirmed null context. The JSON in
`outputs/pre-presentation-audit/price-query-reconstruction.json` explicitly
labels this as static reconstruction, not an internal runtime query trace.
No new scenario prediction or reference label was used to tune a query.

## Hypotheses to test after the presentation

1. Supply process context consistently for label changes, with a bounded
   process name derived from BPMN alone. Check whether generic validation
   vocabulary then stays within the intended business process.
2. Separate changed requirements from unchanged/duplicated label/documentation
   context. Preserve old meaning for matching existing code, rather than
   simply dropping old labels or adding scenario-specific translations.
3. Compare lexical-language and semantic-model limitations on additional cases;
   do not assume that manually translating the known price case proves a
   general retrieval improvement.

Each variant should be versioned, compared against the same untouched
references and tested on a fresh operation. Do not penalize all tests or boost
the known backend file to move this one target into five. Retain the current
production file-level Hybrid for the presentation.

## Fresh unscored case

`staff-inventory/cancel-reason-min10/` contains an exact copied Staff L3
baseline, a deterministic mutation specification, the new BPMN and frozen
hashes. Its separate source-only reference was authored before any mapper
inference for this case. It changes cancellation validation, not a task count,
and is not yet appended to the existing scored benchmark.
