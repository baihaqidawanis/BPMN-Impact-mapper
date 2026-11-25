# Brief UI/UX untuk Gemini di Antigravity

Rapikan UI/UX BPMN Impact Mapper di `C:\BPMN-Impact-Mapper`. Ini alat kerja untuk membandingkan BPMN, membaca bukti dampak di source code, dan meninjau rencana perubahan. Prioritaskan alur kerja, hierarki informasi, dan keterbacaan. Haqi terbuka untuk perubahan layout yang signifikan selama identitas dan fitur produk tetap terjaga.

## Orientasi kedua repo

1. Baca `collab/CLAUDE-GPT.md` untuk memahami arsitektur Mapper, alur analisis, peran agent, dan hubungan dengan Simple-ERP. Baca `collab/STATUS.md` dan entri terakhir `collab/WORKLOG.md` untuk mengetahui progres serta pekerjaan berikutnya. Gunakan `README.md` Mapper untuk menelusuri modul dan dokumentasi teknis.
2. Baca `C:\Simple-ERP\README.md` dan `C:\Simple-ERP\docs\PROCESS-MAPPING.md` secara read-only. Telusuri BPMN dan modul source yang relevan untuk memahami contoh yang muncul di Mapping dan Planning.
3. Untuk tugas UI/UX sekarang, gunakan `DESIGN.md`, bagian **Handoff UI Gemini** di `collab/STATUS.md`, dan implementasi di `web/src`. Periksa perubahan Git yang sudah ada. Dahulukan arahan terbaru Haqi dan verifikasi catatan lama terhadap kode saat ini.

Ketika Haqi meminta Gemini melanjutkan tugas Codex, baca ulang catatan dan diff terbaru, lalu lanjutkan pekerjaan yang Haqi pilih dari titik terakhir. Jika pekerjaan terhenti, catat file yang disentuh, keputusan penting, hasil verifikasi, dan langkah konkret berikutnya di WORKLOG. Simpan status aktif di STATUS agar agent berikutnya bisa melanjutkan dari checkpoint yang sama.

## Arah desain

- Pertahankan sidebar charcoal full-height, rata kiri, beserta perilaku collapse dan mobile. Gunakan workspace terang, aksen biru untuk aksi dan pilihan, serta tipografi sistem yang mudah dibaca.
- Fokus pada layout dan penempatan informasi. Gunakan baris untuk daftar kandidat, panel untuk area kerja yang benar-benar terpisah, dan bukti source yang mudah dipindai. Hindari hero pemasaran, metrik dekoratif, kartu berlapis, dan animasi yang mengganggu pembacaan.
- Mulai dari alur **Mapping setup → Change Explorer → Planning**. Rapikan layar pendukung bila perlu untuk konsistensi. Pertahankan dua diagram Before/After dalam panel yang jelas dan beri Canvas ruang kerja.
- Di Mapping, buat upload dan kesiapan input mudah dipahami. Begitu hasil tampil, beri ruang untuk kandidat, jalur file, dan bukti. Pertahankan navigasi perubahan, Show all changes, By file, dan ekspor.
- Di Planning, buat mode review dan aksi sesi mudah ditemukan. Bedakan **waktu aktif**, **estimasi menuju final review**, **progres batch**, dan **cakupan kandidat**. Pakai angka dari data sesi. Biarkan status Paused, Needs review, error, dan session lama tanpa catatan waktu tetap jelas.
- Pastikan desktop, tablet, dan ponsel nyaman digunakan. Cek navigasi keyboard, fokus, kontras, label kontrol, overflow code/table, dan reduced motion.

## Batas perubahan

- Ubah presentasi di `web/src`. Boleh menyusun ulang komponen presentasi, CSS, dan microcopy. Gunakan komponen serta dependensi yang sudah ada. Perbarui tes UI terkait bila selector perlu mengikuti perubahan tampilan.
- Pertahankan perilaku mapping, pengaturan mode dan model, save/restore dan deep link, polling, pause/resume, aturan stopwatch/estimasi, serta ekspor. Pakai handler dan kontrak API yang sudah ada.
- `C:\Simple-ERP` adalah studi kasus read-only. Perlakukan backend Go, prompt/model, evaluasi, BPMN eksperimen, dan `.data` sebagai di luar scope. Gunakan fixture atau mock untuk preview. Biarkan proses backend dan sesi agent aktif tetap berjalan.
- Periksa diff sebelum edit dan pertahankan perubahan kerja Haqi atau agent lain. Jangan commit atau push tanpa permintaan.

## Verifikasi dan handoff

Jalankan `npm.cmd --prefix web run typecheck`, `npm.cmd --prefix web run check:encoding`, dan `git diff --check`. Jika ada browser atau Playwright dengan fixture yang tersedia, cek alur utama di desktop dan mobile, termasuk Planning saat running, paused, dan selesai. Laporkan apa yang benar-benar dicek dan bagian yang belum diverifikasi secara visual. Ringkas perubahan dan tambahkan satu entri di `collab/WORKLOG.md`.
