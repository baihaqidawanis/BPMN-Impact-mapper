# Pegangan bersama Claude dan GPT

Terakhir diperiksa: 17 November 2026. Dokumen ini adalah konteks kerja, bukan bukti bahwa semua fitur sudah benar. Cocokkan dengan kode, git diff, checkpoint dan output tes saat melanjutkan.

## Mulai dari sini

1. Baca dokumen ini, [STATUS.md](STATUS.md), lalu entri terbaru [WORKLOG.md](WORKLOG.md).
2. Periksa `git status --short` dan instruksi repo/subfolder. Banyak perubahan terakhir belum di-commit. Jangan reset, overwrite atau menganggap semua perubahan milikmu.
3. Baca kode sesuai peta di bawah. Scan source dan konfigurasi aman secara bertahap; jangan memasukkan seluruh repo ke satu prompt. Lewati dependency, build, cache, file biner, secrets dan data sesi besar sampai ada kebutuhan spesifik.
4. Sebelum edit, tulis fokus dan status pekerjaan pada STATUS. Satu editor aktif per tugas/file; pergantian GPT/Claude dilakukan Haqi, bukan otomatis.
5. Setelah milestone dan sebelum handoff, perbarui STATUS dan tambahkan entri WORKLOG: perubahan, tes aktual, keterbatasan, langkah berikutnya. Jangan menghapus riwayat entri agent lain. Baca ulang perubahan file sebelum menyimpan.

Instruksi langsung Haqi dan kebijakan lingkungan tetap berlaku. Catatan ini tidak memberi izin baru untuk deployment, source ERP, cloud inference, penghapusan data atau mengakses credential.

## Haqi dan arah TA

- Panggil pengguna **Haqi**. Bahasa Indonesia santai, ringkas, jelas; jelaskan function/file dengan istilah wajar. Hindari klaim sempurna/100% dan skor manual tanpa rubrik.
- Target yang disampaikan Haqi: sisa sekitar satu bulan. Fokus sistem yang selesai dan evaluasi yang dapat dipertanggungjawabkan. Tanggal pengumpulan/persetujuan dosen harus dikonfirmasi jika dibutuhkan.
- Tujuan: alat pihak ketiga membaca repository software serta BPMN before/after, mengidentifikasi calon dampak, memeriksa source dengan agent, lalu menghasilkan rencana teknis developer.
- **BPMN adalah blueprint. Software target tidak harus membaca BPMN saat runtime.** Simple-ERP menjadi studi kasus, bukan aturan hardcode untuk semua software.
- Output: impact-report JSON/Markdown dan change-plan JSON/Markdown. Tidak mengubah source target, menjalankan kode target, menerapkan migration atau membuktikan program hasil rencana berhasil.
- Tiga role: Analyst, Planner, Reviewer. Orkestrasi sendiri di Go; LangGraph/LangChain/Hermes, Neo4j, fine-tuning dan agent coding bukan kebutuhan aktif.
- Saran teman Abimanyu pernah mencakup unified graph/Neo4j dan multi-agent per layer. Bedakan dengan metode Haqi; jangan mengklaim mapper memakai knowledge graph lintas BPMN-source.
- Kelas Sistem Enterprise: kelompok Dea mengerjakan BM25, kelompok lain Semantic Retrieval sebagai proyek terpisah; mereka tidak diwajibkan agentic AI. Implementasi/eksperimen mereka belum otomatis hasil TA tervalidasi; atribusi kontribusi diperlukan.

## Alur dan empat metode

```text
Before/after BPMN -> XML parser -> graph model -> structural diff + konteks
Repo read-only -> inventory -> source index ----------------------------+
                                                                      v
               BM25 / Semantic / Fusion / Context-aware Hybrid
                             -> impact report + evidence
                             -> Analyst -> Planner -> Reviewer
                             -> change-plan.json -> change-plan.md
```

