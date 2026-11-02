# PR positive-price source reference

Status: **pre-inference, source-derived reference; not independently validated
ground truth**. Prepared 2026-10-28 without inspecting mapper output for this
scenario. The scenario changes the existing validation task: estimated unit
price must be greater than zero (rather than allowing zero). No new task, API
field, or runtime BPMN dependency is implied.

Existing-file targets in the companion JSON:

- `frontend/features/purchasing/purchase-request-panel.tsx` — current `create`
  guard allows zero (`unitPrice < 0`); the form must reject zero.
- `backend/internal/purchasing/adapter/postgres/store.go` — `CreateRequest`
  allows zero (`l.UnitPrice < 0`); direct API calls also need rejection.

`backend/migrations/000005_operations.up.sql` currently allows zero via
`estimated_price >= 0`. A robust implementation may need a **new migration**
to enforce the changed constraint; do not rewrite the applied migration. That
new file does not exist in the source snapshot, so the companion JSON's
existing-file evaluation cannot label it. Existing zero-valued records and
`CopyRequest` require a product/data-migration decision. Do not present the
two-file reference as a complete implementation plan.

Snapshot: Simple-ERP HEAD `c934497` with 232 pre-existing dirty-status entries;
HEAD alone does not fix the entire tree. Baseline BPMN SHA-256
`93f10bf8766b4a8d1849c3803b698fc85a3d1baa05ab4a7e3453078577137d0c`;
new BPMN `539af56fbfdface3dcd75b3123a6ddba0fc01e3cfd5e8bd539747054b6ad2fa2`;
reference JSON `fc030cf9057bb17d34162d088a7b7e08838b23c69c5b698225d2a59651935f37`.
Source SHA-256: frontend panel
`b8185700375fe90e73d40da30ff8addfef4212fd638518c79f5486a4b86bcf03`;
backend store `bc806637c6b847d3aabcf1c80c6d2352c89545ebf2f6b27e33f4bb9331b47688`;
existing migration `cf6ffb45600dcee577898994e370ed8a1a3d3a6d8921007a0d2fb418832f5e32`.

Do not change the reference after viewing rankings. Record any correction as
a separately versioned adjudication, ideally after an independent code diff.
