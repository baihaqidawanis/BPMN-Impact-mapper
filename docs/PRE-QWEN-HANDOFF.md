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

Setelah mapping, buka Planning, periksa model, pilih ADD_TASK ReasonReview untuk
kasus pertama, lalu jalankan review. Path repository dibawa dari Mapping.
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
