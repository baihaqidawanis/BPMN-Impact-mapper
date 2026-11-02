# Inventory submit-reason task: pre-inference source reference

Status: **source-derived reference, not independently adjudicated ground
truth**. Prepared 2026-10-28 before running this structural mutation against
Simple-ERP. The new `role_staff_l3_adj_submit_reason` user task requires Staff
to provide a non-empty adjustment reason before submit. It does not add a new
API field or database column: the existing action request already carries
`reason`, but the current UI sends an empty string for `submit` and the backend
accepts it.

The companion JSON labels three existing files for the primary `ADD_TASK`:

- `frontend/features/inventory/inventory-adjustment-panel.tsx`: collect a
  submit reason and send it instead of an empty string.
- `backend/internal/inventory/adapter/postgres/store.go`: enforce non-empty
  reason in the `submit` branch, including direct API calls.
- `backend/test/integration/operations_test.go`: existing successful
  adjustment submits pass an empty reason and would need updating; add a
  rejection check for an empty reason.

The two flow changes only connect the new task and have empty separate
file labels to avoid double-counting the same implementation work. This is a
product requirement proposal, not proof these files would be the only changes
in a completed implementation.

Snapshot: Simple-ERP HEAD `c934497` with 232 pre-existing dirty-status entries;
HEAD alone is not a frozen full tree. Baseline BPMN SHA-256
`d5c9b4c5d71f1b552a01e915770ea434cb7b3673cfccc6aae0c9be19b430803d`;
new BPMN `078e362843fde3374c5673db69a6b6b7edbe58c6a4ba43f86fd3a4dfd2cdcf3e`;
reference JSON `e58255789940f015cce9c1ab780852d70181ba51a2f06d74e39544892f5955ce`.
Source SHA-256 values: frontend adjustment panel
`f7f3416abf3c9b7bda526dc2e339cf81a5d8d9185a19955596ea4867fcc49f61`;
backend inventory store
`f8d015e104a9a50d3f38402a10d1337e9aeddf58cf8cade4e8e6afe865c03184`;
integration operations tests
`7df3d774cfaab238a369a332245d60dfcbbbb8f1ae12a4b26708575cfe825101`.
Do not change these labels after viewing the new mapper ranking. Version any
adjudication separately and keep all reference files outside the inference
corpus.
