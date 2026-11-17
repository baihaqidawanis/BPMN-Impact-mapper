# Mapping BPMN current: Staff adjustment submit

Tanggal review: 2026-11-05. Review statis sementara; bukan pembuktian runtime 1:1.

BPMN before: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn`. Byte-identik dengan `current-staff-v2/add-task/before.bpmn`.

Data utama: [adj-submit-current.json](adj-submit-current.json). Acuan ini hanya untuk penelitian/evaluasi; jangan diberikan ke retrieval atau agent sebagai kunci jawaban.

## Pemetaan elemen yang sudah ada

| Elemen | Tipe / aktor | Implementasi yang diperiksa | Makna hubungan |
|---|---|---|---|
| `role_staff_l3_adj_submit_start`<br>Kebutuhan aksi | startEvent / Staff | `frontend/features/inventory/inventory-adjustment-panel.tsx` / `InventoryAdjustmentPanel` (baris 19) | Entry boundary; no standalone runtime event |
| `role_staff_l3_adj_submit_input`<br>Kirim adjustment untuk review | userTask / Staff | `frontend/features/inventory/inventory-adjustment-panel.tsx` / `InventoryAdjustmentPanel.action` (baris 16, 19) | Direct user action |
| `role_staff_l3_adj_submit_check`<br>Lock adjustment DRAFT; versi cocok | serviceTask / Sistem Simple ERP | `backend/internal/inventory/adapter/postgres/store.go` / `Store.TransitionAdjustment` (baris 121, 125, 127, 128, 134, 135) | Direct validation |
| `role_staff_l3_adj_submit_valid`<br>Syarat terpenuhi? | exclusiveGateway / Sistem Simple ERP | `backend/internal/inventory/adapter/postgres/store.go` / `Store.TransitionAdjustment` (baris 128, 129, 135, 136) | Branch projection of imperative guards; not a gateway runtime |
| `role_staff_l3_adj_submit_effect0`<br>Ubah adjustment SUBMITTED | serviceTask / Sistem Simple ERP | `backend/internal/inventory/adapter/postgres/store.go` / `Store.TransitionAdjustment` (baris 138, 183, 184, 185) | Direct state effect |
| `role_staff_l3_adj_submit_effect1`<br>Audit INV-ADJ-002; commit | serviceTask / Sistem Simple ERP | `backend/internal/inventory/adapter/postgres/store.go` / `Store.TransitionAdjustment` (baris 132, 187, 190, 193, 194)<br>`backend/internal/inventory/adapter/postgres/store.go` / `audit` (baris 24, 25) | Direct audit and commit |
| `role_staff_l3_adj_submit_done`<br>Respons dikirim | endEvent / Sistem Simple ERP | `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` (baris 107, 108, 110, 111) | End boundary; no extra business effect |
| `role_staff_l3_adj_submit_result_activity`<br>Kirim respons hasil aksi (sukses atau gagal) | sendTask / Sistem Simple ERP | `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` (baris 104, 106, 107, 110)<br>`backend/internal/inventory/adapter/http/handler.go` / `respond` (baris 40, 41, 42, 43) | Presentation of existing HTTP response; not a message broker |

Satu function boleh menerapkan beberapa task. Start/end tidak memerlukan function tersendiri. Gateway dipetakan ke kondisi `if` dan jalur return, bukan runtime BPMN. `sendTask` di sini menggambarkan respons HTTP, bukan pengiriman message broker.

## Jalur operasi pendukung

`InventoryAdjustmentPanel.action("submit") -> POST /inventory-adjustments/{id}/submit -> Handler.action -> Service.Transition -> Store.TransitionAdjustment -> audit/commit -> respons HTTP`.

| Hubungan | File / function / bukti |
|---|---|
| existing create and transition routes are distinct | `backend/internal/inventory/adapter/http/handler.go` / `Handler.Register` (baris 18, 20) |
| action routing, role gate and existing reason payload | `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` (baris 81, 83, 87, 88, 89, 104) |
| existing interface and forwarding; no new argument needed | `backend/internal/inventory/application/service.go` / `Service.Transition` (baris 35, 50, 51) |
| STAFF and SUPER_ADMIN may submit | `backend/internal/auth/domain/role.go` / `CanCreate` (baris 13, 14) |
| existing successful submit with empty reason; tests not executed | `backend/test/integration/operations_test.go` / `TestInventoryApprovalRejectsChangedSnapshot` (baris 253, 262) |
| existing inventory submit success paths; tests not executed | `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` (baris 274, 283, 309) |
| reason already nullable text; no new column needed | `backend/migrations/000005_operations.up.sql` / `inventory_adjustments table` (baris 36) |
| related domain model; not called on the traced Service.Transition -> PostgreSQL runtime path | `backend/internal/inventory/domain/adjustment.go` / `Adjustment.Submit` (baris 14, 15, 16, 17, 23, 24, 27) |

## Sequence flow dari XML current

| Flow | Dari | Ke | Label |
|---|---|---|
| `role_staff_l3_adj_submit_Flow_0` | `role_staff_l3_adj_submit_start` | `role_staff_l3_adj_submit_input` |  |
| `role_staff_l3_adj_submit_Flow_1` | `role_staff_l3_adj_submit_input` | `role_staff_l3_adj_submit_check` |  |
| `role_staff_l3_adj_submit_Flow_2` | `role_staff_l3_adj_submit_check` | `role_staff_l3_adj_submit_valid` |  |
| `role_staff_l3_adj_submit_Flow_3` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_effect0` | Ya |
| `role_staff_l3_adj_submit_Flow_4` | `role_staff_l3_adj_submit_effect0` | `role_staff_l3_adj_submit_effect1` |  |
| `role_staff_l3_adj_submit_Flow_5` | `role_staff_l3_adj_submit_effect1` | `role_staff_l3_adj_submit_result_activity` | Aksi selesai; hasil diterima |
| `role_staff_l3_adj_submit_Flow_6` | `role_staff_l3_adj_submit_valid` | `role_staff_l3_adj_submit_result_activity` | Tidak; Ditolak / gagal; tidak commit |
| `role_staff_l3_adj_submit_result_to_end` | `role_staff_l3_adj_submit_result_activity` | `role_staff_l3_adj_submit_done` |  |

