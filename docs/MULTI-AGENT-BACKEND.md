# Kerangka multi-agent backend

Status: kerangka Go dan [adapter Ollama lab](OLLAMA-LAB.md) opt-in. Tes scripted
dan sintetis live telah berjalan; Planning web dapat diaktifkan dengan --agent-model. Tidak ada dependency
framework baru. Provider planning web lama tetap disabled secara default.

## Peran dan batas

- Analyst: menilai semua kandidat, memakai search_source, read_function dan
  find_callers untuk mencari bukti tambahan dari snapshot source yang diizinkan.
- Planner: membuat rencana natural language untuk kandidat non-rejected. Jika
  bukti kurang, mengembalikan pertanyaan ke analyst melalui need_evidence.
- Reviewer: menerima hasil atau meminta revisi ke analyst/planner beserta alasan.

Kontrak dan instruksi role berada di internal/planning/agents.go. AgentEngine
menerima AgentModel, SourceTools dan AgentSessionStore sebagai port. Adapter
agenttools menggunakan secure ingestion dan source index yang sudah ada.
Temuan source baru masuk sebagai VERIFY, tanpa retrieval score buatan; ranking
laporan semula tidak diubah. Bukti source bukan instruksi untuk model.

Default: 18 turn model, 8 tool calls, 2 revisi/handoff dan maksimal 3 kandidat
tambahan per perubahan. Satu turn dibatasi 90 detik dan provider harus mematuhi
context cancellation. Tidak ada shell, source editing atau coding agent.
Read function mengembalikan potongan terbatas, bukan seluruh isi function/repo.
Kegagalan/ambiguitas tidak boleh dipaksakan menjadi keputusan pasti.

## Checkpoint dan ekspor

filestore.AgentSessionStore menyimpan JSON sesi dengan checksum. Identitas sesi
terikat pada konteks awal, model/prompt, versi protokol, snapshot source dan
anggaran. Model terputus menghasilkan PAUSED; Run dengan ID yang sama melanjutkan
dengan budget yang sudah terpakai. Perubahan binding mengharuskan sesi baru.
Save dilakukan sebelum panggilan model dan tool. Checksum mendeteksi kerusakan,
bukan autentikasi terhadap pengguna lokal yang sengaja mengedit file.

Session berstatus COMPLETE/NEEDS_REVIEW dapat diekspor melalui ChangePlan dan
renderer JSON/Markdown existing. COMPLETE menyatakan alur review selesai,
bukan bukti kebenaran implementasi atau akurasi 100%. Konteks incomplete,
truncated atau warning analisis tetap memaksa NEEDS_REVIEW.

Worker agentjobs menjalankan satu sesi secara serial melalui API async/polling.
Planning web menampilkan aktivitas dan dapat memuat kembali checkpoint; belum
ada antrean atau recovery otomatis yang memanggil model saat server startup.
Checkpoint memakai temporary file lalu rename; ini penyimpanan lokal single
writer, bukan distributed store atau perlindungan concurrent filesystem swaps.

## Bukti dan langkah integrasi

Tes integrasi sintetis menjalankan source tool -> checkpoint -> gangguan model
-> load ulang disk -> analyst -> planner -> review/revisi -> canonical plan.
Source fixture tetap identik. Tes lain mencakup file/evidence karangan, invalid
tool, binding drift, batas revisi, konteks partial, path/symlink dan korupsi sesi.

Adapter AgentModel kini memakai schema-constrained JSON actions melalui Ollama,
bukan native tools API. Berikutnya: perbaiki kelengkapan rencana, uji requirement
ERP terbatas, baru composition-root/HTTP wiring dan UI. Framework Eino tetap
opsional; tidak diperlukan untuk kontrak yang sudah berjalan.

Research freeze sebelumnya tetap historis. Kode baru ini mengubah policy hash;
jangan menimpa freeze lama untuk menghilangkan drift.

Verifikasi: `go test -p 1 ./cmd/... ./internal/...` dan
`go vet -p 1 ./cmd/... ./internal/...` lulus. Pemindaian `go test ./...`
terhalang direktori temporary lama di outputs yang tidak dapat diakses Windows;
seluruh package aplikasi diperiksa lewat dua root eksplisit di atas.
