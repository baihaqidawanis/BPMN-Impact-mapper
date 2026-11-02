# Peta eksperimen

**Freeze sebelum LLM:** [paket terkunci dan audit acuan perubahan](simple-erp/research-freeze/README.md).
Sepuluh kasus lama adalah development/regression, bukan pengujian independen.
Paket mengikat BPMN, label, source review dan kode metode melalui hash; label
existing tetap dipertahankan. Kasus uji independen baru belum disiapkan.

**Paket aktif:** [Staff setelah perapian gateway](simple-erp/current-staff-v2/README.md).
Paket current-staff sebelumnya tetap menjadi snapshot historis.
Baseline paket aktif adalah salinan Staff kanonis setelah normalisasi.

**Acuan before:** [mapping tiga modul × tiga role](simple-erp/baseline-traceability/all-roles/README.md),
berisi 72 proses L3 / 354 aktivitas dengan JSON dan tabel task -> file/function,
bukti source, role guard dan indeks 40 diagram L0–L2. Snapshot Staff P2P awal
tetap disimpan. Bukan input retrieval atau label otomatis file yang perlu berubah.

**Paket historis:** [empat skenario lama](simple-erp/PANDUAN-EMPAT-SKENARIO.md).
Paket lama memakai `frozen-staff-baselines`, bukan file kanonis terbaru.
Hasil lamanya tetap disimpan sebagai hasil pengembangan, tidak diganti oleh
hasil baru atau diklaim sebagai pengujian independen.

**Investigasi metode:** [context ablation](simple-erp/CONTEXT-STUDY.md),
[konteks baseline beberapa langkah](simple-erp/BASELINE-CONTEXT-STUDY.md),
[transfer P2P dan seleksi kandidat](simple-erp/p2p-transfer/README.md),
[uji penghapusan/kondisi dan audit seleksi](simple-erp/selection-challenges/README.md),
[uji query kondisi](simple-erp/CONDITION-FOCUS-STUDY.md),
[uji indeks potongan teks](simple-erp/code-window-study/README.md),
[file fusion](simple-erp/FILE-FUSION-2026-10-29.md),
[audit Inventory](simple-erp/INVENTORY-ERROR-AUDIT.md).

**Cakupan struktur tambahan:** [tujuh pasangan Staff](simple-erp/structural-staff/README.md)
dan [contoh gabungan awal](simple-erp/staff-single-end/README.md).
Jangan mencampur baseline antar paket. Berhasil memparse/render diagram tidak
berarti akurasi mapping sudah terukur.

Output run dan timing berada di `outputs/` dan tidak di-commit. Script Python di
`simple-erp/` menyiapkan fixture/referensi dan audit; engine produksi tetap Go.
