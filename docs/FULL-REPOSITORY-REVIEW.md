# Full repository review

Mode ini membaca seluruh file teks yang lolos secure ingestion untuk **satu
perubahan BPMN yang dipilih**. Hybrid memberi daftar petunjuk awal; file lain
tetap diperiksa, dan daftar temuan akhir tidak dipotong menjadi top-k.

## Memakai web

1. Restart backend dengan flag Qwen yang biasa dipakai. Tidak ada dependency baru.
2. Jalankan mapping atau buka saved analysis yang sesuai dengan source saat ini.
3. Di Workspace pilih langkah Planning. Path mengikuti Mapping; pilih perubahan BPMN.
   Laporan lama tanpa path tersimpan perlu diisi sekali pada perangkat ini.
4. Pilih **Full repository review** pada Review scope dan mulai sesi baru.
5. Perhatikan Repository coverage: eligible files, source chunks, dan planning
   batches. Sesudah selesai, unduh plan Markdown/JSON beserta cakupannya.

Pilihan **Focused review** tetap tersedia. Default UI untuk sesi baru adalah full;
API yang tidak mengirim mode tetap memakai focused demi kompatibilitas. Mode
tidak boleh diganti ketika melanjutkan sesi yang sama.

Request API tambahan memakai `"mode": "full"` bersama `repository_path` dan
`change_index` pada endpoint agent-sessions yang sudah ada. Semua path adalah
path pada mesin backend; model lab tidak membuka path laptop secara langsung.

## Cara kerja

Backend membentuk snapshot read-only dari file yang diizinkan. File dibagi
menjadi potongan maksimal 4096 byte dengan overlap sekitar 128 byte, mengikuti
batas UTF-8. File kosong juga mendapat satu potongan. Manifest divalidasi agar
tidak melewatkan awal/akhir file atau meninggalkan celah antarbagian.

File petunjuk Hybrid diproses lebih dulu, lalu file lainnya secara deterministik.
Setiap request analyst membawa seluruh isi potongan tersebut, identitas perubahan
BPMN, dan petunjuk file awal. Model mengembalikan keputusan RELEVANT, WEAK atau
REJECT dengan evidence ID yang harus valid. Tidak ada source tool tambahan pada
giliran klasifikasi ini: orchestrator memastikan semua potongan mendapat giliran.

Hasil per file digabung: RELEVANT mengungguli WEAK, lalu REJECT. Semua file yang
tidak ditolak diteruskan ke planner dan reviewer dalam batch maksimal tiga file.
Planner dapat meminta analyst mencari bukti tambahan. Batas tiga ini ukuran
prompt **per batch**, bukan batas jumlah file pada rencana akhir.

Rencana semua batch digabung menjadi canonical JSON dan Markdown. Source snapshot
fingerprint, model/prompt identity, dan cakupan disimpan. Keputusan file lengkap
tersedia pada agent context/session; receipts tiap potongan disimpan terpisah.

## Progres dan pemulihan

Sesi utama ada di `.data/agent-sessions/SESSION_ID.json`. Receipt potongan ada
di `SESSION_ID-scan-N.json`; checkpoint planning ada di `SESSION_ID-plan-N.json`.
Receipt scan menyimpan potongan source yang dikirim, keputusan, dan binding-nya.
Folder ini bersifat lokal, tidak di-commit, dan mengandung source repository.

Potongan baru dihitung selesai setelah jawaban klasifikasi lolos validasi dan
receipt tersimpan. Jawaban gagal tidak menambah coverage. Gangguan koneksi
menghasilkan PAUSED; pulihkan tunnel, isi path yang sama, dan Resume review.
Potongan dengan receipt lengkap tidak dikirim ulang. Setiap potongan memiliki
batas tiga percobaan lintas resume agar kegagalan berulang tidak berjalan selamanya.

Source, model, prompt, atau perubahan BPMN yang berbeda memerlukan sesi baru.
Check model tidak boleh memperbarui model identity ketika sebuah review berjalan.
Refresh halaman tidak memulai inference lagi. Backend dan terminal tunnel tetap
harus berjalan; penutupan backend menghentikan worker dan checkpoint dapat dipakai
untuk resume setelah backend dinyalakan kembali.

Full mode tidak mempunyai deadline job keseluruhan secara default. Batas satu
pemanggilan model dan pembatalan server tetap berlaku. Focused mode tetap memakai
batas job sebelumnya. Mode ini hanya satu worker, tidak menjalankan request model
paralel ke GPU lab.

## Arti 100%

100% berarti seluruh potongan file **yang diizinkan ingestion** telah diberikan
ke model dan mendapat keputusan valid. Ini tidak membuktikan model memahami
semuanya dengan benar. Cakupan scan dan status mutu rencana berbeda: coverage
bisa penuh tetapi plan masih NEEDS_REVIEW.

Secrets, dependency, binary, symlink, generated files, dan corpus leakage tetap
dikecualikan sesuai kebijakan ingestion. Batas file/repo dan encoding yang
didukung tetap berlaku. Jangan menyebutnya 100% dari seluruh isi folder mentah.
Source tidak dieksekusi atau diubah. Evaluasi/ground truth tidak diberikan ke model.

Audit lokal tanggal 2 November 2026 atas snapshot Simple-ERP saat pengembangan
menghasilkan 198 eligible files, 276 chunks, dan 571258 byte unik. Audit itu
melakukan **nol inference call**. Full review snapshot tersebut membutuhkan
sedikitnya 276 klasifikasi model sebelum planning. Jumlah dan durasi berikutnya
bergantung pada repo, model dan koneksi; angka ini bukan benchmark latency.

Checkpoint utama full mode dibatasi 64 MiB, focused tetap 2 MiB. Ekspor plan
dibatasi 16 MiB; hasil terlalu besar ditolak secara eksplisit, bukan dipotong.
Perencanaan lintas batch menggunakan konteks terbatas dan tidak menjamin semua
dependensi lintas file dipahami. Kualitasnya tetap perlu diuji dengan model nyata.

## Verifikasi

- Tes byte coverage, Unicode, overlap, file kosong dan immutable snapshot lulus.
- Manifest dengan celah ditolak sebelum inference.
- Tes scan terputus/resume: receipt lengkap tidak dipanggil ulang; source berubah ditolak.
- Delapan file diperiksa dan tujuh temuan direncanakan dalam tiga batch: tidak ada top-k truncation.
- Jawaban dengan evidence palsu tidak dihitung dan berhenti setelah tiga percobaan.
- API menggunakan adapter Ollama asli dengan model HTTP tiruan: full scan,
  pemulihan koneksi, penyimpanan, coverage dan unduhan lulus.
- Browser: mode full, coverage, output plan dan refresh tanpa duplicate start lulus.

Tidak ada pengujian full scan memakai Qwen lab pada pekerjaan ini. Saat VPN
tersedia, mulai sesi baru pada kasus inventory yang sama dan nilai hasil file
serta rencananya terpisah dari coverage.
