# BPMN change plan

This draft is awaiting review. Resolve the open questions before implementing it.

## Process changes

1. **Modify sequence flow (Bypass completed).** The affected code still needs to be identified.
2. **Add task (Periksa apakah semua selisih stok nol).** A file change is proposed.
3. **Add task (Tutup sebagai NO_ADJUSTMENT tanpa posting stok).** A file change is proposed.
4. **Add gateway (Semua selisih nol?).** A file change is proposed.
5. **Modify sequence flow (Ya).** The affected code still needs to be identified.

## Prerequisites

- In `backend/internal/inventory/adapter/postgres/store.go`, create a new forward migration file (e.g., `000013_inventory_adjustment_status.up.sql`) to add `NO_ADJUSTMENT` to the CHECK constraint on the `status` column of the `inventory_adjustments` table, and include a corresponding rollback script.
- Create new migration file to update database constraint for `NO_ADJUSTMENT` status.
- Create a new forward migration file (e.g., `000013_inventory_adjustment_status.up.sql`) to alter the `inventory_adjustments.status` CHECK constraint to include `NO_ADJUSTMENT`, and provide a corresponding rollback script that handles existing `NO_ADJUSTMENT` rows by remapping them or blocking rollback if any exist.
- Create a new forward migration file (e.g., `000013_inventory_adjustment_status.up.sql`) to alter the `inventory_adjustments.status` CHECK constraint to include 'NO_ADJUSTMENT', and provide a corresponding rollback script that handles existing 'NO_ADJUSTMENT' rows by remapping them or blocking rollback if any exist.

## Proposed implementation

### 1. `backend/internal/inventory/adapter/postgres/store.go` (MODIFY)

**Add task (Periksa apakah semua selisih stok nol).** Add zero-delta validation logic to `TransitionAdjustment`, update database constraints to support `NO_ADJUSTMENT` status, and ensure audit logging for the new state.

Today: The `TransitionAdjustment` method in `backend/internal/inventory/adapter/postgres/store.go` validates DRAFT status and version, then queries stock movements to calculate deltas. It currently lacks logic to compare counted quantities against snapshots per line to determine if the total delta is zero; it proceeds directly to a generic submission path or conflict handling without checking for the zero-delta condition.

Required: When submitting a stock adjustment, the system compares counted quantities with snapshots per line. If all deltas are zero, the status becomes `NO_ADJUSTMENT` without posting stock or entering approval. If any line has a non-zero delta (even if total sum is zero), the standard `SUBMITTED` approval path is taken.

**Why this edit**

The BPMN requirement `CheckZeroDelta` mandates a specific validation step after version checks: comparing `counted_quantity` with `snapshot_quantity` per line. If all deltas are zero, the process must bypass approval and set status to `NO_ADJUSTMENT`. The current code lacks this conditional logic and the corresponding database constraint for `NO_ADJUSTMENT` status.

1. In `backend/internal/inventory/adapter/postgres/store.go`, add logic in `TransitionAdjustment` to iterate through stock movements and compare `quantity` against the snapshot. If all deltas are zero, set status to `NO_ADJUSTMENT` and skip the approval flow; otherwise, proceed to standard submission.
2. In `backend/internal/inventory/adapter/postgres/store.go`, create a new forward migration file (e.g., `000013_inventory_adjustment_status.up.sql`) to add `NO_ADJUSTMENT` to the CHECK constraint on the `status` column of the `inventory_adjustments` table, and include a corresponding rollback script.
3. In `backend/internal/inventory/adapter/postgres/store.go`, ensure the audit log is recorded for the `NO_ADJUSTMENT` state change within the same transaction before commit.

**Acceptance checks**

- Submit an adjustment where all line deltas are zero; expect status NO_ADJUSTMENT, no stock postings, and a successful response.
- Submit an adjustment with opposing non-zero deltas (e.g., +2 on one line, -2 on another); expect status SUBMITTED and entry into the approval queue.
- Submit an adjustment with a stale version number; expect HTTP 409 Conflict and no state change.

Evidence: `CH03-C03-E01`, `CH03-C03-E04`, `dabe0c3addf663f33e7cf8e2`, `1a71a46acabab1e9048cc705`, `f0499280a7ce2009f9274861`, `4546c965f23f09b4f4a06643`, `3f13915e4e1b6da066ab2aee`, `25db3b69b9fe157aae25487e`

