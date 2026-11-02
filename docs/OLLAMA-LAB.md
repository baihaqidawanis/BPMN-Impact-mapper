# Qwen lab: adapter dan runner multi-agent

Status: adapter Ollama, CLI dan Planning web opt-in tersedia. Hybrid tetap
menghasilkan ranking awal; tombol Qwen menjalankan review tambahan secara terpisah.
Tidak ada dependency baru, download model, perubahan server lab atau source ERP.

## Akses laptop

## Jalankan dari web

Restart backend dengan satu baris ini dari root Mapper:

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-impact-mapper-server --agent-model qwen3.5:9b-q8_0 --agent-url http://127.0.0.1:11435
```

Frontend: `npm run dev` dari folder web. VPN dan SSH tunnel harus tetap aktif.
Jalankan mapping Hybrid seperti biasa atau buka Saved analyses dari source yang
masih cocok. Di Workspace buka langkah Planning; path mengikuti Mapping. Pilih satu perubahan
BPMN, lalu Run Qwen review. Rencana, review issues dan keputusan file ditampilkan
terpisah dari ranking awal. Download Markdown/JSON tersedia setelah plan tersimpan.

Web memakai 32K context, 3 kandidat awal + 1 source connection per perubahan,
2 evidence per kandidat dan excerpt 120 rune. Satu worker menerima satu sesi;
start mengembalikan 202, browser memantau progres melalui GET. Refresh hanya
memuat kembali sesi, bukan mengulang inference. Repo path perlu diisi ulang
untuk resume. Plan/sesi disimpan di .data/plans dan .data/agent-sessions.
Tanpa --agent-model, fitur dinonaktifkan dan mapping biasa tetap berjalan.
Snapshot offline lama dipisahkan dalam bagian Offline audit snapshots.

Verifikasi web: dua tes browser lulus (mock model untuk alur UI/refresh, serta
regresi demo/snapshot offline). API dengan Qwen lab nyata dan fixture sintetis
juga lulus: outputs/lab-web-smoke-20261030T122959, 11 turn, 2 tools, 2 revisi,
40,85 detik, final NEEDS_REVIEW. Tidak mengirim source ERP. Windows checkpoint
replacement kini retry permission error secara terbatas tanpa menghapus file
lama. Source/ranking tidak diubah. Kualitas rencana model tetap perlu ditinjau.

## Tunnel

VPN tersambung, buka PowerShell laptop dan biarkan tunnel ini tetap berjalan:

```powershell
ssh -N -o ExitOnForwardFailure=yes -L 127.0.0.1:11435:127.0.0.1:11434 mci@10.28.83.127
```

Di terminal kedua:

```powershell
curl.exe -sS --max-time 10 http://127.0.0.1:11435/api/version
```

Backend memakai numeric loopback HTTP. Tidak perlu menyalin SSH password ke
aplikasi. Model dipilih eksplisit: qwen3.5:9b-q8_0. Adapter memeriksa model digest,
capabilities dan versi Ollama; cloud aliases/remote models ditolak. Tidak ada
automatic pull atau fallback ke layanan cloud. Redirect juga tidak diikuti.

## Jalankan CLI setelah koneksi kembali

Gunakan laporan baru dari source yang sama. Ganti reportPath sesuai file yang
diunduh. Export context tidak menjalankan model atau membaca ulang repository.

```powershell
cd C:\BPMN-Impact-Mapper
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'
$reportPath='.\impact-report.json'
go run ./cmd/bpmn-impact-mapper planning-context --report $reportPath --output outputs/planning-context-lab.json --candidates 5 --related 1 --evidence 3 --excerpt 160
go run ./cmd/bpmn-impact-agents --repo C:\Simple-ERP --context outputs/planning-context-lab.json --session erp-trial-01 --model qwen3.5:9b-q8_0 --data-dir .data
```

Output export context harus file baru. Kandidat dibatasi untuk mengendalikan
panjang konteks, bukan dipilih berdasarkan kunci jawaban. Lebih banyak perubahan
dalam satu analisis dapat tetap melewati batas prompt; runner menolak input
terlalu besar tanpa mengirim source ke model atau memotong diam-diam.

Sesi tersimpan di .data/agent-sessions. Run dengan ID yang sama melanjutkan sesi
PAUSED jika konteks, model/prompt, tools dan anggaran cocok. Sesi yang sudah final
tidak memanggil model lagi. Perubahan kebijakan atau source membutuhkan sesi baru.
Data directory harus di luar repository yang dianalisis. Worker CLI satu sesi
harus dijalankan serial; belum ada concurrent-worker locking.

Rencana final disimpan melalui canonical JSON/Markdown existing di
.data/agent-plans/PLAN_ID. COMPLETE berarti proses review selesai, bukan bukti
implementasi benar. NEEDS_REVIEW tetap membutuhkan pemeriksaan. Aksi VERIFY
tidak otomatis dipromosikan menjadi MODIFY oleh model.

## Cara model memakai tool

Ollama menerima JSON-schema output untuk tepat satu aksi per turn. Ini protokol
aksi JSON yang dijalankan AgentEngine, bukan pemakaian native tools API Ollama.
Analyst boleh memilih search_source/read_function/find_callers, planner boleh
meminta bukti tambahan, reviewer boleh meminta revisi. Tool hanya membaca index
snapshot yang lolos ingestion. Lookup gagal/duplikat memberi observasi terbatas;
budget tetap terpakai. Tool tidak membuka shell atau menulis source.

Default 16K context, maksimal output 4096 token, temperature 0, thinking false,
keep_alive 2m. Prompt dan response dibatasi. Identitas sesi mengikat model digest,
runtime version, role schemas dan konfigurasi; keluaran model tidak dijanjikan
byte-identical walaupun temperature nol.

## Verifikasi dan keterbatasan live

Tes httptest memeriksa struktur request, model digest drift, cloud aliases,
redirect, output terpotong, field asing dan trailing JSON. Tes scripted mencakup
source discovery, handoff/revisi dan resume dari checkpoint disk.

Tes lab hanya mengirim fixture Go sintetis satu function, bukan Simple-ERP atau
label evaluasi. Percobaan awal memperlihatkan aksi ganda, argumen tool kurang,
lookup berulang dan function name hilang. Schema diperketat, budget diperlihatkan
ke model dan lookup duplikat tidak diulang. Percobaan berikutnya mencapai
analyst/planner tetapi output reviewer belum lolos; tunnel juga sempat terputus.
Setelah tunnel dipulihkan, perbaikan mencakup cardinality kandidat, enum evidence
ID persis, target revisi, batas array/string dan instruksi role yang konsisten.
Package Go tidak lagi disajikan sebagai function tambahan oleh source tools.
Artefak gagal dipertahankan, tidak ditimpa/dilaporkan sebagai lulus.

Run `outputs/lab-agent-smoke-20261030T115916` menyelesaikan semua peran dalam
37,07 detik: 10 turn, 2 tool, 2 revisi, final NEEDS_REVIEW. Run setelah instruksi
diselaraskan di `outputs/lab-agent-smoke-20261030T120107` menyelesaikan ketiga
peran dalam 8,26 detik: 3 turn, final COMPLETE. CLI nyata juga menghasilkan
plan ID yang sama di `outputs/lab-cli-smoke-20261030`; pemanggilan ulang sesi
tetap 3 turn, tidak mengulang giliran model.

Ini bukti protokol/orchestration, bukan akurasi atau mutu rencana. Draft terakhir
masih berfokus pada inspeksi dan verifikasi existing, belum rencana perubahan
yang cukup rinci. Persetujuan reviewer model tidak dianggap penilaian independen.
Belum ada percobaan source ERP lewat model atau wiring API/web.

Tes live hanya opt-in:

```powershell
$env:BPMN_IMPACT_LAB_SMOKE='1'; go test -p 1 ./internal/llm -run '^TestOllamaAgentLiveSynthetic$' -count=1 -v
```

Jangan aktifkan flag tersebut pada test rutin. Setelah tiga role lolos pada
fixture, baru uji requirement ERP terbatas, lalu wiring API/web. Pembandingan
Hybrid vs Hybrid + Multi-Agent Qwen harus memisahkan ranking awal, rekomendasi
file sesudah review dan kualitas rencana; belum ada angka akurasi baru.