- Parser dan diff ditulis dalam Go. Jenis gateway dibaca dari XML, bukan menebak XOR/AND dari jumlah panah. ID adalah petunjuk identitas struktural; label, edge, tipe dan scope turut diperiksa. Batas detail ada di matriks kemampuan.
- **BM25:** baseline kata dari elemen berubah. Cari pada dokumen source yang terindeks.
- **Semantic:** embedding query perubahan dan potongan source menggunakan runtime Python lokal `sentence-transformers`, model `all-MiniLM-L6-v2`; cosine similarity. Bukan membandingkan embedding dua XML BPMN untuk menemukan diff. Cache/identitas model direkam; vector database tidak wajib.
- **Fusion:** ranking BM25 dan Semantic digabung dengan RRF, k=60.
- **Hybrid produksi:** BM25 diperkaya konteks BPMN, kemudian digabung Semantic melalui RRF. Konteks dapat mencakup before/after, predecessor/successor, lane/actor, flow/branch/kondisi dan informasi requirement yang tersedia. Aturan query bergantung jenis perubahan; jangan mengklaim selalu semua tetangga masuk setiap query.
- **Bobot task ID 45% dan enam sinyal adalah formula historis, bukan produksi sekarang.** ID, path, nama function dan code clues adalah bukti penjelas; tidak ada penalti ID baru tidak ditemukan. Kandidat retrieval tetap VERIFY, bukan kepastian perlu diedit.
- RRF menggabungkan posisi, bukan menjumlahkan skor mentah BM25/cosine. Nilai retrieval bukan probabilitas atau akurasi. Akurasi file diukur memakai acuan evaluasi.
- Ada graf BPMN dalam memori dan relasi source statis terbatas. Tidak ada Neo4j/unified knowledge graph pada metode utama.
- Agent mengambil evidence melalui tools terkontrol: search_source, inspect_source, read_source, read_function, find_callers. Search hit bukan bukti sudah membaca seluruh file.

## Peta repository

### Mapper: C:\BPMN-Impact-Mapper

Go modular backend, Next.js/React frontend, bpmn-js canvas. Periksa go.mod/package.json untuk versi terbaru.

| Area | Lokasi | Tanggung jawab |
|---|---|---|
| Domain/ports | internal/domain, mapping, reportstore, planstore | Kontrak perubahan, kandidat dan penyimpanan |
| Analysis | internal/app, bpmn, diff, ingest, index | Inventaris, parser, diff, index |
| Retrieval | internal/retrieval, semantic, fusion, wiring | Ranking dan komposisi metode |
| Planning core | internal/planning | Validasi, agent loop, focused/full, budget, render |
| Job service | internal/agentjobs | Async, Stop/Resume, model config, source snapshot |
| Adapters | internal/llm, agenttools, filestore | API model, source tools read-only, persistence |
| HTTP/composition | internal/api, cmd/bpmn-impact-mapper-server | Endpoints dan dependency wiring |
| Web | web/src/app, components, features, lib | Mapping, Canvas, Planning, Saved analyses, Research |
| Research | internal/evaluation, benchmark; experiments/; scripts/ | Evaluasi terpisah dari inference |

Core tidak bergantung langsung pada transport Ollama/filesystem; adapter menunjuk inward. Tes arsitektur tersedia. Ini arsitektur pragmatis; bukan klaim semua facade sudah pure ports-and-adapters. Go punya AST/symbol/relasi terbatas; TS/JS dan bahasa lain umumnya text-only. Jangan mengklaim dukungan AST universal.

- `.data/`: analisis, snapshot BPMN, agent sessions, plans, caches. Privat dan harus dipertahankan untuk resume. Bukan folder sampah.
- `outputs/`: hasil eksperimen/manual historis, umumnya di-ignore. Hasil root pernah dipindahkan ke outputs/manual-inputs; cari dulu sebelum menyatakan hilang.
- `Planinng - Answer/`: ejaan folder aktual untuk ekspor manual terbaru Haqi. Jangan otomatis dijadikan input model atau di-commit.
- `experiments/history/`: snapshot lama untuk reproduksi, bukan panduan input aktif.
- `OpenVPN.log` mengandung identifier/token; sudah di-ignore. Jangan menyalin/menampilkan isinya mentah, memasukkannya ke handoff, mengirimnya ke model atau Git.

### Studi kasus: C:\Simple-ERP

**Read-only pada pekerjaan mapper.** Jangan memperbaiki ERP untuk membuat hasil mapper tampak benar. Permintaan eksplisit mengubah BPMN tertentu tidak memberi izin mengubah source/migration/data ERP.

