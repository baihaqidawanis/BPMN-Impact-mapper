# Freeze penelitian sebelum LLM

Freeze ID: c584ecd0dcbf523272863d69249d2fd3

Sepuluh kasus lama dicatat sebagai development/regression. Belum ada kasus uji independen. Label tidak diubah mengikuti ranking mapper.

Hash mengikat BPMN, label, source review, source ERP dan kode kebijakan saat ini. Ini pemeriksaan drift, bukan backup repository atau bukti runtime.

| Kasus | Keluarga kebutuhan | Scope | File acuan |
|---|---|---|---|
| normalized-staff-add-task | inventory-submit-validation | subset | 3 |
| normalized-staff-add-xor | inventory-submit-validation | subset | 3 |
| normalized-staff-add-and | inventory-submit-validation | subset | 2 |
| normalized-staff-combined | inventory-submit-validation | subset | 3 |
| normalized-staff-unchanged-control | unchanged-control | complete | 0 |
| p2p-transfer-checks | pr-submit-master-price-validation | subset | 2 |
| p2p-transfer-unchanged-control | unchanged-control | complete | 0 |
| selection-challenge-remove-check | pr-copy-eligibility | subset | 2 |
| selection-challenge-change-condition | pr-copy-eligibility | subset | 3 |
| selection-challenge-unchanged-control | unchanged-control | complete | 0 |

## normalized-staff-add-task

Require nonempty adjustment submit reason. Reject failures without changing state; preserve version/access guards.

Before: experiments/simple-erp/current-staff/add-task/before.bpmn

After: experiments/simple-erp/current-staff/add-task/after.bpmn

Label: experiments/simple-erp/current-staff/add-task/reference.json

Acuan perubahan sementara:

- backend/internal/inventory/adapter/postgres/store.go: Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.
  - Store.TransitionAdjustment L134: case "submit":.
  - Store.TransitionAdjustment L135: from != "DRAFT".
  - Store.TransitionAdjustment L138: d.Status = "SUBMITTED".