**Add task (Tutup sebagai NO_ADJUSTMENT tanpa posting stok).** Implement zero-delta detection in `TransitionAdjustment`, route to `NO_ADJUSTMENT` state, and update database schema to support the new status value.

Today: The `TransitionAdjustment` method in `backend/internal/inventory/adapter/postgres/store.go` currently validates that the input status is DRAFT (lines 135-136) and then unconditionally sets the status to SUBMITTED (line 138) without checking line-level deltas or handling a zero-delta case.

Required: When submitting a stock adjustment, the system compares counted quantities with snapshots per line. If all deltas are zero, the status becomes `NO_ADJUSTMENT` without posting stock or entering approval. If any line has a non-zero delta (even if total sum is zero), the standard `SUBMITTED` approval path is taken.

**Why this edit**

The BPMN requirement introduces a new branch for zero-delta adjustments. The current code lacks the logic to calculate deltas, compare them against snapshots, and route to a NO_ADJUSTMENT state. A migration is required to add the `NO_ADJUSTMENT` status value to the database constraint.

1. In `backend/internal/inventory/adapter/postgres/store.go`, modify `TransitionAdjustment` to query `inventory_adjustment_lines` (lines 280-284) to compare `counted_quantity` against `snapshot_quantity` for each line.
2. In `backend/internal/inventory/adapter/postgres/store.go`, add logic to set status to `NO_ADJUSTMENT` and skip approval posting if all deltas are zero, otherwise proceed to set status to `SUBMITTED`.
3. Create a new forward migration file (e.g., `000013_inventory_adjustment_status.up.sql`) to alter the `inventory_adjustments.status` CHECK constraint to include `NO_ADJUSTMENT`, and provide a corresponding rollback script that handles existing `NO_ADJUSTMENT` rows by remapping them or blocking rollback if any exist.
4. In `backend/internal/inventory/adapter/postgres/store.go`, ensure audit logging is recorded for the `NO_ADJUSTMENT` state change within the same transaction before commit.

**Acceptance checks**

- Submit an adjustment where all line deltas are zero; verify status becomes NO_ADJUSTMENT, version increments, and no stock balance rows are posted.
- Submit an adjustment with at least one non-zero delta; verify status becomes SUBMITTED and the standard approval flow is triggered.

Evidence: `dabe0c3addf663f33e7cf8e2`, `1a71a46acabab1e9048cc705`, `f0499280a7ce2009f9274861`, `4546c965f23f09b4f4a06643`, `3f13915e4e1b6da066ab2aee`, `25db3b69b9fe157aae25487e`

### 2. `backend/test/integration/operations_test.go` (MODIFY)

**Add task (Periksa apakah semua selisih stok nol).** Add new test cases for zero-delta bypass and non-zero delta normal paths while preserving existing rejection assertions.

Today: The test `TestInventoryApprovalRejectsChangedSnapshot` (lines 253-272) currently validates the rejection path for changed snapshots by calling `s.Transition` with an adjustment ID and version, expecting a failure or specific error handling. It does not explicitly test the new logic where zero-delta adjustments bypass approval.

Required: The system must compare counted quantities against snapshots per line during submit. If all deltas are zero, the adjustment status becomes NO_ADJUSTMENT without posting stock or entering approval. If any line has a non-zero delta (even if total sum is zero), the standard SUBMITTED approval path is taken.

**Why this edit**

The BPMN requirement introduces a `CheckZeroDelta` service task that compares counted quantities against snapshots per line. If all deltas are zero, the flow must branch to `CloseWithoutAdjustment` (status NO_ADJUSTMENT) instead of the standard approval path. The existing test focuses on rejection scenarios; we must preserve those assertions and add distinct test cases for the new zero-delta bypass and non-zero delta normal paths.

1. Add a new test function `TestInventoryApprovalZeroDeltaBypass` in `backend/test/integration/operations_test.go` to verify the zero-delta path.
2. In `TestInventoryApprovalZeroDeltaBypass`, construct an adjustment where all line deltas are zero, call `s.Transition`, and assert the resulting status is NO_ADJUSTMENT and no stock balances were updated.
3. Add a test case within `TestInventoryApprovalZeroDeltaBypass` (or a separate one) for opposing non-zero deltas to ensure it follows the standard SUBMITTED path.
4. Ensure existing tests like `TestInventoryApprovalRejectsChangedSnapshot` remain unchanged in their assertions regarding rejection logic, but verify they do not inadvertently cover the zero-delta case.

