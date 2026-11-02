# Staff PR total confirmation: pre-inference source reference

Status: **source-derived reference, frozen before mapper inference for this
scenario; not implementation-verified or independently adjudicated truth**.
Prepared 2026-10-28 from the BPMN requirement and read-only Simple-ERP source.
The author has seen mapper output for other scenarios, so this is a clean
*scenario-specific* pre-inference judgment, not a fully blind reviewer study.

## Requirement and evaluation unit

After entering the purchase-request purpose, product, quantity, and estimated
unit price, Staff sees the calculated estimated total and must explicitly
confirm that total before clicking **Buat PR**. Without confirmation, no
request is sent and no DRAFT is created. This is a UI review step, not a new
server-side validation, data field, approval state, or database record. The
existing API and persistence contracts remain unchanged. The new Staff
`userTask` is `role_staff_l3_pr_create_confirm_total` after the existing input
task; the two sequence-flow edits only connect it and have no separate
implementation-file labels.

The frozen reference JSON includes production UI and existing E2E tests;
documentation and hypothetical new test files are outside this existing-file
judgment. A different product choice (for example, requiring a server-issued
confirmation token) would invalidate this reference and require a new scenario.

## Expected existing-file changes

| File relative to Simple-ERP | Action | Source-only reason |
| --- | --- | --- |
| `frontend/features/purchasing/purchase-request-panel.tsx` | MODIFY | The current create form collects purpose, product, quantity, and unit price, then immediately sends JSON on submit (lines 14–49). It has neither an estimated-total review nor a confirmation guard. Add the calculated display and prevent submission until confirmed. |
| `frontend/e2e/operations.spec.ts` | MODIFY | The test creates a PR by filling the form and clicking **Buat PR** (lines 14–20). Once confirmation is mandatory, it must perform that step to continue testing the same workflow. |
| `frontend/e2e/dashboard.spec.ts` | MODIFY | The dashboard test creates a PR as an audit fixture (lines 17–23); it also must confirm before creation. This is test fixture maintenance, not a dashboard feature change. |

The test labels assume the required confirmation is an explicit interaction
(checkbox, dialog, or equivalent). Tests may use a shared helper, but these
existing call sites cannot keep their current click-only path unchanged.

## Explicit non-impacts

- `frontend/lib/api.ts`: the request remains JSON; no multipart transport.
- `backend/internal/purchasing/adapter/http/handler.go`,
  `backend/internal/purchasing/application/service.go`, and
  `backend/internal/purchasing/adapter/postgres/store.go`: the existing create
  contract and persistence are unchanged. A UI-only confirmation cannot be
  enforced against direct API calls; that limitation is part of this scenario.
- `backend/migrations/**` and `api/openapi.yaml`: no schema or API change.
- Purchase-order, login, and PR state-transition code: unrelated to this
  pre-create review step.

## Snapshot and leakage boundary

- Simple-ERP HEAD at preparation: `c934497`, with 232 existing dirty-status
  entries. HEAD alone does **not** freeze the whole working tree.
- Old Staff L3 BPMN SHA-256:
  `93f10bf8766b4a8d1849c3803b698fc85a3d1baa05ab4a7e3453078577137d0c`.
- New BPMN SHA-256:
  `fdc4732140f41ba446f232e9c83726a8f818cdd0480c62ca9786d8349fb9a4f5`.
- Pre-inference reference JSON SHA-256:
  `b240c01cb895e3506e05fd8da5fa120f05189f0e299a235efe908f09d5c61095`.
- Source file SHA-256 values, respectively in table order:
  `b8185700375fe90e73d40da30ff8addfef4212fd638518c79f5486a4b86bcf03`,
  `268d39d02ca596950fb479fbca22d29147e5ebc9a6b80a491cda3ad03d52ee2c`,
  `7407a4bae83047665d122a6147d1a14bd62fe5d4deba1195fa2b73ce81dd4dac`.
- The JSON and this rationale live outside Simple-ERP and must not enter
  inference or tuning. Run the mapper only **after** this reference is frozen.
  Do not revise labels based on its rankings; create a versioned adjudication
  instead. For formal TA accuracy, verify against an independent implementation
  diff or reviewer and freeze the entire analyzed source snapshot.

After the first Hybrid report, the v1 BPMN documentation was found to include
explicit statements that backend/API/database changes were unnecessary. That
is an input-side implementation hint and limits the pilot's fairness. The
separate `confirm-total-v2.bpmn` removes the hint, but is not an independent
holdout because this reference and the v1 ranking are already known. Neither
file should be used to tune retrieval weights and then claim a blind result.
