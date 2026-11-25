# Pegangan kerja TA Haqi

Baca file ini untuk memahami penelitian dan repo. Buka [PROGRESS.md](PROGRESS.md) untuk mengetahui tugas aktif, bukti terakhir, dan langkah berikutnya. Haqi menentukan pergantian antar-agent. Catat hasil setiap sesi kerja di **PROGRESS.md** agar agent berikutnya bisa meneruskan dari titik yang sama.

## Tujuan penelitian

TA mengembangkan **BPMN Impact Mapper**, alat pihak ketiga yang membandingkan dua versi BPMN, mencari file dan fungsi yang mungkin terdampak, lalu menghasilkan rencana perubahan software berbasis bukti source. Judul kerja berbahasa Inggris adalah *Evidence-Constrained Agentic Change Planning from BPMN Changes Using Deterministic Impact Mapping*. Status judul mengikuti konfirmasi Haqi dan dosen.

BPMN Before menggambarkan perilaku yang tersedia pada baseline. BPMN After berisi rancangan perubahan bisnis. Mapper membaca repo source secara read-only dan mengeluarkan rekomendasi. Developer meninjau rencana sebelum implementasi. Sistem belum menguji bahwa kode yang diusulkan benar-benar berhasil saat diterapkan.

## Repo dan tanggung jawab

| Lokasi | Isi dan penggunaannya |
|---|---|
| `C:\BPMN-Impact-Mapper` | Repo TA. Backend Go, frontend Next.js/React, Canvas bpmn-js, embedding lokal Python, dan adapter Ollama. |
| `C:\Simple-ERP` | Objek studi kasus read-only. Backend Go modular, PostgreSQL, dan frontend Next.js. Baca README, `docs/PROCESS-MAPPING.md`, dan kode modul terkait. BPMN kanonis di `docs/bpm-haqi/roles/`. |
| `C:\POC TA` | POC historis untuk pola proses. Gunakan sebagai sumber konteks metodologi. Baseline pengujian Mapper saat ini berasal dari Simple-ERP. |

Di Simple-ERP, area utama Haqi mencakup P2P, Inventory, serta Master Data/Configuration. Peran utamanya Staff, Supervisor, dan Super Admin. L0 memberi konteks, L1 menunjukkan modul/proses, L2 merinci alur, dan L3 merinci aktivitas. Satu analisis membandingkan satu pasangan BPMN. Tautan di BPMN Library menavigasi ke level lain.

## Alur teknis

1. **Parse + diff BPMN.** Parser Go membentuk graf; diff mengidentifikasi penambahan, penghapusan, perubahan tipe, kondisi, lane, dan aliran. Konteks dapat memuat aktivitas tetangga dan kondisi lama/baru. Flow pendukung bisa dikelompokkan ke perubahan induk sambil tetap tersedia di JSON.
2. **Index source dan retrieval.** Source dibaca dari repo yang dipilih. Empat metode penelitian adalah **BM25**, **Semantic**, **BM25 + Semantic Fusion**, dan **BPMN-Context-Aware Hybrid Retrieval**. Hybrid utama memakai context-aware BM25 dan semantic cosine yang digabung melalui RRF dengan `k=60`. ID, path, dan bukti source membantu menjelaskan kandidat. Bobot ID 45% adalah metode historis. `intfloat/multilingual-e5-small` dipilih melalui environment untuk percobaan lokal terakhir. Pastikan identitas embedding pada laporan ketika membandingkan metode.
3. **Impact report.** Setiap perubahan dapat punya beberapa kandidat file atau symbol dan cuplikan source. Top K berlaku pada hasil mapping per perubahan. Ketika masuk Planning, beberapa perubahan terkait bisa digabung dan menambah kandidat terkait. File yang sama bisa dihitung pada lebih dari satu perubahan.
4. **Planning berbasis bukti.** Analyst menilai target, Planner membuat usulan edit, Reviewer memeriksa konsistensi dan dukungan source. Lookup memakai snapshot source yang disetujui. AST Go memberi fakta sintaks seperti deklarasi dan posisi argumen. Keluaran berupa JSON dan Markdown yang memuat perilaku saat ini, usulan, acceptance checks, dan bukti.
5. **Ekspor dan evaluasi.** Saved analysis menyimpan laporan serta BPMN. Agent session menyimpan checkpoint. Evaluasi memakai acuan yang disimpan terpisah dari jalur inference. Mapper mengusulkan perubahan source untuk direview developer.

