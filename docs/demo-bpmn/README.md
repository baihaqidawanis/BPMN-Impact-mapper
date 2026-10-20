# Manual BPMN demo fixtures

This directory contains a **read-only demonstration fixture** for BPMN Impact
Mapper. It does not modify Simple-ERP and is not a product rule or ground
truth.

## Add a user task to Procure-to-Pay

- **Baseline BPMN:** `C:\Simple-ERP\docs\bpmn\l2-procure-to-pay.bpmn`
- **New BPMN:** `docs\demo-bpmn\simple-erp-p2p-add-user-task.bpmn`
- **Repository path:** `C:\Simple-ERP`
- **Recommended method:** BPMN-Context-Aware Hybrid Retrieval, Top-K 5.

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

## Canvas-safe synthetic add/remove demo

For a small diagram that visibly opens in BPMN.io or the Mapper canvas, use
`web/public/demo/insert-task-old.bpmn` and
`web/public/demo/insert-task-new.bpmn`. Old to new adds the `B` user task;
new to old removes it. Both files include BPMN-DI shapes, edge waypoints, and
stable process/flow IDs. They are synthetic process examples, **not** the
Simple-ERP process or verified code-impact ground truth. Use the existing
synthetic demo repository for source mapping; use this pair to inspect the
diagram and graph diff.

The Simple-ERP-specific add-task example above also includes BPMN-DI. Do not
upload the DI-free `testdata/bpmn/*` parser fixtures to a visual editor: those
fixtures test semantic parsing and are not presentation diagrams.

## Add a gateway and task to the real Simple-ERP P2P baseline

For a source-backed case study, **do not** use the synthetic `web/public/demo`
pair as the old BPMN. Use these manual inputs in the Mapping tab:

- Repository path: `C:\Simple-ERP` (read-only; the ERP does not need to run).
- Baseline BPMN: `C:\Simple-ERP\docs\bpmn\l2-procure-to-pay.bpmn`.
- New BPMN: `docs\demo-bpmn\simple-erp-p2p-add-gateway-task.bpmn`.
- Repository name: `Simple-ERP`.
- Revision: identify the actual indexed checkout, e.g. `c928a29+dirty` for the
  local checkout inspected on 2026-10-20; do not claim `69724e6` if the source
  working tree is different. Revision is metadata, not an automatic Git checkout.

The new file is derived from the unchanged baseline BPMN (its working-tree blob
matches commit `69724e6` as of that inspection). After Submit purchase request,
an exclusive gateway asks whether details are complete. A complete request
continues to Review purchase request; otherwise the new Admin user task
Clarify purchase request details leads to the same review. The expected diff is
one `ADD_GATEWAY`, one `ADD_TASK`, and four `MODIFY_SEQUENCE_FLOW` records.
The file includes BPMN-DI coordinates for the gateway, task, and all changed
flows. The local canvas imported it without warnings; bpmn.io can open the
same file for visual inspection. This is a *hypothetical change*, not an
implemented Simple-ERP feature or validated impact ground truth. Source-code
candidates remain `VERIFY` unless stronger evidence supports an action.

## Canvas-safe synthetic gateway branch demo

Use `web/public/demo/add-branch-old.bpmn` as baseline and
`web/public/demo/add-branch-new.bpmn` as the changed diagram. It introduces an
exclusive routing gateway and an Escalate user task. Reversing the pair removes
that branch. Both contain BPMN-DI shapes and edge waypoints. Browser tests
confirm every node and sequence-flow path renders in the Mapper canvas; the
synthetic source repository is still only a demo, not accuracy ground truth.
