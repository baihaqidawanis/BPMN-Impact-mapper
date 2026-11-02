# Staff adjustment cancellation reason: source-only reference

Prepared before mapper inference for this case. Status: provisional,
single-assistant source review, not implemented or independently adjudicated.
Other Staff mapping results are already known; this is a fresh operation-level
development case, not an independent blind study or a new domain.

Requirement: cancelling a DRAFT or APPROVED stock adjustment requires at least
ten Unicode code points after trimming leading/trailing whitespace. A shorter
or whitespace-only reason must fail before changing status/version or creating
a cancellation audit. A ten-character reason passes this new rule if existing
authorization, ownership, state and version checks pass. This validation is
specific to cancel; submit, reject and reverse retain their current rules.
No new fields, statuses, tables, routes or workflow runtime are introduced.

The BPMN changes the existing Staff input-task label to state this requirement;
it does not insert a task or gateway. The existing service label's requirement
that a reason exists is still true, but less detailed than the new input rule.

Existing-file targets:

- `frontend/features/inventory/inventory-adjustment-panel.tsx`, component
  `InventoryAdjustmentPanel`, function `action` (around line 16): currently
  sends the reason for cancel without length validation. Enforce this rule on
  cancel and explain rejection to the user; preserve the shared field's other
  actions. Counting must agree with the backend for Unicode input.
- `backend/internal/inventory/adapter/postgres/store.go`, `TransitionAdjustment`
  cancel branch (around lines 172-180): checks `reason == ""` only, so direct
  API calls can bypass a UI-only rule. Enforce the trimmed-length requirement
  on cancel without weakening existing guards.
- `backend/test/integration/operations_test.go`: existing
  `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` exercises
  cancellation and subsequent recovery. Extend coverage for 9/10-character
  boundaries, whitespace, Unicode and unchanged state/version on rejection.
  Existing valid long-reason recovery should still pass. The existing file is
  an expected regression-test extension, not proof that all its current lines
  must change. A separate new test file is a possible implementation choice
  outside this existing-file reference policy.

Excluded targets: the HTTP handler already decodes `Reason`, the application
service already passes it through, and the existing schema/audit can store it.
No transport or schema contract change is required by this requirement.
Documentation and hypothetical new files are outside this existing-file
evaluation. The three targets are not guaranteed to be an exhaustive future
implementation diff. Never adjust this reference from candidate rankings;
record any later adjudication as a new version.

The paired baseline and mutation live in
`experiments/simple-erp/staff-inventory/cancel-reason-min10/`. Input/source
hashes and preparation status are recorded there. Ground truth must not enter
the Simple-ERP retrieval corpus. No mapper ranking for this case was inspected
during preparation.