- Go backend modular monolith: `backend/internal/{auth,masterdata,purchasing,inventory,sales,finance,reporting,platform}`. Biasanya domain/application/adapter dan transport per modul.
- Frontend Next.js: `frontend/features/...`. API kontrak: api/openapi.yaml. Migration: backend/migrations. Tes integrasi: backend/test/integration.
- Role aktual: STAFF, SUPERVISOR, SUPER_ADMIN. Proses meliputi purchasing/P2P, sales/O2C, inventory, master data/config dan finance. Scope Haqi yang diprioritaskan: P2P, Inventory, Master Data/Configuration; Abimanyu mengerjakan bagian lain.
- BPMN kanonis: `docs/bpm-haqi/roles/{staff,supervisor,super-admin}/`. Staff: p2p, inventory, master-data dan level-0. Supervisor/Super Admin memakai master-data-konfigurasi. Jangan menebak semua path seragam.
- `docs/bpmn/` adalah baseline awal yang berbeda dari bpm-haqi; pilih pasangan eksperimen yang cocok.
- L0 konteks, L1 modul/proses, L2 alur lebih luas, L3 detail aktivitas. Editor/mapping utama memakai satu pasangan. Library/hubungan antarlevel bukan bukti perubahan otomatis semua level atau dukungan bulk impact end-to-end.
- Mapping before: `experiments/simple-erp/baseline-traceability/` pada mapper, termasuk adj-submit-current.md/json dan staff-p2p-l3.md/json. ERP juga punya docs/PROCESS-MAPPING.md; dokumentasi bisa historis.
- Haqi memvalidasi Staff lebih dulu; audit statis dan render diagram tidak membuktikan semua role/modul 1:1 runtime. Verifikasi scope spesifik sebelum memberi klaim kesesuaian.
- Konvensi dosen pada baseline penelitian: satu start/end per scope; jangan gateway langsung ke gateway/end, gunakan aktivitas yang bermakna. Ini policy diagram studi kasus, bukan hukum universal BPMN. Jangan mengarang kegiatan bisnis hanya demi memenuhi warna/konvensi. SendTask respons bukan bukti message broker/listener.
- ERP dapat memakai Docker Compose dengan port 3000/8080, bentrok dengan mapper. Analisis source tidak memerlukan ERP dijalankan. Jangan menyalakan ERP/migration atau menulis DB hanya untuk membaca source.

### POC lama: C:\POC TA

Bukan baseline ERP utama. Pernah memakai konfigurasi JSON dan eksekusi graf BPMN. Invisible behavior/pola yang didokumentasikan: Skip, Redo, XOR-AND-Stack dan Non-free choice. Prime/Non-Prime adalah label atomik/komposit pada POC; SWITCH belum ada padanan. Baca [batas invisible task](../docs/INVISIBLE-TASK-SCOPE.md). Perubahan aturan yang tergambar dapat dianalisis; penemuan invisible transition dari event log belum diimplementasikan.

## Input percobaan aktif

- Repo path `C:\Simple-ERP`, nama `Simple-ERP`. Revision kosong boleh untuk working tree lokal. Isi commit di form tidak otomatis checkout; jangan menganggap source sama tanpa verifikasi snapshot/hash.
- Before: `C:\Simple-ERP\docs\bpm-haqi\roles\staff\inventory\level-3\role-staff-l3-adj-submit.bpmn`.
- After: `C:\BPMN-Impact-Mapper\experiments\simple-erp\current-staff-v2\add-task\after.bpmn`.
- Requirement: alasan tidak kosong sebelum submit stock adjustment. Primary ADD_TASK / ReasonReview. Dua sequence-flow detail juga ada pada analisis tersimpan.
- Paket current-staff-v2 memiliki add-task, add-xor, add-and, combined dan unchanged-control. Baca README/manifest/reference tiap kasus. Empat bentuk positif bukan otomatis empat domain kebutuhan independen.
- Saved analysis: `89a7e1615e6e85aa107dfe9e04f6dbfd`.
- URL: http://localhost:3000/?analysis=89a7e1615e6e85aa107dfe9e04f6dbfd
- Product Hybrid, top candidates 10. Planning Focused memproses semua perubahan. Full review memindai eligible chunks; tidak berarti secrets/dependency ikut atau seluruh repo masuk satu prompt.
- Jika BPMN/source utuh, saved analysis dapat digunakan tanpa upload ulang. Model/prompt/source policy berubah: **Start new AI review**, bukan Resume checkpoint lama.

## Fakta source skenario alasan submit

Verifikasi ulang jika ERP berubah:

1. frontend/features/inventory/inventory-adjustment-panel.tsx: state reason sudah ada; actionReason hanya mengirim reason untuk reject/cancel/reverse, submit mengirim string kosong.
2. backend/internal/inventory/adapter/postgres/store.go: **Store.TransitionAdjustment**, cabang submit memeriksa DRAFT/version lalu menjadi SUBMITTED; belum ada guard alasan wajib. UPDATE sudah menyimpan reason. Jangan salah ke ReverseAdjustment, menambah field/signature atau migration tanpa kebutuhan terbukti.
3. backend/test/integration/operations_test.go: beberapa success path submit memakai alasan kosong; aturan baru memerlukan input success yang sesuai serta negative test tanpa alasan dan pemeriksaan state/version tidak berubah.