## Batas kesesuaian before dengan kode

- Kondisi DRAFT, version, row lock, status SUBMITTED, audit dan response cocok dengan kode yang ditelusuri.
- Baseline belum meminta alasan wajib ketika submit: UI mengirim string kosong untuk aksi submit; guard backend hanya memeriksa DRAFT/version pada jalur tersebut.
- Error query, update, audit dan commit tidak semuanya digambar sebagai cabang tersendiri. Gateway adalah ringkasan keputusan, sehingga jangan klaim setiap exit kode sudah dimodelkan 1:1.
- Kontrol autentikasi/RBAC merupakan konteks request pendukung, bukan task tambahan dalam diagram ini.
- Domain `Adjustment.Submit` adalah implementasi terkait tetapi tidak dipanggil oleh jalur service/store yang ditelusuri. Jangan otomatis menganggapnya wajib berubah untuk route ini.

## Perbandingan dengan after add-task

After: `current-staff-v2/add-task/after.bpmn`. Task baru `ReasonReview` (userTask) disisipkan di antara input dan pemeriksaan lock. `Flow_1` diarahkan ke task baru; `ReasonChecked` menghubungkannya ke pemeriksaan lama.

| Bagian | Before current | Kebutuhan after | Klasifikasi dampak |
|---|---|---|---|
| Frontend `frontend/features/inventory/inventory-adjustment-panel.tsx` | Submit selalu mengirim reason kosong | Isi alasan, trim/validasi, lalu kirim melalui aksi submit | MODIFY, acuan utama |
| Backend `backend/internal/inventory/adapter/postgres/store.go` | Submit menerima reason kosong; kolom dan parameter sudah ada | Guard alasan hanya untuk aksi submit sebelum perubahan state; pertahankan lock/version | MODIFY, acuan utama |
| Tes `backend/test/integration/operations_test.go` | Beberapa happy path submit inventory mengirim alasan kosong | Perbarui happy path inventory, tambah kasus kosong/whitespace dan efek state/audit | MODIFY, acuan utama |
| Handler `backend/internal/inventory/adapter/http/handler.go` dan service `backend/internal/inventory/application/service.go` | Sudah menerima/meneruskan reason | Verifikasi forwarding; signature/endpoint baru tidak dibutuhkan oleh skenario ini | RELATED, tidak wajib diubah |
| Schema `backend/migrations/000005_operations.up.sql` | Kolom reason sudah tersedia | Tetap izinkan draft sesuai aturan; alasan wajib pada submit tidak membuktikan kebutuhan NOT NULL global | RELATED, tidak wajib diubah |
| Domain `backend/internal/inventory/domain/adjustment.go` | Submit model terpisah memeriksa status | Perlu keputusan konsistensi domain bila ingin menyelaraskan; tidak terbukti diperlukan pada route yang ditelusuri | OPTIONAL / bukan acuan minimum |

Acuan after tetap [reference.json](../current-staff-v2/add-task/reference.json): tiga file untuk satu perubahan utama ADD_TASK/ReasonReview. Tabel before memiliki lebih banyak file terkait; keterkaitan bukan kewajiban mengubah semua file.

## Change plan Qwen yang tersedia

Plan terbaru masih `change-plan-83516040b1309d2943c992ee3861609d.md`, prompt v6, status NEEDS_REVIEW. Belum ada keluaran baru setelah perbaikan v7.

- Penilaian kualitas manual: **55/100**; rubrik dan metrik tersimpan di [laporan evaluasi](../../../docs/AI-EVALUATION-2026-11-05.md).
- Pemilihan enam file mempertahankan tiga file acuan: **precision 50%, recall 100%, F1 66,67%**.
- Tiga tambahan: domain adjustment, dokumentasi business rules, migration. Tidak semuanya dibuktikan wajib berubah oleh skenario.
- Kekeliruan masih ada: string dibandingkan nil, guard semua aksi, create tercampur submit, NOT NULL tanpa kebutuhan schema, dan tes belum cukup spesifik.
- AP/MAP dan RR/MRR Qwen tidak tersedia karena daftar plan tidak punya ranking relevansi. Hybrid untuk query ini memiliki AP=1 dan RR=1; bukan hasil agregat seluruh skenario.

## Verifikasi dan provenance

JSON menyimpan hash BPMN dan source/test yang dirujuk, seluruh node (termasuk gateway/start/end) serta sequence flow current. Hash dan excerpt harus dicek ulang bila ERP berubah. Source/test hanya dibaca.
