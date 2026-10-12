# Simple-ERP pilot protocol

This directory contains **case-study experiment inputs**, not product rules and not ground truth. The mapper contains no Simple-ERP-specific behavior.

## Artifact separation

- Method input: one old BPMN, one generated new BPMN, the read-only repository, selected method, and retrieval limits.
- Frozen baseline: the SHA-256 in each mutation specification and `mapping-config.json`, prepared before reviewer adjudication.
- Excluded corpus: `docs/PROCESS-MAPPING.md`, `docs/TRACEABILITY-COVERAGE.md`, `docs/checkpoints/**`, `docs/RELEASE-CANDIDATE.md`, audit reports, plans, generated reports, ground truth, and implementation diffs. Normal ingestion excludes these categories; they must not inform prediction.
- Reviewer data: must be authored independently after inference. Predictions are not reviewer labels.
- Evaluation artifacts: generated BPMN and reports are written below ignored `outputs/pilots/` and must not be committed.

The six specifications exercise label, gateway-condition, sequence-flow-label, actor-role, terminal-outcome, and removal/rewiring changes. `coverage/` adds focused `ADD_TASK` and `ADD_GATEWAY` specifications. They are controlled research scenarios, not claims about requested Simple-ERP features. Final metrics require two independent reviewers plus explicit adjudication of disagreements.

Example generation:

```text
bpmn-impact-mapper mutate --spec experiments/simple-erp/specs/p2p-task-label.json --baseline C:/Simple-ERP/docs/bpmn/l2-procure-to-pay.bpmn --output outputs/pilots/generated/p2p-task-label.bpmn
```

The command verifies the baseline hash, applies every operation exactly once, validates the resulting BPMN through the production parser, and refuses to overwrite an existing file.
