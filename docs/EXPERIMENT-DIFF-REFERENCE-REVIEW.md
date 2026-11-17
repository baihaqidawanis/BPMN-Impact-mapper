# Audit diff dan acuan evaluasi eksperimen

Tanggal: 2026-11-04. Parser dan graph diff produksi dipakai langsung. Acuan ditinjau lewat source, terpisah dari peringkat mapper. Tidak ada inference ke model lab.

## Paket aktif

| Skenario | Jumlah perubahan terdeteksi | Cakupan acuan |
|---|---:|---|
| add-and | 15 | Subset: satu perubahan utama saja |
| add-task | 3 | Subset: satu perubahan utama saja |
| add-xor | 4 | Subset: satu perubahan utama saja |
| combined | 20 | Subset: satu perubahan utama saja |
| unchanged-control | 0 | Lengkap, kontrol tanpa perubahan |

## Efek perapian paket lama

| File after | Diff sebelumnya | Diff saat ini | Entri tambahan |
|---|---:|---:|---|
| `experiments/history/simple-erp/current-staff/add-and/after.bpmn` | 15 | 19 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_6`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-task/after.bpmn` | 3 | 7 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_6`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/add-xor/after.bpmn` | 4 | 8 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_6`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/current-staff/combined/after.bpmn` | 18 | 24 | `ADD_TASK/PrepareMasterCheckInputs`, `MODIFY_SEQUENCE_FLOW/PrepareMasterCheckInputsCompleted`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_6`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_adj_submit_result_merge` |
| `experiments/history/simple-erp/p2p-transfer/checks/after.bpmn` | 15 | 19 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_submit_Flow_6`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_submit_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_pr_submit_result_merge` |
| `experiments/history/simple-erp/selection-challenges/change-condition/after.bpmn` | 3 | 6 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_copy_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_copy_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_pr_copy_result_merge` |
| `experiments/history/simple-erp/selection-challenges/remove-check/after.bpmn` | 6 | 9 | `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_copy_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_pr_copy_result_to_activity`, `REMOVE_GATEWAY/role_staff_l3_pr_copy_result_merge` |
| `experiments/history/simple-erp/staff-single-end/after.bpmn` | 22 | 28 | `ADD_TASK/PrepareMasterCheckInputs`, `MODIFY_SEQUENCE_FLOW/MergeToResponse`, `MODIFY_SEQUENCE_FLOW/PrepareMasterCheckInputsCompleted`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_5`, `MODIFY_SEQUENCE_FLOW/role_staff_l3_adj_submit_Flow_6`, `REMOVE_GATEWAY/ResponseMerge` |
| `experiments/history/simple-erp/structural-staff/add-or/after.bpmn` | 11 | 13 | `ADD_TASK/UseZeroDeltaResult`, `MODIFY_SEQUENCE_FLOW/UseZeroDeltaResultCompleted` |

Perubahan notasi tetap merupakan perubahan BPMN yang sah bagi parser. Jangan menghapusnya dari keluaran produk secara khusus untuk studi kasus. Untuk eksperimen terkontrol baru, gunakan pasangan current-staff-v2 dengan baseline yang sudah dirapikan di kedua sisi. Nilai historis dari pasangan lama tidak dicampur dengan pasangan baru.

## Review acuan berbasis source

- `frontend/features/inventory/inventory-adjustment-panel.tsx`: fungsi `action` memakai alasan hanya untuk reject/cancel/reverse; submit mengirim string kosong. Kebutuhan alasan submit memerlukan perubahan pengiriman dan validasi form.
- `backend/internal/inventory/adapter/postgres/store.go`: `TransitionAdjustment`, cabang submit, memeriksa DRAFT lalu mengganti status. Validasi alasan serta pemeriksaan ulang master untuk skenario terkait belum ada di cabang ini. Guard versi tetap digunakan.
- `backend/test/integration/operations_test.go`: lokasi pengujian alur operasi yang perlu mencakup alasan kosong/valid dan master yang berubah sebelum submit. Status MODIFY pada acuan berarti perubahan test yang diharapkan, bukan bukti test sudah dijalankan.

Acuan add-task/add-xor/combined tetap tiga file tersebut. Acuan add-and tetap store dan integration test. Tidak ditambahkan schema atau endpoint baru hanya karena ada task baru. Data/master dapat diperiksa memakai struktur yang ada. Ini keputusan desain source-derived, bukan kunci jawaban independen atau bukti hasil implementasi.

Semua manifest positif terkait telah memakai truth_scope=subset. Entri seperti CollectMasterContext, gateway join, dan flow pendukung belum memiliki label lengkap. File lain tidak otomatis negatif. Precision/MAP dari subset hanya diagnostic pada perubahan berlabel, bukan klaim ketepatan seluruh proses. Kontrol identik tetap scope=complete.

## Kesiapan paket AI

Tes adapter menggunakan struktur lengkap kasus combined, seluruh perubahan, evidence source, dan README; uji mock memastikan batas request dan identitas evidence. Tes job/planning juga mencakup koneksi terputus, resume, stop/cancel, batch gagal, dan hasil parsial. Ini menguji alur aplikasi; kualitas reasoning model nyata tetap perlu uji VPN.
