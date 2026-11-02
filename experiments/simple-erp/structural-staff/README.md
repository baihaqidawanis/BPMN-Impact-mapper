# Skenario struktur Staff: paket pengujian lanjutan

Contoh tambahan sesuai konvensi satu start/end, aktivitas setelah join, dan
mutasi gabungan XOR + AND + task tersedia di
[`../staff-single-end/README.md`](../staff-single-end/README.md).
Paket lama ini tetap beku; jangan mencampur baseline dari kedua paket.

Repository path web: `C:\Simple-ERP`. Name: `Simple-ERP`. Revision boleh kosong
untuk file lokal sekarang. Product / Hybrid, top candidates 10. Satu pasangan
per analisis; jangan Load synthetic demo.

Folder utama:
`C:\BPMN-Impact-Mapper\experiments\simple-erp\structural-staff`.
Di setiap subfolder, upload `before.bpmn` sebagai Baseline dan `after.bpmn`
sebagai New BPMN. Baseline adalah salinan BPMN Staff dari Simple-ERP; repository
kasus tidak diubah. File before/after boleh dibuka di bpmn.io secara terpisah.

| Subfolder | Perubahan yang diusulkan | Acuan file sementara |
|---|---|---|
| remove-task | Hilangkan langkah pengisian alasan pembatalan adjustment; cancellation tetap ada, guard actor/status/version tetap ada | Inventory panel, inventory store, operations integration test |
| add-xor | Wajib alasan sebelum submit; satu jalur lolos atau satu jalur ditolak | Inventory panel, inventory store, operations integration test |
| remove-gateway-control | Hapus gateway visual yes/no, tampilkan validasi gagal sebagai boundary error pada service task; aturan bisnis sama | Kosong: kontrol tidak perlu perubahan source |
| remove-gateway | Hapus keputusan eksplisit DRAFT-only; service validation mengizinkan DRAFT/APPROVED submit ulang, reset reviewer, wajib approval baru. Status lain, akses, version guard dan rollback tetap dijaga | Inventory store, operations integration test |
| move-role | Submit adjustment hanya Supervisor, dengan override Super Admin yang ada; Staff tetap dapat membuat draft | Inventory panel, inventory HTTP handler, operations integration test |
| add-and | Periksa warehouse aktif DAN seluruh produk aktif sebelum submit; keduanya wajib selesai, gagal berarti rollback | Inventory store, operations integration test |
| add-or | Delta positif memerlukan cek produk aktif; delta negatif cek stok tersedia; campuran menjalankan keduanya dan menunggu semua yang dipilih. Delta nol memakai default bypass | Inventory store, operations integration test |

## Arti diagram dan cakupan acuan

`requirements-and-reference.json` di tiap subfolder menyimpan kebutuhan bisnis,
hash baseline/mutasi, daftar file yang diperkirakan berubah, dan hash source
yang diperiksa. Ini **acuan union file untuk satu skenario**, belum kontrak
evaluasi atomic-change JSON yang dipakai CLI evaluate. Jangan upload file ini
sebagai mapping configuration. Belum ada inference pada tujuh kasus baru ini.

Daftar ini merupakan provisional single-source review, bukan bukti implementasi
atau adjudikasi independen. File terkait tidak otomatis wajib berubah. Tidak
menambahkan nama file baru/migration ke acuan tanpa kebutuhan yang jelas.
Pada AND/OR, output bisa berisi beberapa perubahan task/gateway untuk kebutuhan
yang sama: jangan menggandakan hit union file atau mencampur metrik ini dengan
AP per perubahan utama pada empat skenario sebelumnya.

Pertimbangan source:

- Inventory panel mengirim reason kosong untuk submit, dan membatasi tombol
  dengan canCreate/canApprove. Gunakan aturan lokal aksi submit pada kasus role;
  jangan mengubah izin global CanCreate yang mempengaruhi modul lain.
- Inventory handler memeriksa CanCreate untuk submit. Kasus role perlu kebijakan
  khusus submit; handler sudah memiliki request reason sehingga kasus XOR tidak
  otomatis memerlukan endpoint/field API baru.
- TransitionAdjustment saat submit memeriksa DRAFT lalu mengubah SUBMITTED.
  Kolom reason sudah disimpan dan diaudit, sehingga kasus alasan tidak otomatis
  membutuhkan migration. Aktifnya master data pada waktu create tidak menjamin
  masih aktif pada waktu submit; AND mengusulkan validasi ulang.
- Test operations memiliki alur submit dengan reason kosong dan recovery stok.
  Ia menjadi lokasi awal untuk menguji kebutuhan baru, bukan bukti testnya
  sudah membuktikan perubahan yang belum diimplementasikan.

## Verifikasi dan batasnya

Tes Go memeriksa parser, perubahan utama, shape untuk tiap node, dan waypoint
untuk tiap flow. Import XML bpmn-moddle lokal menerima 12 file awal tanpa warning;
pasangan penghapusan gateway positif ditambahkan kemudian dan diuji terpisah.
Enam tes browser pertama dan satu tes tambahan lulus: seluruh shape/flow
terimpor, garis memiliki panjang positif, dan tidak ada warning import.
Ini belum pembuktian formal token semantics maupun implementasi concurrency;
AND tidak otomatis berarti perlu goroutine. Pemeriksaan browser dicatat
terpisah dalam laporan pengerjaan, jangan disimpulkan dari XML yang valid.

Untuk rencana Markdown berikutnya, jelaskan kebutuhan bisnis, bukti source,
usulan perubahan per layer, test, dan ketidakpastian. Qwen belum diaktifkan oleh
paket ini. Skenario Supervisor adalah usulan perubahan tanggung jawab dari
baseline Staff; bukan klaim baseline Supervisor sudah divalidasi menyeluruh.