Ketiga file tersebut adalah acuan sementara source-derived untuk satu ADD_TASK, bukan ground truth lengkap independen. Acuan disimpan di experiments/simple-erp/current-staff-v2/add-task/reference.json. Baca saat evaluasi, **jangan berikan file/daftar jawaban acuan ke agent aplikasi atau retrieval**. Folder collab juga hanya konteks pengembangan; exclude secara eksplisit jika mapper suatu saat menganalisis repository dirinya sendiri.

## Cara menjalankan (PowerShell laptop)

Tunnel, terminal tersendiri; menunggu tanpa output setelah password adalah normal:

```powershell
ssh -N -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -L 127.0.0.1:11435:127.0.0.1:11434 mci@10.28.83.127
```

VPN kampus diperlukan untuk menjangkau lab dari jaringan yang tidak memiliki rute. Jangan menyimpan password. Tool SSH pada lingkungan Codex sebelumnya berupa stub dan tidak benar-benar mengakses lab; jangan mengakali pembatasannya. Haqi menjalankan SSH sendiri. Koneksi loopback HTTP boleh dicek jika tunnel aktif.

```powershell
curl.exe -sS --max-time 5 http://127.0.0.1:11435/api/version
```

Backend: stop BE lama sebelum memuat kode baru, setelah review aktif selesai/dihentikan:

```powershell
cd C:\BPMN-Impact-Mapper; $env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run .\cmd\bpmn-impact-mapper-server --agent-model qwen3.5:35b-a3b-q4_K_M --agent-context 32768 --agent-output 6144 --agent-profile coding --agent-review-json
```

```powershell
cd C:\BPMN-Impact-Mapper\web; npm run dev
```

Model settings: ketiga role Qwen3.5 35B-A3B Q4, Thinking Off, Coding, context 32768/output 6144. Apply lalu Check model hanya metadata, bukan uji inference. Setting web bertahan selama proses BE hidup; restart kembali ke CLI defaults. Settings terkunci saat job aktif. Model endpoint server-owned loopback, tidak boleh diganti browser atau diam-diam ke cloud.

## Model lab dan pilihan aktif

Daftar lama bukan jaminan kondisi runtime terbaru; cek metadata tanpa inference dahulu.

- Dipilih sekarang: qwen3.5:35b-a3b-q4_K_M untuk ketiga role. Keputusan operasional, bukan bukti model terbaik.
- Alternatif yang sudah terdaftar: qwen3.5:9b-q8_0; glm-4.7-flash:latest; bpmn-qwen-coder:30b-q4_k_m. Coder ini non-thinking. Ada model lebih kecil, embedding dan alias VLM/OCR; tidak semua cocok untuk planning.
- Qwen3.8-27B-Q4_K_M.gguf sekitar 18 GB ada di ~/models/qwen3.8-27b, tetapi impor bpmn-qwen38:27b-q4_k_m gagal **no space left on device**. Jangan mengklaim sudah terdaftar atau rusak. Folder GGUF bukan alias Ollama yang siap dipanggil.
- Lab A30 24 GB, RAM sekitar 31 GiB, Ollama 0.33.1 dari output Haqi. Model internal blobs sekitar 123 GB; disk pernah 96% penuh. Semua model dipakai riset lain. Admin Mas Kurnia menyarankan memakai model yang sudah ada. **Tidak menambah model, menghapus blobs, update/restart service bersama, atau mengubah disk lab tanpa koordinasi.**
- Alias :cloud bukan inference di GPU lab. Keberadaan alias tidak membuktikan akses/quota. Source belum diizinkan dikirim ke layanan cloud.
- GGUF dapat dilayani langsung lewat llama.cpp, tetapi aplikasi memakai adapter Ollama; jalur alternatif belum disiapkan/diuji end-to-end. Bukan tugas aktif sekarang.
- Jangan mengganti model/config saat review berjalan. Jangan menambah Thinking untuk semua role sekaligus sebagai tebakan perbaikan.

## Evaluasi dan rencana developer