- backend/test/integration/operations_test.go: Extend existing integration test for new failures, success and preserved invariants.
  - TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot L283: .Transition(.
- frontend/features/inventory/inventory-adjustment-panel.tsx: Existing reason input sends empty reason on submit; require and send it for submit.
  - InventoryAdjustmentPanel L16: const actionReason =.
  - InventoryAdjustmentPanel L16: reason: actionReason.

Kontrak yang tetap dapat digunakan:

- Existing endpoint, request/response and schema suffice; no migration.
- AND requires both checks to finish before mutation; no goroutine requirement.

## normalized-staff-add-xor

Require nonempty adjustment submit reason. Reject failures without changing state; preserve version/access guards.

Before: experiments/simple-erp/current-staff/add-xor/before.bpmn

After: experiments/simple-erp/current-staff/add-xor/after.bpmn

Label: experiments/simple-erp/current-staff/add-xor/reference.json

Acuan perubahan sementara:

- backend/internal/inventory/adapter/postgres/store.go: Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.
  - Store.TransitionAdjustment L134: case "submit":.
  - Store.TransitionAdjustment L135: from != "DRAFT".
  - Store.TransitionAdjustment L138: d.Status = "SUBMITTED".
- backend/test/integration/operations_test.go: Extend existing integration test for new failures, success and preserved invariants.
  - TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot L283: .Transition(.
- frontend/features/inventory/inventory-adjustment-panel.tsx: Existing reason input sends empty reason on submit; require and send it for submit.
  - InventoryAdjustmentPanel L16: const actionReason =.
  - InventoryAdjustmentPanel L16: reason: actionReason.

Kontrak yang tetap dapat digunakan:

- Existing endpoint, request/response and schema suffice; no migration.
- AND requires both checks to finish before mutation; no goroutine requirement.

## normalized-staff-add-and

At submit recheck active warehouse AND all products. Reject failures without changing state; preserve version/access guards.

Before: experiments/simple-erp/current-staff/add-and/before.bpmn

After: experiments/simple-erp/current-staff/add-and/after.bpmn

Label: experiments/simple-erp/current-staff/add-and/reference.json

Acuan perubahan sementara:

- backend/internal/inventory/adapter/postgres/store.go: Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.
  - Store.TransitionAdjustment L134: case "submit":.
  - Store.TransitionAdjustment L135: from != "DRAFT".
  - Store.TransitionAdjustment L138: d.Status = "SUBMITTED".
  - Store.CreateAdjustment L86: FROM warehouses.
  - Store.CreateAdjustment L103: FROM products.
- backend/test/integration/operations_test.go: Extend existing integration test for new failures, success and preserved invariants.
  - TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot L283: .Transition(.

Kontrak yang tetap dapat digunakan:

- Existing endpoint, request/response and schema suffice; no migration.
- AND requires both checks to finish before mutation; no goroutine requirement.
- No new input; existing frontend/handler reuse version and error response.

## normalized-staff-combined

Require nonempty adjustment submit reason. At submit recheck active warehouse AND all products. Reject failures without changing state; preserve version/access guards.

Before: experiments/simple-erp/current-staff/combined/before.bpmn

After: experiments/simple-erp/current-staff/combined/after.bpmn

Label: experiments/simple-erp/current-staff/combined/reference.json

Acuan perubahan sementara:

- backend/internal/inventory/adapter/postgres/store.go: Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.
  - Store.TransitionAdjustment L134: case "submit":.
  - Store.TransitionAdjustment L135: from != "DRAFT".
  - Store.TransitionAdjustment L138: d.Status = "SUBMITTED".
  - Store.CreateAdjustment L86: FROM warehouses.
  - Store.CreateAdjustment L103: FROM products.
- backend/test/integration/operations_test.go: Extend existing integration test for new failures, success and preserved invariants.
  - TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot L283: .Transition(.
- frontend/features/inventory/inventory-adjustment-panel.tsx: Existing reason input sends empty reason on submit; require and send it for submit.
  - InventoryAdjustmentPanel L16: const actionReason =.
  - InventoryAdjustmentPanel L16: reason: actionReason.

Kontrak yang tetap dapat digunakan:

- Existing endpoint, request/response and schema suffice; no migration.
- AND requires both checks to finish before mutation; no goroutine requirement.

## normalized-staff-unchanged-control

Identical BPMN control.

Before: experiments/simple-erp/current-staff/unchanged-control/before.bpmn

After: experiments/simple-erp/current-staff/unchanged-control/after.bpmn

Label: experiments/simple-erp/current-staff/unchanged-control/reference.json

Acuan perubahan sementara:

Tidak ada kebutuhan berubah; kontrol identik.

Kontrak yang tetap dapat digunakan:

- No requirement change.

## p2p-transfer-checks

At PR submit recheck all products active AND estimated prices strictly positive. Reject failures without changing state; preserve version/access guards.

Before: experiments/simple-erp/p2p-transfer/checks/before.bpmn

After: experiments/simple-erp/p2p-transfer/checks/after.bpmn

Label: experiments/simple-erp/p2p-transfer/checks/reference.json

Acuan perubahan sementara:

- backend/internal/purchasing/adapter/postgres/store.go: Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.
  - Store.TransitionRequest L147: case "submit":.
  - Store.TransitionRequest L148: from != "DRAFT".
  - Store.TransitionRequest L151: d.Status = "SUBMITTED".
  - Store.CreateRequest L53: FROM products.
  - Store.CreateRequest L49: l.UnitPrice < 0.
- backend/test/integration/operations_test.go: Extend existing integration test for new failures, success and preserved invariants.
  - TestP2PPartialReceiptPostsImmutableSourceLinkedLedger L93: .TransitionRequest(.

Kontrak yang tetap dapat digunakan:

- Existing endpoint, request/response and schema suffice; no migration.
- AND requires both checks to finish before mutation; no goroutine requirement.
- No new input; existing frontend/handler reuse version and error response.

## p2p-transfer-unchanged-control

Identical BPMN control.

Before: experiments/simple-erp/p2p-transfer/unchanged-control/before.bpmn

After: experiments/simple-erp/p2p-transfer/unchanged-control/after.bpmn

Label: experiments/simple-erp/p2p-transfer/unchanged-control/reference.json

Acuan perubahan sementara:

Tidak ada kebutuhan berubah; kontrol identik.

Kontrak yang tetap dapat digunakan:

- No requirement change.

## selection-challenge-remove-check

Allow copying REJECTED PR with inactive products; retain status guard and source state.

Before: experiments/simple-erp/selection-challenges/remove-check/before.bpmn

After: experiments/simple-erp/selection-challenges/remove-check/after.bpmn

Label: experiments/simple-erp/selection-challenges/remove-check/reference.json

Acuan perubahan sementara:

- backend/internal/purchasing/adapter/postgres/store.go: Remove active-product guard only; shared function remains.
  - Store.CopyRequest L488: status != "REJECTED".
  - Store.CopyRequest L488: !mastersActive.
  - Store.CopyRequest L488: "REJECTED".
- backend/test/integration/operations_test.go: Extend Copy PR tests; preserve unchanged sales assertions and retained guards.
  - TestCopyRejectedDocumentsRequiresActiveMasters L359: purchasing.CopyRequest(.

Kontrak yang tetap dapat digunakan:

- No new route, request field, database column or role permission.
- Existing application delegation/domain Copy do not implement the changed guard.
- Copy remains visible for REJECTED; no frontend change required.

## selection-challenge-change-condition

Allow REJECTED or CANCELLED PR copy; preserve active products and source state.

Before: experiments/simple-erp/selection-challenges/change-condition/before.bpmn

After: experiments/simple-erp/selection-challenges/change-condition/after.bpmn

Label: experiments/simple-erp/selection-challenges/change-condition/reference.json

Acuan perubahan sementara:

- backend/internal/purchasing/adapter/postgres/store.go: Extend status eligibility and audit origin.
  - Store.CopyRequest L488: status != "REJECTED".
  - Store.CopyRequest L488: !mastersActive.
  - Store.CopyRequest L488: "REJECTED".
- backend/test/integration/operations_test.go: Extend Copy PR tests; preserve unchanged sales assertions and retained guards.
  - TestCopyRejectedDocumentsRequiresActiveMasters L359: purchasing.CopyRequest(.
- frontend/features/purchasing/purchase-request-panel.tsx: Expose existing Copy action for CANCELLED too.
  - PurchaseRequestPanel L57: document.status === "REJECTED".
  - PurchaseRequestPanel L57: action(document, "copy").

Kontrak yang tetap dapat digunakan:

- No new route, request field, database column or role permission.
- Existing application delegation/domain Copy do not implement the changed guard.

## selection-challenge-unchanged-control

Identical BPMN control.

Before: experiments/simple-erp/selection-challenges/unchanged-control/before.bpmn

After: experiments/simple-erp/selection-challenges/unchanged-control/after.bpmn

Label: experiments/simple-erp/selection-challenges/unchanged-control/reference.json

Acuan perubahan sementara:

Tidak ada kebutuhan berubah; kontrol identik.

Kontrak yang tetap dapat digunakan:

- No requirement change.

## Aturan pengujian baru

- Freeze method and requirements before first inference.
- Use new requirements; group by baseline process and requirement family, including controls; never split near-duplicate mutations across development/test.
- Prepare source-based labels without candidates; record reviewer provenance.
- Do not tune on independent test results; retire exposed/tuned cases to development.
- Report per-case/per-family results and controls separately; unlabeled nodes are not negatives.

## Batas interpretasi

- Label dari satu review AI terhadap source; bukan review independen atau implementation diff after.
- Mapping before bukan daftar file wajib berubah. Requirement yang menentukan perubahan.
- Node/flow di luar subset bukan otomatis negatif. Label node tambahan dievaluasi terpisah.
- Source test terkait belum dijalankan; pilihan file test tetap provisional.
- Tidak ada hasil ranking/metrik baru atau model yang dijalankan dalam audit ini.
- Verifikasi konfigurasi runtime dan model hash pada hasil inference; freeze ini tidak membuktikan model siap.

Verifikasi dari root Mapper:

    python experiments/simple-erp/research_freeze.py --repo C:\Simple-ERP --output experiments/simple-erp/research-freeze --verify