**Acceptance checks**

- Verify that a test case with all line deltas equal to zero results in an adjustment status of NO_ADJUSTMENT and no stock posting.
- Verify that a test case with opposing non-zero deltas (e.g., +2 on one line, -2 on another) results in the standard SUBMITTED approval flow.
- Verify that a test case with a stale version or invalid precondition returns a 409 Conflict without state change.

Evidence: `48a66802c2885b1a4f2eb284`, `89a8abda82db022004a72719`, `8feadb5f156bfda0005b1714`, `0ade8b08413c4dce23616828`, `965fd5bf5bceeb592d243990`, `37fbd7f1d034a0b44681e320`

**Add task (Tutup sebagai NO_ADJUSTMENT tanpa posting stok).** Add test coverage for the zero-delta bypass path in `backend/test/integration/operations_test.go` to verify the new `CloseWithoutAdjustment` behavior.

Today: The test `TestInventoryApprovalRejectsChangedSnapshot` in `backend/test/integration/operations_test.go` exercises a path where an adjustment with changes triggers a transition to 'submit' and inserts into `stock_balances`, implying the current code or test setup does not explicitly cover the zero-delta bypass logic.

Required: The system must compare counted quantities against snapshots per line during submit. If all deltas are zero, the adjustment status becomes NO_ADJUSTMENT without posting stock or entering approval. If any line has a non-zero delta (even if total sum is zero), the standard SUBMITTED approval path is taken.

**Why this edit**

The BPMN requires a new service task `CloseWithoutAdjustment` that executes when all line deltas are zero. The existing test fixture only covers the non-zero delta path (lines 262-266). A new test case is required to verify the zero-delta logic, ensuring it sets status to NO_ADJUSTMENT and skips stock posting.

1. Add a new test function `TestInventoryApprovalZeroDeltaBypass` in `backend/test/integration/operations_test.go` that constructs an adjustment with zero deltas on all lines.
2. In the new test, call `s.Transition` and assert the resulting status is NO_ADJUSTMENT and no rows are inserted into `stock_balances`.
3. Add a subtest or separate case for opposing non-zero deltas to verify they follow the standard SUBMITTED path.

**Acceptance checks**

- When an adjustment has zero delta on all lines, the system returns a success response and the adjustment status becomes NO_ADJUSTMENT without updating stock balances.
- When an adjustment has non-zero delta on any line (even if total sum is zero), the system follows the standard SUBMITTED approval path.

Evidence: `48a66802c2885b1a4f2eb284`, `89a8abda82db022004a72719`, `8feadb5f156bfda0005b1714`, `0ade8b08413c4dce23616828`, `965fd5bf5bceeb592d243990`, `37fbd7f1d034a0b44681e320`

## Open review items

- `backend/internal/inventory/adapter/postgres/store.go`. Add gateway (Semua selisih nol?). The edit affects the `TransitionAdjustment` method which is the correct owner for the submit logic, but the proposal assumes the schema allows 'NO_ADJUSTMENT' without verifying the current constraint.
- `backend/internal/inventory/adapter/postgres/store.go` (contracts). The proposed change to set status 'NO_ADJUSTMENT' requires a database schema update, but the draft claims a migration is needed without citing the existing CHECK constraint or table definition in the source.
- `backend/internal/inventory/adapter/postgres/store.go` (scope). The edit affects the `TransitionAdjustment` method which is the correct owner for the submit logic, but the proposal assumes the schema allows 'NO_ADJUSTMENT' without verifying the current constraint.
- `backend/internal/inventory/adapter/postgres/store.go` (failure_tests). The draft proposes adding a new status 'NO_ADJUSTMENT' but does not provide evidence that the current database schema or CHECK constraints permit this value.
- Modify sequence flow (Bypass completed) has no evidence-backed file plan.
- Modify sequence flow (Ya) has no evidence-backed file plan.
- Schema source invalid: backend/internal/inventory/adapter/postgres/store.go queries non-zero-only ledger table "stock_movements" (CHECK <> 0) to evaluate per-line zero deltas before posting; query each line's quantities from "inventory_adjustment_lines" (snapshot_quantity vs counted_quantity) instead.

Plan `6e1d2fc30eb3570b3ae08a57e8e74fc4` from analysis `a01dd63bc020b8b8dd3b4b6d1344df6a`. The JSON export contains all assessments, source references and reviewer checks.
