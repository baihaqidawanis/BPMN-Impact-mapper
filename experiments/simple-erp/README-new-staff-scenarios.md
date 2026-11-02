# Staff L3 case-study mutations

These are **case-study experiment inputs**, not mapper unit-test fixtures or
implemented Simple-ERP changes. Use the matching baseline in read-only
`C:\Simple-ERP`. The new BPMN files include the baseline's BPMN-DI layout, so
they should open visibly in bpmn.io. Both use existing stable element IDs.

| Scenario | Old BPMN in Simple-ERP | New BPMN here | Expected diff |
| --- | --- | --- | --- |
| PR positive estimated unit price | `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-create.bpmn` | `staff-p2p/role-staff-l3-pr-create-positive-price.bpmn` | `MODIFY_TASK_LABEL` on `role_staff_l3_pr_create_check` |
| Stock adjustment submit only with a nonzero delta | `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn` | `staff-inventory/role-staff-l3-adj-submit-nonzero-delta.bpmn` | `MODIFY_GATEWAY_CONDITION` on `role_staff_l3_adj_submit_valid` and supporting `MODIFY_SEQUENCE_FLOW` on `role_staff_l3_adj_submit_Flow_3` |
| Stock adjustment submit with a required reason (structural revision) | `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn` | `staff-inventory/role-staff-l3-adj-submit-reason-task.bpmn` | `ADD_TASK` on `role_staff_l3_adj_submit_reason` and two supporting flow changes |

The second row changes a gateway condition but **does not add a task**. Its
existing report remains valid for that specific input. The third row is a
separate, versioned structural scenario: it adds a visible Staff user task
before the existing submit task. The backend must require a non-empty reason
on submit, and the frontend must collect/send it. It is **not** a replacement
for the previously analyzed BPMN/report; run a new analysis for it.

Run **one pair at a time** in Mapping. Repository path is the local Simple-ERP
directory, repository name `Simple-ERP`; revision may be left blank, but record
the source snapshot separately. Use the production Hybrid method and top 10,
then export JSON. Do not place the reference JSON files in the repository being
indexed or use them to tune ranking before the report. The references are in
`docs/ground-truth/`, prepared from source inspection before running either
scenario against Simple-ERP; they are *not* implementation-verified ground truth.

The copied baseline BPMN contains source-path documentation already present in
Simple-ERP. This may bias retrieval and limits any formal accuracy claim. For
the TA evaluation, create a separately frozen, sanitized BPMN/source snapshot
and independently adjudicate the expected changes.
