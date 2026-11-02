# Review hasil Qwen inventory — 2 November 2026

Input yang diperiksa: `change-plan-32b42ff02364e3992226219a161dea3e.md`
dan `agent-context-d0897ce0da2185870b3911defa4cc334.json` di root.
Output asli tidak diedit. Tidak ada pemanggilan lab atau perubahan ERP.

## Kesimpulan

Pemilihan tiga file masuk akal: adapter inventory, panel frontend, dan integration
test. Rencana sudah mempunyai proposed changes dan acceptance checks, tetapi
belum layak diikuti langsung. Status `NEEDS_REVIEW` tepat. Ini screening statis,
bukan pengukuran akurasi independen atau hasil implementasi yang telah diuji.

## Temuan source

1. Frontend sudah mempunyai state `reason` dan input alasan. Function `action`
   mengirim alasan untuk reject/cancel/reverse, tetapi mengirim string kosong
   untuk submit. Rencana seharusnya memanfaatkan input tersebut dan mengubah
   validasi serta pengiriman submit; tidak langsung menambah input kedua.
2. Backend `TransitionAdjustment` melayani beberapa action. Persyaratan BPMN
   berlaku sebelum submit. Validasi tanpa pembatas action di awal function
   berpotensi mengubah approve/post dan action lain di luar scope.
3. Lokasi usulan backend keliru: rollback defer berada sebelum query, bukan
   sesudah query. Jangan menyisipkan return baru sebelum defer terpasang setelah
   transaksi dimulai. Pertahankan lifecycle transaksi dan guard state/version.
4. Test perlu kasus alasan valid, kosong, dan whitespace; kegagalan harus menjaga
   state/version tetap. Mengganti data test positif saja tidak cukup, dan test
   negatif tidak seharusnya menjadi pilihan alternatif yang boleh dilewati.
5. Beberapa acceptance checks salah ditempatkan lintas file; misalnya tampilan
   input UI muncul dalam bagian backend. Plan juga masih memuat snippet kode,
   padahal tujuan produk adalah rencana perubahan dalam bahasa natural.
6. Review issue berakhir pada `action(`. Checkpoint menyimpan kalimat terpotong
   yang sama; ini bukan pemotongan oleh renderer Markdown.

Sesi menjalankan 15 turn, delapan tool, dua revisi. Agent context menunjukkan
banyak bukti backend dari awal function dan potongan JSX sebelum input alasan.
Model belum memperoleh konteks yang cukup untuk membedakan input yang tidak ada
dengan input yang sudah ada tetapi belum dikirim saat submit.

## Agent context itu apa?

JSON tersebut adalah paket bukti untuk sesi agent: satu perubahan BPMN, kandidat
awal, skor retrieval, dan cuplikan source beserta ID bukti. Isinya diperbarui
dengan hasil lookup. ID pada bagian Evidence dalam plan merujuk paket ini.
Ini bukan source repository lengkap, ground truth, ataupun hasil implementasi.
Paket memuat cuplikan source; periksa isinya sebelum dibagikan ke luar.

## Perbaikan offline

Ditambahkan `inspect_source` untuk membaca beberapa bagian relevan dalam satu
file yang sudah disetujui ingestion. Query berasal dari label perubahan BPMN.
Pemilihan deterministik mencakup baris terpisah dan bagian tambahan dari baris
panjang yang membawa kata query berbeda. Maksimal enam bukti baru, masing-masing
240 karakter, sehingga dua bukti awal tetap muat dalam batas delapan.

Inspect menggunakan pencarian teks, tidak menyimpulkan dependensi atau absennya
validasi. Source reading/paging tetap tersedia untuk memperluas cuplikan.
Bootstrap analyst memakai inspection tersebut menggantikan delapan baris awal
yang sering hanya berisi transaksi dan deklarasi. Ranking mapping tidak diubah.

Pemeriksaan read-only pada frontend ERP membuktikan bukti baru dapat mencakup
state alasan, pengiriman `actionReason`, submit, serta `input value={reason}`
yang berada jauh di dalam baris JSX panjang. Algoritma tidak mempunyai aturan
khusus nama file, modul, task ID, atau istilah bisnis ERP.

Tes sintetis memeriksa bagian yang jauh dari awal file, baris panjang, ordering
deterministik, batas excerpt dan penolakan path. Tes integrasi HTTP offline tetap
memeriksa sesi, validasi output, penyimpanan dan unduhan. File ekspor root kini
di-ignore Git agar tidak tidak sengaja ikut commit.

## Langkah selanjutnya

Tidak perlu mengubah skenario atau menambahkan metode retrieval saat ini. Saat
VPN tersedia, restart backend dan mulai sesi Qwen baru dengan input yang sama.
Periksa apakah plan memakai input yang ada, membatasi aturan pada submit, dan
menyertakan pengujian negatif serta perlindungan state/version. Belum ada bukti
bahwa kualitas jawaban Qwen meningkat sampai percobaan ulang tersebut dilakukan.
