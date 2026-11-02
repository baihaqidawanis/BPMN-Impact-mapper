# Perapian BPMN Staff — 29 Oktober 2026

Kelanjutan 30 Oktober: [konvensi Supervisor dan Super Admin](ROLE-BPMN-NORMALIZATION.md).
Dokumen ini mencatat pekerjaan Staff sebelumnya; generator selanjutnya mencakup
ketiga role.

File kanonis berada di `C:\Simple-ERP\docs\bpm-haqi\roles\staff`.
32 file diperiksa; 28 diganti di tempat, 4 sudah sesuai dan tetap identik.
Tidak membuat varian baru di folder Staff. Role Supervisor/Super Admin dan
source aplikasi ERP tidak diubah.

## Hasil

- Seluruh diagram memiliki satu start dan satu end.
- Seluruh join gateway langsung dilanjutkan aktivitas.
- L3 memakai send task `Kirim respons hasil aksi (sukses atau gagal)` di lane
  Sistem Simple ERP. Handler inventory, purchasing, finance, sales dan masterdata
  sudah mengirim hasil atau pesan error; ini bukan tambahan operasi bisnis.
- L0–L2 memakai user task `Periksa hasil dan status proses` di lane Staff.
  Membaca hasil/status yang ditampilkan UI, bukan approval baru atau request baru.
- Status seperti REJECTED, belum submitted, belum posted, dan gagal/tidak commit
  tetap pada cabang menuju XOR merge. End umum tidak berarti semua cabang sukses.
- Semua kondisi flow, atribut task/gateway yang lama, role, `calledElement` dan
  binding namespace antarlevel dipertahankan. Tata letak node bisnis lama tetap;
  bagian akhir diagram diperpanjang untuk merge, aktivitas dan end.

Dasar inspeksi source: handler HTTP masing-masing modul memanggil `respond` pada
jalur berhasil maupun error. Finance memakai helper `result`. Frontend PR,
Inventory, Finance dan Master Data menampilkan status/data serta notice/error.
Ini pemeriksaan struktur dan source statis, bukan pembuktian seluruh jalur runtime.

## Verifikasi

- Checker sebelum: 23 SINGLE_END dan 5 ACTIVITY_AFTER_JOIN.
- Checker final: 32 parsed, nol temuan struktur/review/konvensi.
- 32 tes browser final lulus: import tanpa warning, seluruh shape muncul dan
  panjang seluruh edge lebih dari nol. Lima diagram perwakilan diperiksa visual.
- Verifier perbandingan memeriksa kondisi, actor, QName `calledElement`, tata
  letak node lama dan isolasi jalur keluar dari operasi bisnis asli.
- Semua 32 hash file kanonis cocok dengan versi yang diuji di browser.
- 91 hash di luar scope tetap sama, mencakup BPMN role lain, source yang diperiksa
  dan generator. Frontend typecheck lulus.

Backup lengkap, versi review, manifest hash serta audit sebelum/selepas apply:
`outputs/staff-normalization-2026-10-29-final/` di Mapper. Folder ini di-ignore Git.
Script: `experiments/simple-erp/normalize_staff_bpmn.py`; verifier:
`experiments/simple-erp/verify_staff_normalization.py`.

## Dampak ke eksperimen

Baseline kanonis berubah secara notasi. Mutasi empat skenario lama tetap memakai
baseline lama yang kini disimpan di `experiments/simple-erp/frozen-staff-baselines`.
Dua manifest Staff dan panduan empat skenario diarahkan ke salinan tersebut;
hash input, ground truth dan skor tersimpan tidak diubah atau dihitung ulang.

Untuk skenario baru: upload baseline kanonis terbaru, duplikasi/edit lewat Canvas,
lalu bandingkan pasangan yang sama-sama menggunakan konvensi baru.

Kelanjutan: generator `Simple-ERP/scripts/bpm-haqi/build.py` sekarang memakai
helper `staff_convention.py` untuk keluaran Staff. XML dinormalisasi dan preview
SVG diturunkan dari XML yang sama. Metadata jumlah node/flow memakai hasil final.
Uji dilakukan pada salinan generator di Mapper: 32 BPMN Staff dan preview SVG
berhasil dibuat; checker konvensi bersih. Generator produksi belum dijalankan
ulang pada folder kanonis, sehingga tidak menimpa perapian diagram yang ada.
Role selain Staff memakai perilaku generator semula. Backup generator berada di
`outputs/generator-staff-review/original-build.py`.