| Area | Lokasi |
|---|---|
| BPMN dan mapping | `internal/bpmn`, `internal/diff`, `internal/app`, `internal/ingest`, `internal/index` |
| Retrieval dan ranking | `internal/retrieval`, `internal/semantic`, `internal/fusion`, `internal/wiring` |
| Agent dan sesi | `internal/planning`, `internal/agentjobs`, `internal/agenttools`, `internal/llm`, `internal/filestore` |
| API dan web | `internal/api`, `cmd/bpmn-impact-mapper-server`, `web/src`, `web/tests/e2e` |
| Evaluasi dan eksperimen | `internal/evaluation`, `internal/benchmark`, `experiments/` |

## Skenario TA

| Skenario | Perubahan yang menjadi subjek uji |
|---|---|
| **Add Task** | Tambah pengisian dan validasi alasan sebelum submit stock adjustment. Perubahan primer `ReasonReview`. |
| **Gateway** | Ubah kondisi submit adjustment, tambahkan persiapan respons penolakan dan penggabung jalur hasil. Perubahan acuan `role_staff_l3_adj_submit_valid`. |
| **Invisible Prime Task: Skip** | Adjustment dengan seluruh selisih nol melewati antrean approval dan ditutup tanpa posting ledger. |
| **Invisible Prime Task: Redo** | Supervisor minta revisi, lalu Staff memperbaiki dan mengirim ulang PR yang sama ke antrean review. |
| **Invisible Prime Task: Switch** | Saat approve PR, pemeriksaan ulang produk nonaktif bisa mengalihkan alur ke penolakan dengan alasan otomatis. |
| **Non-Free Choice (NFC)** | Saat submit PR, simpan keputusan `STANDARD` atau `ESCALATED`. Review kemudian memakai rute tersimpan untuk menentukan otorisasi. Perubahan ambang tidak boleh mengubah rute PR yang sudah disubmit. |

Pasangan [Add Task dan Gateway](../experiments/simple-erp/current-staff-v2/README.md) serta [Skip, Redo, Switch, dan NFC](../experiments/simple-erp/poc-patterns/README.md) memakai Before/After. Sebutan invisible pada TA mengacu pada pola perubahan otomatis yang direpresentasikan di rancangan After. Penemuan invisible transition dari event log berada di luar lingkup implementasi. XOR–AND–Stack disimpan sebagai fixture historis. [Batasan pola](../docs/INVISIBLE-TASK-SCOPE.md)

## Cara menilai hasil

