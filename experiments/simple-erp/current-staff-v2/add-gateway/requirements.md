# Inventory submit with XOR split and join

Baseline: [Inventory submit before BPMN](../add-task/before.bpmn).
Variant: [Inventory submit after BPMN](after.bpmn).

The existing split permits submission when the adjustment is DRAFT, the version matches, and the submit reason is present. A failing branch preserves state and version and prepares the rejection response. The join merges the success and rejection branches before the existing response task.

## Provisional file reference

The reference measures the changed business condition on the existing split. The join, response preparation, diagram labels, and rewired sequence flows are supporting diagram changes. Review them for coverage without counting them as independent file-level benchmark cases.

| File in Simple-ERP | Source evidence and expected change |
| --- | --- |
| `frontend/features/inventory/inventory-adjustment-panel.tsx` | The action builds a payload with an empty reason on submit. Send the entered reason when submitting and surface invalid input. |
| `backend/internal/inventory/adapter/postgres/store.go` | The submit branch checks DRAFT and updates state. Require a reason before the mutation, preserving version checks and the behavior of other actions. The draft insert omits reason, so a global constraint would affect draft creation. |
| `backend/test/integration/operations_test.go` | Existing inventory submit tests pass an empty reason. Update success fixtures and test that a missing reason leaves status and version unchanged. |

This is a source-derived reference for one BPMN change. No implementation changes were made to Simple-ERP.
