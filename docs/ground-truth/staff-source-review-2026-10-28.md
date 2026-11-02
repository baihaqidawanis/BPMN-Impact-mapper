# Single-review Simple-ERP source reference

Prepared 2026-10-28 by the mapper development assistant from BPMN XML and
Simple-ERP source **before opening any mapper ranking for these two inputs in
this review**. The assistant is not an independent second reviewer, and both
mutations existed before this review. These files are provisional diagnostic
labels, not adjudicated TA ground truth. No implementation diff exists for the
hypothetical change. Another reviewer could reasonably include a UI affordance
or other tests; the reference below defines the minimum *necessary existing
files* under the stated interpretation.

Repository: read-only `C:\Simple-ERP`, HEAD
`c9344971a573d5c44d9d4e58f23657eda4d18a86`, with 232 existing dirty
status entries at review time. HEAD alone does not reproduce this source tree.
The benchmark's method-independent input fingerprint pins the allowed current
source corpus and both BPMNs; the manifest separately pins the two BPMN files
and each reference JSON. Do not copy this reference into the indexed repo.

## Positive: stock adjustment with a nonzero difference

Old Staff L3 BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn`,
SHA-256 `d5c9b4c5d71f1b552a01e915770ea434cb7b3673cfccc6aae0c9be19b430803d`.
New BPMN: `experiments/simple-erp/staff-inventory/role-staff-l3-adj-submit-nonzero-delta.bpmn`,
SHA-256 `e40124746ac36683e784184ff1517455f514fb29b27e850903ee2c9adc993f9b`.

The success branch now requires at least one counted quantity different from
its snapshot. `backend/internal/inventory/adapter/postgres/store.go` has the
`TransitionAdjustment` submit branch around line 135: it checks DRAFT but does
not check counted versus snapshot before setting SUBMITTED. It therefore needs
the rule inside the database transaction. `backend/test/integration/operations_test.go`
already exercises create, submit, approve, and snapshot conflicts around lines
254-310; it needs a no-delta submission rejection and a positive-delta pass.
Both are assigned to `MODIFY_GATEWAY_CONDITION` on
`role_staff_l3_adj_submit_valid`. The changed success flow is the same rule's
representation and has no *independent* file impact.

`frontend/features/inventory/inventory-adjustment-panel.tsx` can submit and
display backend errors. It lacks snapshot quantity, so frontend prevalidation
is optional UX work rather than a necessary correctness change. The inventory
HTTP handler already returns the store error as 409, and the service delegates
to the store; no separate route, service, schema, or migration edit is proven
by this BPMN difference. An exact error message or cross-layer UI behavior is
not specified by the new BPMN and remains unresolved.

## Negative: PR failure-branch display label

Old Staff L3 BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-create.bpmn`,
SHA-256 `93f10bf8766b4a8d1849c3803b698fc85a3d1baa05ab4a7e3453078577137d0c`.
New BPMN: `experiments/simple-erp/staff-p2p/role-staff-l3-pr-create-sequence-label.bpmn`,
SHA-256 `b7f84aa786367a38e769bbcc5322157ab054e5822c715196248aa1d73e0607eb`.

Only the `name` of `role_staff_l3_pr_create_Flow_6` changes from `Tidak` to
`Data PR tidak lengkap`. Its source, target, and `conditionExpression` remain
unchanged. No source-code behavior or displayed UI copy is specified to change;
the expected existing-file impact set is empty. This is an abstention control.

The BPMN source includes process-level documentation naming source files. If
the mapper uses that text as query evidence, it may bias the result. The
case-study references and experiment outputs must remain outside retrieval.
