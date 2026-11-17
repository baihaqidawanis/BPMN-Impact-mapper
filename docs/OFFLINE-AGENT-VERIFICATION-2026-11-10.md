# Verifikasi offline agen ? 10 November 2026

## Status

Perbaikan program sudah diverifikasi offline. Kualitas rencana dari model lab BELUM dinyatakan selesai. Tidak ada inferensi lab, server FE/BE, browser, unduhan dependency, perubahan Simple-ERP, commit, atau push pada verifikasi ini.

## Perbaikan yang diperiksa

- Orkestrasi Go tetap memakai Analyst, Planner dan Reviewer. Model dan profil bisa dipilih per peran; identitas gabungan mengikat konfigurasi sesi. Tidak ada perpindahan model diam-diam.
- Planner yang melewati konteks dipisah per target tanpa membuang requirement atau graph. Hasil bagian disimpan, terikat ke source/model/draft/feedback, dan dipakai ulang saat resume. Semua panggilan tambahan tetap mengurangi budget. Reviewer gabungan tetap diperlukan.
- Reviewer wajib memberikan kesimpulan per file untuk requirement, kontrak, scope dan kasus kegagalan/test. Referensi bukti dibatasi ke kandidat yang benar. Persetujuan model bukan bukti independen bahwa rencana benar.
- NO_ACTION mengizinkan file relevan dinilai tidak membutuhkan edit. Harus ada current/target behavior, alasan, bukti dan pemeriksaan kontrak yang sudah tersedia. Usulan edit dan langkah implementasi harus kosong. Keputusan ini tetap melewati Reviewer.
- Daftar/jumlah file dengan usulan edit mengecualikan NO_ACTION dan menggabungkan target berulang untuk file yang sama. Assessment tanpa edit tetap tersedia terpisah di web dan Markdown. Belum ada pembaruan benchmark historis berdasarkan keputusan baru ini.
- Penemuan source memakai BM25 atas snapshot yang disetujui, mendahulukan implementasi/config/test dari dokumentasi. Dokumen tetap dapat dibaca langsung atau melalui halaman berikutnya. Ini perubahan tool penemuan agen; ranking mapper deterministik tidak diubah.
- Hasil pencarian fungsi dalam file yang sudah menjadi kandidat whole-file memperkaya bukti tanpa menggandakan target.
- Catatan Reviewer disimpan/diekspor dan dibatalkan ketika draft/bukti diganti. Timeout, pembatalan dan kegagalan memori mendapat diagnosis yang berbeda. Log inferensi mencatat model/peran/durasi/token, tanpa source atau reasoning privat.
- outputs/go.mod menjadi batas modul untuk fixture Go hasil generasi. Root go test tidak lagi masuk folder output lama yang aksesnya ditolak. Semua isi outputs selain go.mod tetap diabaikan Git.

## Verifikasi

Go dijalankan berurutan dengan GOMAXPROCS=1, GOFLAGS=-p=1, GOMEMLIMIT=128MiB, GOTOOLCHAIN=local, GOPROXY=off dan GOSUMDB=off. Cache berada di workspace.

- go test ./...: lulus, termasuk pengujian ulang setelah perbaikan hitungan NO_ACTION.
- go vet ./...: lulus.
- npm run typecheck: lulus pada pemeriksaan ulang 10 November.
- npm run check:encoding: lulus pada pemeriksaan ulang 10 November.
- git diff --check: lulus pada pemeriksaan penutupan.

Tes regresi mencakup resume Planner tanpa mengulang bagian selesai, budget/persetujuan, pemisahan requirement, rujukan bukti Reviewer, invalid JSON/rujukan, keputusan NO_ACTION, pembatalan/timeout, pemilihan model per peran, deduplikasi penemuan, dan paging dokumentasi. Model tiruan memeriksa kontrak program; hasil ini tidak menilai kecerdasan model nyata.

## Hasil live sebelumnya yang tetap belum selesai

Dua uji penuh tersimpan berakhir NEEDS_REVIEW tanpa ekspor plan final valid:

- .data/model-probes/pipeline-20261109T143631/ ? 84 turn; sebelum perbaikan penemuan/konteks berikutnya.
- .data/model-probes/pipeline-20261109T145809/ ? 83 turn; perbaikan penemuan dan pemecahan Planner sudah masuk, tetapi sebelum kontrak NO_ACTION terakhir.

Turn adalah langkah orkestrasi, bukan jumlah pasti permintaan inferensi. Artefak privat tidak dipublikasikan. Konfigurasi tiga model bukan konfigurasi pemenang yang sudah terbukti.

## Langkah saat VPN tersedia

1. Gunakan saved analysis yang sama; buat sesi agen baru karena kontrak/model/profil berubah. BPMN tidak perlu diunggah ulang jika snapshot tersimpan utuh.
2. Bandingkan Analyst 9B versus 35B dengan source, query, konteks, budget, profil dan schema yang sama. Belum ada bukti langsung bahwa 9B menang melawan 35B sebagai Analyst.
3. Ulangi satu skenario inventory dengan kontrak terbaru. Periksa pemilihan file dan ketepatan rencana secara terpisah; nilai NO_ACTION juga, bukan hanya status COMPLETE.
4. Baru sesudah alur lengkap menghasilkan rencana yang masuk akal, ukur seleksi file terhadap acuan provisional. Jangan memakai urutan Markdown sebagai ranking untuk MAP/MRR.

## Landasan pilihan model

Keduanya keluarga Qwen3.5. 35B-A3B adalah MoE (35B total, sekitar 3B aktif per token); 9B adalah model dense. Model lab memakai kuantisasi berbeda, sehingga benchmark model resmi bukan pengukuran langsung GGUF lab.

Model card resmi: [Qwen3.5-35B-A3B](https://huggingface.co/Qwen/Qwen3.5-35B-A3B), [Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B). Hasil resmi mendukung 35B sebagai kandidat lebih kuat secara umum; pengujian tugas aplikasi tetap menentukan konfigurasi akhir.
