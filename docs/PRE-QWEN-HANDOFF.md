# Handoff untuk percobaan berikutnya

## Yang sudah siap

- Paket aktif [current-staff-v2](../experiments/simple-erp/current-staff-v2/README.md)
  memuat lima pasangan yang konsisten dengan baseline ERP setelah perapian.
- Source/referensi terikat hash; acuan tetap terpisah dari inference.
- Modul `features/analysis/run-analysis.ts` mengatur single/comparison melalui
  port analisis yang disuntikkan. React mengelola tampilan/state; `lib/api.ts`
  menangani HTTP; `analysis-repository.ts` menangani path privat. Modul use case
  tidak mengimpor React, window/localStorage, adapter HTTP atau aturan Simple-ERP.
- Backend tetap menggunakan domain/application contracts dan adapter untuk XML,
  inventory/index/retrieval, persistence, HTTP dan model. Tes arsitektur memeriksa
  arah dependensi paket inti; UI baru tidak mengubah engine/ranking tersebut.
- Library, hubungan antarlevel, History dan pemulihan sesi tetap terpisah dari
  retrieval dan review. Menjelajah diagram tidak menjalankan Qwen.

## Jalankan lokal

Backend (satu baris PowerShell dari folder Mapper):

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-impact-mapper-server --agent-model qwen3.5:9b-q8_0 --agent-url http://127.0.0.1:11435
```

Frontend di terminal lain:

```powershell
cd C:\BPMN-Impact-Mapper\web
npm run dev
```

Buka `http://localhost:3000`. Mulai dari Overview -> New analysis. Isian dan
file kasus pertama ada di README paket aktif. Hybrid membutuhkan runtime
embedding lokal; BM25 bisa dipakai untuk memastikan alur aplikasi tanpa itu.

## Saat VPN sudah tersedia

Terminal terpisah untuk tunnel (biarkan terbuka):

```powershell
ssh -N -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -L 127.0.0.1:11435:127.0.0.1:11434 mci@10.28.83.127
```

Gunakan model lokal yang tersedia dan diizinkan di lab; nama pada contoh adalah
model yang terakhir dipakai dalam sesi ini. Jangan memilih varian cloud tanpa
meninjau tujuan pengiriman source. Password SSH tidak dimasukkan ke aplikasi.

Setelah mapping, buka Planning dan periksa model. Mulai dari Focused review;
seluruh perubahan BPMN diproses dalam batch internal dan disatukan menjadi satu
rencana. Tidak perlu memilih satu ADD_TASK. Path repository dibawa dari Mapping.
Full-repository review tetap berarti semua chunk source yang diizinkan oleh
inventory, bukan seluruh byte termasuk secrets/dependencies/excluded corpus.
Ekspor plan dan agent context/checkpoint setelah selesai.

## Yang masih perlu diuji dengan model nyata

Kualitas rekomendasi file, kesesuaian bukti, keberhasilan seluruh review dan
kegunaan rencana teknis pada baseline ini. Tes respons model tiruan membuktikan
alur/checkpoint, bukan kualitas reasoning. Seleksi Qwen tidak dianggap kunci
jawaban; bandingkan dengan acuan yang sama secara terpisah. Tidak ada klaim
kenaikan precision/recall dari pekerjaan persiapan ini.

Tidak diperlukan penambahan metode, bulk analysis, framework agent baru atau
perubahan source ERP sebelum menjalankan percobaan pertama ini.

## End-to-end offline gate (2026-11-03)

The actual HTTP Ollama adapter, asynchronous API, source snapshot, orientation,
scan, planning, final consistency review, persistence, Markdown download and
restart restoration are exercised together against a local model stub. The
multi-change/README scenario passes. This verifies plumbing, not model judgment.
A separate test sends the real combined BPMN structures and all 20 changes with
a maximum 4096-byte source chunk and a maximum-size README summary through the
32K-context adapter. Derived adjacency and duplicate inline values are omitted
only from model transport; the complete elements/flows and all change identities
remain available. The engine still validates exact evidence/targets after decoding.

Use a fresh AI session after updating backend/prompt/corpus policies. Large final
plans can still exceed the bounded consistency-review request; they are marked
NEEDS_REVIEW instead of being silently truncated. No claim of arbitrary repository
scale, model accuracy or VPN availability is implied by these offline gates.

## Retry checklist after the November 4 update

1. Restart the backend to load the changed agent action contract.
2. When campus VPN is available, open the SSH tunnel above and leave it running.
3. Open a saved mapping, or run Mapping with the before/after BPMN pair.
4. In Planning, Check model, select Focused review, and Start new AI review.
5. All changes are processed; the final plan shows unresolved requirements explicitly. Do not resume an old session created under an older model/protocol identity.
6. Export plan Markdown and agent context JSON. A NEEDS_REVIEW result is a partial proposal, not a verified final answer.

Planner evidence requests are now offered only while tools and revision allowance
remain. Once exhausted, the model must describe uncertainty in its plan instead
of requesting an unavailable handoff. Source evidence validation remains strict.

Hasil pemeriksaan terbaru: [audit diff/acuan](EXPERIMENT-DIFF-REFERENCE-REVIEW.md) dan [evaluasi offline 4 November](OFFLINE-EVALUATION-2026-11-04.md). Acuan mencakup perubahan utama, bukan seluruh impact proses.

## Focused coverage upgrade

Focused review now includes all displayed candidates per BPMN change, in two-candidate batches. It may take longer than the old three-seed workflow. New plan fields separate observed behavior, required behavior and design rationale. Restart backend and start a fresh session after this change. See [agent review design](AI-AGENT-REVIEW-DESIGN.md). No live model request was made during implementation.
