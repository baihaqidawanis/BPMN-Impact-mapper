# Kesiapan integrasi agent — 1 November 2026

Status: kontrak dan alur integrasi offline lulus. Kualitas output model lab belum
diuji ulang dengan kontrak baru. Tidak ada inference ke lab pada pekerjaan ini.

## Yang sudah disiapkan

| Bagian | Perilaku yang diperiksa |
|---|---|
| Analyst | Instruksi hanya untuk analyst; source lookup tetap tersedia di luar kandidat awal. |
| Revisi | Riwayat source dan lokasi halaman berikutnya bertahan; file yang sudah dibaca tidak dipaksa dibaca ulang. |
| Planner | Draft sebelumnya tersedia saat revisi planner. Setiap file wajib memuat usulan perubahan dan acceptance checks. |
| Reviewer | Memeriksa rencana dan bukti, dapat mengembalikan ke analyst atau planner. Tidak menuntut perubahan kode nyata. |
| Output salah | Satu kesempatan perbaikan kontrak per sesi, tetap memakai budget turn. Kegagalan berulang berhenti dengan status yang jelas. |
| Bukti palsu | Target atau evidence ID yang tidak disediakan ditolak; tidak otomatis dibetulkan menjadi seolah valid. |
| Penyimpanan | Checkpoint dan plan tersimpan; pembacaan ulang hasil tidak mengulang inference. |
| UI dan Markdown | Menampilkan proposed changes, implementation steps, acceptance checks, bukti dan review issues. |

Kontrak baru memakai `agent-loop-v3` dan identitas prompt
`ollama-agent-json-v3-*`. Sesi lama tetap menjadi arsip; mulai sesi baru setelah
restart backend. Mapping deterministik dan ranking awal tidak berubah.

## Struktur rencana

Setiap file memiliki:

- `summary`: tujuan perubahan pada file tersebut;
- `proposed_changes`: perilaku yang diusulkan, termasuk batas tindakan yang berubah;
- `steps`: urutan pekerjaan developer;
- `acceptance_checks`: kondisi yang harus berhasil/gagal setelah implementasi;
- `evidence_ids`: bukti source yang mendukung pemilihan file.

`VERIFY` tetap boleh berisi usulan perubahan. Status tersebut bukan larangan
merencanakan implementasi, dan bukan pernyataan bahwa perubahan sudah dilakukan.
Penambahan field ini kompatibel dengan laporan offline lama; agent baru wajib
mengisinya. Validator memeriksa struktur dan bukti, bukan menjamin kebenaran makna.

## Bukti pengujian

`TestAgentOfflineContractPipeline` memakai adapter Ollama asli dengan server HTTP
tiruan lokal, API async, source tools, penyimpanan disk, dan unduhan Markdown.
Tujuh kasus diperiksa: alur sukses dengan revisi, file palsu, evidence palsu,
rencana tidak lengkap, pemulihan satu JSON rusak, JSON rusak berulang, dan
koneksi gagal. Source sintetis tetap tidak berubah. Sukses model tiruan bukan
pengukuran akurasi Qwen.

Unit test tambahan memeriksa pemisahan instruksi role, field wajib dalam schema,
riwayat source saat revisi, batas perbaikan, dan pemadatan riwayat tanpa mutasi.
Tes browser memeriksa output baru, link unduhan, dan refresh tanpa memulai ulang.

Perintah verifikasi tanpa lab:

```powershell
.\scripts\verify-agents.ps1
```

Tambahkan `-Browser` untuk menjalankan tes browser lokal. Go, npm, dependency web
yang sudah terpasang, dan Chrome diperlukan; script tidak mengunduh dependency.
Hasil terbaru disimpan di `outputs/agent-readiness/result.json` dan tidak di-commit.
Laporan script saat pekerjaan ini: seluruh pemeriksaan lulus. Tes browser Qwen
juga dijalankan terpisah dan lulus. Tes seluruh package aplikasi dan vet lulus:

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
```

## Saat lab tersambung

1. Jalankan backend dengan model lokal yang dipilih, lalu Check model.
2. Mulai sesi baru pada skenario inventory yang sama, tanpa mengubah BPMN lagi.
3. Periksa apakah usulan spesifik, menggunakan input yang sudah ada, menjaga
   tindakan di luar scope, dan mempunyai acceptance checks yang sesuai.
4. Catat mutu mapping dan rencana terpisah dari keberhasilan API/JSON.

Yang masih memerlukan model nyata: kepatuhan Qwen terhadap instruksi, kebenaran
interpretasi source, kelengkapan rencana, latency dan pemakaian konteks/GPU.
Hasil tidak boleh dipaksa `COMPLETE` hanya agar terlihat berhasil. Tujuan kesiapan
offline adalah menghilangkan masalah integrasi yang bisa diuji sekarang; bukan
menjanjikan tidak akan pernah ada kekeliruan model.