- Ambang evaluasi akhir dan rubrik kuantitatif plan Markdown dikunci di [metodologi TA](../docs/METHODOLOGY.md#frozen-acceptance-criteria-for-the-thesis-evaluation).

- Pisahkan **ranking kandidat**, **cakupan source yang diperiksa**, **file yang diusulkan untuk edit**, dan **ketepatan isi plan**. Status `NEEDS_REVIEW` berarti masih ada catatan. Status selesai juga perlu pemeriksaan source.
- Hitung precision dan recall pada lingkup yang sama, misalnya satu perubahan BPMN dan file unik dengan proposal edit. `VERIFY` dan `NO_ACTION` perlu dipisahkan dari edit yang diusulkan. Untuk retrieval ranking gunakan metrik seperti Precision@K, Recall@K, MRR, dan MAP sesuai definisi studi.
- Gateway dan Add Task dipakai untuk pengembangan. Acuan Gateway yang tersedia memuat tiga file untuk satu kondisi. Angka ini belum mewakili seluruh skenario. Bekukan konfigurasi, repo revision, pasangan BPMN, dan kriteria penilaian sebelum percobaan skenario yang belum dipakai tuning.
- Simpan acuan, dokumen handoff, keluaran eksperimen, dan data sensitif di luar corpus saat menjalankan inference. Gunakan referensi setelah plan dibuat. Tes offline membuktikan perilaku yang diuji, sedangkan hasil live berulang dan penilaian source dibutuhkan untuk klaim kualitas.

## Menjalankan dev di Windows

Gunakan terminal terpisah untuk tunnel, BE, dan FE. Tunnel lab memakai sesi SSH yang diotorisasi Haqi. VPN kampus diperlukan bila jaringan tidak memiliki rute ke host lab.

```powershell
ssh -N -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -L 127.0.0.1:11435:127.0.0.1:11434 mci@10.28.83.127
```

```powershell
cd C:\BPMN-Impact-Mapper
$env:GOCACHE = "$PWD\.gocache"
$env:BPMN_IMPACT_SEMANTIC_MODEL = "intfloat/multilingual-e5-small"
go run ./cmd/bpmn-impact-mapper-server --addr 127.0.0.1:8082 --agent-url http://127.0.0.1:11435 --agent-model qwen3.5:35b-a3b-q4_K_M --agent-context 32768 --agent-output 6144 --agent-profile coding --agent-review-json
```

```powershell
cd C:\BPMN-Impact-Mapper
$env:BPMN_MAPPER_BACKEND_URL = "http://127.0.0.1:8082"
$env:BPMN_MAPPER_TEST_BUILD = "1"
npm.cmd --prefix web run dev -- -p 3001
```

Model utama pada percobaan terakhir adalah Qwen3.5 35B-A3B Q4 untuk ketiga peran, profile Coding, Thinking Off, dengan context 32768 dan output 6144. Cek konfigurasi sesi untuk memastikan kondisi run. Gunakan **Resume** untuk checkpoint yang kompatibel. Gunakan **Start new AI review** saat protokol, source snapshot, atau model berubah. Biarkan backend dan tunnel aktif selama run. Alias model cloud atau perubahan layanan GPU lab memerlukan persetujuan tersendiri.

## Cara bergantian antar-agent

1. Baca bagian **Lanjut dari sini** di [PROGRESS.md](PROGRESS.md). Cocokkan dengan `git status`, source, dan checkpoint terkait. Utamakan instruksi terbaru Haqi saat catatan sudah lama.
2. Ambil tugas yang ditunjuk Haqi. Gemini dapat membaca dokumentasi kedua repo. Untuk tugas UI, fokus pada presentasi serta handler yang sudah ada, gunakan [DESIGN.md](../DESIGN.md), dan jaga saved analysis, pause/resume, timing, serta keluaran studi kasus.
3. Lindungi `.data`, file ERP, bukti evaluasi, kredensial, dan perubahan agent lain. Hindari dua agent mengedit file yang sama bersamaan. Publikasikan perubahan hanya sesuai scope yang Haqi minta.
4. Verifikasi perubahan yang dibuat. Gunakan `go test ./...` untuk backend dan `npm.cmd --prefix web run typecheck` serta `npm.cmd --prefix web run check:encoding` untuk web. Sesuaikan uji browser dengan fitur yang disentuh. Catat hasil nyata dan gap verifikasi.
5. Sebelum selesai atau berpindah agent, perbarui **Lanjut dari sini** dan tambahkan entri singkat bertanggal di **PROGRESS.md**. Cantumkan tujuan, file, keputusan, tes, hasil, langkah konkret berikutnya, serta commit/branch jika ada.

Rujukan teknis lengkap ada di [arsitektur](../docs/ARCHITECTURE.md), [metodologi](../docs/METHODOLOGY.md), [kemampuan BPMN](../docs/BPMN-CAPABILITY-MATRIX.md), [alur Focused](../docs/FOCUSED-COMBINED-REVIEW.md), dan [arah TA](../TA-DIRECTION-CONCLUSION.md). Catatan lama disimpan di [arsip collab](archive/).
