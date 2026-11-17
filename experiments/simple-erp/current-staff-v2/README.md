# Paket aktif: Staff Inventory setelah perapian gateway

Pakai folder ini untuk percobaan baru. Paket `current-staff` sebelumnya dan
`research-freeze` tetap menjadi snapshot historis; hasilnya tidak ditimpa.

Kelima `before.bpmn` adalah salinan byte-identik dari:
`docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn`
di Simple-ERP. Hash tersimpan di `snapshot.json` dan `manifest.json`. Diagram
lama tidak lagi diasumsikan memiliki node `result_merge`. Semua jalur gagal
tambahan menuju aktivitas respons yang memang ada pada baseline. Kasus gabungan
menyiapkan referensi master sebelum AND, sehingga tidak ada XOR langsung ke AND.

| Folder | Kebutuhan baru | Perubahan utama untuk review Qwen |
|---|---|---|
| add-task | Alasan wajib sebelum submit adjustment | ADD_TASK / ReasonReview |
| add-xor | Alasan ada -> lanjut; kosong -> hasil gagal | ADD_GATEWAY / ReasonDecision |
| add-and | Validasi ulang warehouse dan semua produk aktif | ADD_GATEWAY / MasterChecksSplit |
| combined | Alasan wajib, persiapan referensi, AND pemeriksaan master dan keputusan hasil | ADD_GATEWAY / ReasonDecision |
| unchanged-control | Tidak ada perubahan | Tidak ada perubahan untuk direview |

## Isian web

- Repository path: `C:\Simple-ERP`.
- Repository name: `Simple-ERP`.
- Revision: boleh kosong untuk working tree saat ini. `local-current-staff-v2`
  pada manifest adalah label snapshot, bukan klaim hash commit Git.
- Old BPMN: `before.bpmn` dari folder kasus yang dipilih. Boleh memakai file
  kanonis ERP di atas selama hash masih cocok.
- New BPMN: `after.bpmn` dari folder kasus yang sama.
- Product / Hybrid, top candidates 10 jika embedding lokal tersedia. Untuk
  pemeriksaan alur tanpa embedding, Research / BM25, top candidates 10.
- Rekomendasi pertama: **add-task** untuk tes singkat, atau **combined** untuk
  XOR dan AND sekaligus. Di Planning pilih **Focused review** dan buat sesi baru;
  semua perubahan diproses, bukan satu ADD_TASK pilihan manual.

Contoh file baru pertama:
`C:\BPMN-Impact-Mapper\experiments\simple-erp\current-staff-v2\add-task\after.bpmn`.

Canvas memakai satu pasangan. Library dapat menyimpan baseline/varian ini
sebagai versi yang berbeda. Jangan mencampur baseline baru dengan after dari
paket historis. Jangan upload reference/source-review/snapshot sebagai input
retrieval atau sebagai bukti yang diberikan kepada Qwen.

## Acuan evaluasi

`reference.json` dan `source-review.json` disiapkan sebelum inference, dari
inspeksi source. Panel sudah punya input alasan untuk aksi lain, tetapi submit
mengirim alasan kosong. Store memeriksa DRAFT/versi dan mengubah status; kebutuhan
baru memerlukan validasi tambahan. Tidak dibutuhkan endpoint atau schema baru.

Acuan ini **sementara, dari satu review source**, bukan ground truth independen
atau implementation diff nyata. Kasus positif hanya memberi label satu anchor
terpilih. Node lain, termasuk persiapan referensi pada kasus gabungan, belum
dilabeli sebagai keseluruhan impact. Semua hasil tidak boleh dinilai seolah
referensi mencakup seluruh proses. Kontrol identik memiliki cakupan lengkap.
Jangan menjadikan beberapa representasi kebutuhan yang sama sebagai bukti
generalisasi ke berbagai repository.

Validasi ulang sebelum eksperimen:

```powershell
python experiments/tools/verify_current_staff_pack.py --repo C:\Simple-ERP --pack experiments/simple-erp/current-staff-v2
```

Verifier gagal jika baseline/source/reference drift, flow menggantung, diagram
kurang shape/edge, atau konvensi satu start/end dan gateway langsung dilanggar.
Regenerasi harus ke folder baru untuk menjaga snapshot yang sudah dipakai.

Pemeriksaan awal BM25 dijalankan dua kali per kasus: JSON identik, semua anchor
utama ditemukan, setiap kandidat memiliki evidence, kontrol nol perubahan.
Ini pemeriksaan fungsi dan determinisme; bukan pengukuran kualitas Semantic,
Hybrid atau Qwen. Laporan smoke ada di `outputs/pre-qwen-ready/` dan tidak
di-commit. Hasil/matriks historis di Research tidak otomatis diganti.

## Perapian diagram 2026-11-03

Empat after aktif memakai jalur ortogonal yang menghindari shape task/gateway.
Cabang split memakai port berbeda supaya tidak terlihat menyatu. AND join tetap
memiliki dua masukan dan satu keluaran: itu penggabungan, bukan cabang yang hilang.
XML proses, kondisi, ID dan lane tidak berubah. Baseline dan unchanged-control
utuh. Hash diagram pada manifest/snapshot diperbarui karena byte XML berubah;
laporan historis tetap mengacu pasangan lama. Backup sebelum routing ada di
outputs lokal. Unggah ulang after untuk percobaan baru.

Verifier juga memeriksa kardinalitas split/join dan memastikan garis diagram
aktif tidak menembus node lain. Generator memakai router yang sama; regenerasi
ke folder baru tetap dapat direproduksi. Paket historis belum dirapikan ulang.

## Konfirmasi ulang 2026-11-04

- Empat after aktif dirapikan pada posisi label alur, agar tidak menimpa task, nama gateway, atau batas lane. Alur, kondisi, ID dan tipe task tetap sama.
- Sepuluh file before/after berhasil diimpor dan dirender dengan bpmn.io terpasang, tanpa warning dan tanpa elemen diagram hilang. Gambar hasil cek ada di `outputs/diagram-review/` (lokal, tidak di-commit).
- Pemeriksaan 23 after di seluruh paket tidak menemukan gateway dengan masukan/keluaran hilang atau gateway satu-masuk-satu-keluar. Ini bukan klaim semua layout historis sudah dirapikan. Paket aktif untuk percobaan baru tetap folder ini.
- Baseline masih byte-identik dengan ERP. Routing dapat dijalankan ulang tanpa perubahan byte lebih lanjut.

Verifikasi render tanpa VPN:

```powershell
node scripts/verify-experiment-diagrams.cjs
```

Untuk percobaan lengkap: before dari ERP pada path di atas, after dari
`experiments/simple-erp/current-staff-v2/combined/after.bpmn`. Upload ulang after
karena layout XML dan fingerprint berubah. Jalankan Mapping lalu sesi AI baru
sesudah restart backend dan koneksi tunnel tersedia. Gunakan Zoom in untuk
membaca kasus combined yang lebar; seluruh diagram tetap ada.
