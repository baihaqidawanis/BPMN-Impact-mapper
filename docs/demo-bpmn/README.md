# Manual BPMN demo fixtures

This directory contains a **read-only demonstration fixture** for BPMN Impact
Mapper. It does not modify Simple-ERP and is not a product rule or ground
truth.

## Add a user task to Procure-to-Pay

- **Baseline BPMN:** `C:\Simple-ERP\docs\bpmn\l2-procure-to-pay.bpmn`
- **New BPMN:** `docs\demo-bpmn\simple-erp-p2p-add-user-task.bpmn`
- **Repository path:** `C:\Simple-ERP`
- **Recommended method:** Graph-Aware Hybrid, Top-K 5.

The fixture inserts exactly one `bpmn:userTask`:

```text
PUR-PR-002  Submit purchase request
  -> PUR-PR-002A  Validate purchase request details
  -> PUR-PR-003  Review purchase request
```

It changes `FLOW-P2P-003`, adds `FLOW-P2P-003A`, and adds the new task to
the Operator lane. There is no duplicate `PUR-PR-002A` and no `serviceTask`
with that ID.

Expected mapper changes are one `ADD_TASK` plus related
`MODIFY_SEQUENCE_FLOW` records. Because the new task has no implementation in
the unchanged repository, `VERIFY` candidates are expected; a zero direct
trace-ID signal for the new task is normal.
