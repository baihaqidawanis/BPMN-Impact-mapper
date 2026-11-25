# Skip ground truth

The reference covers existing files for `ADD_GATEWAY / ZeroDeltaDecision` and its two business outcomes. Evaluate unique files proposed for an edit against `reference.json`. Review new artifacts and plan correctness separately.

The current Before already joins the successful submit and the HTTP 409 response path before sending the result. After reuses that XOR join, the conflict preparation task, and their original IDs. The new Skip path starts only after DRAFT and version validation succeeds. The join and the conflict task do not introduce a separate requirement or an edit to the HTTP handler.

Use the paired BPMN files in this folder for evaluation. Their descriptions state business behavior. File paths and function names that identify the implementation live in these reference notes.

| File | Source-backed reason |
|---|---|
| `backend/internal/inventory/adapter/postgres/store.go` | At lines 120–187, `Store.TransitionAdjustment` locks the adjustment, validates its version and DRAFT state, and writes SUBMITTED for the submit action. This is where the request must branch on every line's counted quantity versus its snapshot. `PostAdjustment` at lines 196–265 already skips individual zero movements during a later operation. |
| `backend/test/integration/operations_test.go` | The integration tests at lines 253 and 274 exercise the inventory transition through the actual runtime store. Add cases for an all-zero adjustment, opposing nonzero deltas, and stale submission. Preserve existing nonzero approval and snapshot tests. |

The baseline status constraint in `backend/migrations/000005_operations.up.sql:36` excludes `NO_ADJUSTMENT`. A forward migration must allow the new state, with an explicit rollback strategy. The original migration is evidence for the change. Score the proposed migration by description because the new file does not yet exist and has no fixed path.

The application service delegates to `Store.TransitionAdjustment` at `backend/internal/inventory/application/service.go:51`. The domain method `Adjustment.Submit` is only exercised by tests in the current inventory code. Approval and pending-count logic already filter by SUBMITTED. Presentation work can be assessed as a separate proposal if it has a concrete contract change.

Acceptance checks should show that all-zero lines lead to terminal NO_ADJUSTMENT with a version advance and a committed audit entry on valid submit, with no approval queue entry or ledger or balance mutation. One nonzero line follows the existing SUBMITTED path, including when two lines have opposing nonzero deltas that sum to zero. A stale version or non-DRAFT status keeps the existing HTTP 409 response and leaves state and ledger unchanged. Add dedicated Skip cases while preserving the existing approval and changed-snapshot regression tests.