- Precision = file acuan yang ditemukan / semua file yang diusulkan; recall = file acuan yang ditemukan / semua file acuan. Keduanya perlu scope sama: perubahan BPMN, file/function dan status draft/approval.
- MAP/MRR/R-precision untuk retrieval yang punya ranking. Urutan daftar plan bukan ranking relevansi; jangan mengarang MAP agent dari urutan Markdown.
- Skor cosine/BM25/RRF bukan akurasi. Tinggi/rendahnya skor tidak menggantikan evaluasi terhadap acuan.
- Top-5/Top-10 batas tampilan pilihan user, bukan confidence threshold otomatis. Jangan memilih K setelah melihat kunci jawaban untuk mempercantik precision.
- NO_ACTION bukan prediksi file yang harus diedit. File-level match bukan berarti semua function di file itu berubah.
- File selection, sumber yang benar-benar dibaca, status proses, dan ketepatan isi rencana adalah empat hal berbeda. NEEDS_REVIEW tidak membuktikan tidak ada dampak; COMPLETE juga bukan bukti rencana benar.
- Acuan saat ini provisional dan beberapa kasus hanya melabeli satu anchor. Tidak boleh menghitung semua perubahan tanpa label sebagai negatif pasti. Jaga data development dan evaluasi baru terpisah.
- Output plan: current behavior, target behavior, rationale, concrete proposed changes, ordered steps, acceptance checks, assumptions dan evidence. Petunjuk developer dalam bahasa natural; bukan patch yang diterapkan ke ERP. JSON kanonis dirender menjadi Markdown.

## Pengujian dan batas operasional

```powershell
cd C:\BPMN-Impact-Mapper; $env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go test ./...
```

```powershell
cd C:\BPMN-Impact-Mapper\web; npm run typecheck; npm run check:encoding
```

Untuk laptop terbatas RAM, jalankan berurutan: GOMAXPROCS=1, GOMEMLIMIT=192MiB, GOFLAGS=-p=1. Bila dependensi offline tersedia, GOPROXY=off, GOSUMDB=off, GOTOOLCHAIN=local dan GOMODCACHE repo-local .data/go-mod-cache. Jangan mengubah environment global pengguna.

Tes lab bertag labdiagnostics dan opt-in environment. Tes normal tidak membuktikan kualitas LLM. Browser/build/embedding/full scan model dapat berat; jangan dinyalakan tanpa kebutuhan dan jangan mengganggu sesi aktif. Eksekusi Go test tidak sama dengan browser/runtime/deployment proof.

Git: stage path yang ditinjau, bukan git add .; jangan commit .data, cache, laporan privat, token, outputs besar atau screenshot. Jangan revert penghapusan BASELINE-METHODS.md milik user tanpa alasan. Periksa diff --check. Commit/push hanya sesuai permintaan Haqi; jaringan kantor dapat memblokir GitHub. Jangan bypass sandbox atau akses SSH.

## Rujukan aktif dan yang historis

- [Alur Focused terbaru](../docs/FOCUSED-COMBINED-REVIEW.md), [model settings](../docs/WEB-AGENT-SETTINGS.md), [arsitektur](../docs/ARCHITECTURE.md).
- [Metode](../docs/METHODOLOGY.md), [semantic](../docs/SEMANTIC_RETRIEVAL.md), [batas BPMN](../docs/BPMN-CAPABILITY-MATRIX.md).
- [Paket input](../experiments/simple-erp/current-staff-v2/README.md), [baseline traceability](../experiments/simple-erp/baseline-traceability/README.md).
- TA-DIRECTION-CONCLUSION.md memuat arah penelitian; beberapa bagian lama pernah menyebut agent belum diimplementasikan dan bobot historis. Kode serta status terbaru harus diperiksa jika bertentangan.
- PLAN.md dan dokumen bertanggal memuat keputusan/hasil lama; bukan otomatis backlog aktif. Jangan menghapus bukti eksperimen yang mendasari angka TA hanya karena tanggalnya lama.

## Prompt singkat untuk melanjutkan dengan agent lain

> Baca CLAUDE.md dan collab/CLAUDE-GPT.md, STATUS.md, WORKLOG.md. Periksa git status dan instruksi lokal. Lakukan scan source Mapper secara bertahap serta Simple-ERP secara read-only, hindari secrets/cache/dependency. Verifikasi status handoff dari kode dan checkpoint. Lanjutkan prioritas tertinggi tanpa mengubah source ERP, acuan evaluasi atau konfigurasi sesi aktif diam-diam. Laporkan temuan dengan file/function yang jelas dan tes yang benar-benar dijalankan. Sebelum selesai, update STATUS dan tambahkan WORKLOG; jangan mengklaim peningkatan akurasi dari tes mock.
