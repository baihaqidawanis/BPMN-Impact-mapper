# Inventory nonzero-delta source reference

Status: **pre-inference, source-derived reference; not independently validated
ground truth**. Prepared 2026-10-28 without inspecting mapper output for this
scenario. When Staff submits a DRAFT stock adjustment, at least one line's
counted quantity must differ from its saved snapshot quantity. A no-op
adjustment is rejected. This is a new submit guard, not a new field or step.

The existing-file target is
`backend/internal/inventory/adapter/postgres/store.go`, specifically the
`submit` branch of `TransitionAdjustment`. It currently checks only `DRAFT`;
the stored adjustment lines already contain `snapshot_quantity` and
`counted_quantity`. An implementation should compare them in the transaction
before changing status. The existing form submits an ID/version and has no
snapshot data, so this rule cannot be reliably enforced by its current UI;
frontend change is not required for correctness. Add a backend regression
test as implementation work, but no existing test file is designated a required
retrieval target in this reference.

The gateway-condition change is the behavioral unit. The outgoing-flow
condition also changes to express it and is labeled with an empty expected
file set to avoid double-counting the same implementation change. This is a
scoring convention, not a claim that the flow has no semantic effect.

Snapshot: Simple-ERP HEAD `c934497` with 232 pre-existing dirty-status entries.
Baseline BPMN SHA-256
`d5c9b4c5d71f1b552a01e915770ea434cb7b3673cfccc6aae0c9be19b430803d`;
new BPMN `e40124746ac36683e784184ff1517455f514fb29b27e850903ee2c9adc993f9b`;
reference JSON `26d5f866f9c5a24effd9f482bbe8c1aae4731c1d0325d935178ed9df86eb4d1c`.
Backend store SHA-256
`f8d015e104a9a50d3f38402a10d1337e9aeddf58cf8cade4e8e6afe865c03184`.

Do not change the reference after viewing rankings. Record any correction as
a separately versioned adjudication, ideally after an independent code diff.
