# Ground-truth protocol

## What ground truth means

Ground truth is the independently established list of repository files that
should be affected by each BPMN change, including the expected action. It is
created only after mapper predictions have been frozen. A candidate returned by
the mapper is never ground truth merely because it has a high score.

The same ground truth must evaluate BM25, Semantic, Fusion, and Graph-Aware
Hybrid. Creating different labels for different methods invalidates the
comparison.

## Evidence strength

Use the strongest feasible source:

1. **Closed reference implementation (preferred).** Freeze the mapper reports,
   implement the scenario on an isolated branch, and record the reviewed
   implementation diff. Changed files are a starting point, not automatic
   truth: remove incidental formatting/refactoring and add required files the
   implementation accidentally omitted.
2. **Two independent architecture reviews.** When no implementation exists,
   give two reviewers the repository snapshot, old/new BPMN, and a precise
   requirement card without mapper candidates. Each reviewer lists expected
   files and actions. Explicitly adjudicate every disagreement.
3. **Retrospective co-change commit.** Use a historical change only when its
   parent/child revisions contain a relevant BPMN change and the corresponding
   implementation change. Exclude unrelated files and document that filtering.

Do not present an expert-review label as an observed implementation diff. Both
are useful, but they answer different questions.

## Per-scenario procedure

1. Write a requirement card describing the intended behavior and explicit
   non-goals. A label such as "add validation task" is insufficient unless the
   validation behavior and rejection/error outcome are defined. Start from
   [`docs/templates/SCENARIO-CARD.md`](templates/SCENARIO-CARD.md).
2. Freeze repository revision, old BPMN, new BPMN, ignore configuration, Top-K,
   and all method versions.
3. Run every mapping method and retain the canonical reports. Do not show these
   reports to blind reviewers.
4. Use **Blind reviewer pack** in the web application. Give each reviewer only
   their template and the generated review guide; keep the coordinator manifest
   separate.
5. Reviewers inspect the frozen repository and record repository-relative files,
   optional symbols/module/layer, and one action: `ADD`, `MODIFY`, `REMOVE`,
   `VERIFY`, `NO_ACTION`, or `UNRESOLVED`.
6. Upload both completed annotations to **Evaluation dashboard**. Reconcile all
   disagreements explicitly and confirm the snapshot.
7. Evaluate the immutable reports. Export the evaluation bundle, Markdown, and
   final ground truth together.

## New-task limitation

For `ADD_TASK`, a new stable BPMN ID normally does not exist in the baseline
source. There is no objectively known affected file until either:

- the behavior is implemented in a reference branch; or
- reviewers make and document an architecture decision from a sufficiently
  precise requirement.

The mapper may retrieve plausible files before that point, but those predictions
cannot be reused as labels. If the requirement is underspecified, use
`UNRESOLVED` rather than inventing certainty.

## Metrics interpretation

- Recall@k: how much reviewed truth appears in the first k unique files.
- Precision@k: how much of the first k is reviewed as relevant.
- MRR: how early the first relevant file appears.
- MAP: ranking quality when several files are relevant.
- Action accuracy: correct action among relevant files that were retrieved.
- Unresolved rate: proportion of change cases the mapper abstained on.
- False discovery proportion: `FP / (TP + FP)`; this is not a statistical FPR
  because the universe of true-negative repository files is not enumerated.

One scenario is a pipeline check, not accuracy evidence. Use multiple change
types, include at least one negative control, and report results as bounded to
the evaluated repository and scenarios.

## Responsibility boundary

The application can generate mutations, run and freeze predictions, create
blind templates, validate reviewer coverage, reconcile decisions, calculate
metrics, and export reproducible artifacts. Humans must define intended
behavior, produce or inspect a reference implementation, make independent
relevance judgments, and adjudicate disagreements.
