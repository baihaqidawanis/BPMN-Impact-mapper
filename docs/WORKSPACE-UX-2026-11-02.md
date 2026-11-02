# Workspace UI/UX — 2 November 2026

Mapping dan Planning sekarang menjadi dua langkah pada satu workspace analisis.
Operator tidak perlu berpindah ke halaman konfigurasi kedua atau mengetik ulang
repository path setelah mapping. Tidak ada perubahan metode, scoring, atau
orchestration model pada perapian UI ini.

## Navigasi

| Menu | Kegunaan |
|---|---|
| Workspace | Input repository/BPMN, hasil Mapping, lalu Planning dan review Qwen. |
| BPMN Canvas | Melihat baseline dan mengedit salinan BPMN baru; menerapkan draft ke Mapping. |
| Saved analyses | Membuka laporan tersimpan tanpa menjalankan ulang analisis. |
| Research | Eksperimen metode dan evaluasi dengan acuan; muncul hanya saat RND diaktifkan. |
| Guide | Penjelasan metode, evidence, skor dan cara membaca hasil. |

Navigasi atas dipertahankan agar Canvas dan workspace dua kolom tetap punya
ruang horizontal. Workspace memiliki kontrol langkah Mapping/Planning sendiri.
Pergantian langkah mempertahankan state review yang sudah dibuka; tidak otomatis
memulai inference. Tombol Run Qwen tetap tindakan terpisah dan eksplisit.

## Layout dan data bersama

Panel input desktop memakai sticky positioning dengan tinggi sesuai viewport dan
scroll internal. Hasil di kanan dapat digulir sementara konfigurasi tetap dapat
diakses. Pada layar kecil panel kembali mengikuti alur dokumen satu kolom.
Header navigasi dan kontrol langkah tetap terjangkau saat menggulir.

Repository path ditangkap ketika mapping berhasil dan disimpan **per analysis ID**
di browser perangkat ini. Planning menampilkan path tersebut dengan opsi Change
path. Nama repository dan revision dapat dipulihkan dari metadata laporan publik.
Path tidak ditambahkan ke canonical report, shareable URL atau API history.

Laporan lama yang belum mempunyai path lokal tersimpan perlu diisi satu kali.
Path analisis berbeda tidak saling dipinjam. Bila penyimpanan browser diblokir,
path tetap tersedia selama sesi halaman berjalan, tetapi mungkin perlu diisi lagi
setelah refresh. File upload dan draft Canvas tetap tidak dipulihkan otomatis
setelah refresh; laporan tersimpan dan review yang sudah berjalan tetap tersedia.

## Arah visual

Menggunakan font SF/system yang sudah ada, warna netral dan aksen biru untuk
aksi/fokus. Mengurangi bobot badge nomor, bayangan dan uppercase berlebihan.
Hasil tetap berupa daftar perubahan dan file dengan pemisah yang jelas.
Planning menampilkan review dan rencana lebih dahulu; source context dan ekspor
audit dilipat agar tidak menduplikasi seluruh hasil Mapping di tampilan awal.

Referensi yang diperiksa: dokumentasi resmi https://www.tasteskill.dev/docs dan
https://www.tasteskill.dev/. Tidak memasang skill, dependency, font, atau library
baru. Perubahan menggunakan CSS Modules dan React/Next.js yang sudah terpasang.
Akses GitHub tidak diperlukan.

## Verifikasi

Delapan tes browser lulus: sticky workspace dan auto-path/refresh, review Qwen
tiruan tanpa duplicate start, Canvas-first/mobile, demo/unduhan evidence,
pemulihan saved analysis, pemulihan laporan hilang, perpindahan saved report saat
restore lama tertunda, dan source-caller context. Pemeriksaan desktop 1440px dan
mobile 390px tidak menemukan overflow horizontal dalam alur yang diuji.

Typecheck dan git diff --check lulus. Screenshot desktop, Planning dan mobile
ada di `web/test-results/workspace-flow-*/`. Tidak ada model lab yang dipanggil;
tes model memakai respons tiruan dan test mapping memakai fixture lokal.

Setelah memperbarui source, refresh frontend dev yang berjalan. Bila memakai
production build, build dan restart frontend diperlukan. Backend tidak perlu
direstart untuk perubahan UI ini; fitur full-repository yang ditambahkan
sebelumnya tetap membutuhkan backend versi terbaru.


## Encoding, Canvas dan Research

Karakter rusak pada label loading dan pemisah metadata diperbaiki di source UI.
Jalankan `npm --prefix web run check:encoding` untuk memeriksa UTF-8 dan pola
mojibake yang dikenal. Pemeriksaan ini juga masuk ke verify-agents.ps1.
Before dan After sekarang memiliki border, header dan jarak kartu masing-masing.
Kontrol pan/zoom, penanda perubahan dan expanded editor tetap tersedia.

Research adalah ruang eksperimen: ringkasan benchmark, batch skenario/metode,
review acuan, dan evaluasi terhadap expected files. Batch menjalankan tiap
pasangan BPMN dan metode sebagai laporan terpisah; shared baseline dengan banyak
file baru berarti beberapa alternatif skenario. Batch tidak menggabungkan
seluruh BPMN menjadi satu proses dan tidak menjalankan Qwen.

Bulk multi-process masih usulan, belum diimplementasikan. Rekomendasi: daftar
pasangan before/after dengan identitas proses yang jelas; Canvas hanya membuka
satu pasangan aktif. Jangan menampilkan banyak diagram sekaligus atau otomatis
mencocokkan level proses yang berbeda. Untuk validasi TA sekarang, pertahankan
satu pasangan per analisis.

Verifikasi tambahan: suite Chrome editor, batch dan workspace-flow lulus;
encoding guard dan typecheck lulus. Screenshot separate-bpmn-cards.png ada di
web/test-results/editor-canvas-shows-before-*/. Tidak ada panggilan model lab.
