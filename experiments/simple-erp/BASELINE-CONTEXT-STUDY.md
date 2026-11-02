# Eksperimen konteks task baseline beberapa langkah

Opsi offline `multi-hop4` dibandingkan dengan `full` (Hybrid produksi).
Belum diaktifkan pada web atau default CLI analyze.

## Aturan yang ditetapkan sebelum run

- Hanya ADD_TASK/ADD_GATEWAY yang bersebelahan dengan node baru lainnya.
- Telusuri incoming/outgoing sequence flow secara terpisah, BFS maksimal empat
  edge, visited-set maksimal 128 node per arah, maksimal dua anchor per arah.
- Anchor adalah task dengan stable ID/type dan process/subprocess yang sama
  pada kedua versi. Nama diambil dari baseline; tidak memerlukan ID ada di source.
- Berhenti pada task baseline yang ditemukan per jalur. Jangan menyeberang
  process/subprocess atau melewati call activity; tidak memuat model eksternal.
- Tetangga langsung tetap utuh. Anchor dicatat terpisah dalam
  `context.baseline_anchors`, lengkap dengan arah, jumlah hop dan flow path.
- Query BM25/Semantic memakai label anchor, bukan menganggap ID anchor sebagai
  TRACE_ID elemen baru. Tidak menambah bobot ID/path atau confidence probability.
- Tidak memasukkan tabel mapping before, reference labels atau ground truth ke
  pencarian. Model, corpus, RRF, top-k dan setting lain tetap.

Empat hop dipilih agar node join pada rantai penambahan dapat menjangkau task
baseline di luar node-node baru. Ini batas engineering yang eksplisit, bukan
angka optimal berdasarkan teori atau hasil tune. Tidak melakukan sweep kedalaman.

## Dataset dan evaluasi

Manifest: `baseline-context-study-manifest.json`. Kasus: compound (7 node),
AND (6 node), tambah satu task (1 node), kontrol identik (0 node). Ini development
set terekspos yang saling terkait, acuan sementara satu reviewer AI. Referensi
AND diambil dari kebutuhan dua master checks yang sama, dengan ReasonDecision
dikecualikan. Reference JSON lama tidak diubah.

Dua varian, dua pengulangan setiap pasangan: 16 analisis baru. Acuan dibaca oleh
evaluator setelah prediksi; konstruksi metode hanya menerima repo dan BPMN.
Node coverage wajib lengkap. Sequence-flow details dikecualikan, bukan dilabeli
negatif. Union file top 5/10 per node dinilai dengan deduplikasi file.

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/baseline-context-study-manifest.json --output outputs/baseline-context-study-new --variants full,multi-hop4 --node-scope --timeout 1h
```

Output harus folder baru. Simpan perbandingan sebelum/sesudah dengan label yang
sama; hasil satu anchor historis tidak boleh dicampur dengan cakupan semua node.
Control tambah-task tunggal tidak boleh menerima perluasan karena tidak ada
cluster node baru. Analysis ID varian harus berbeda, input fingerprint sama.

## Verifikasi implementasi

Tes mencakup siklus, batas hop, parent scope, fan-out besar, batas jumlah anchor,
tetangga langsung tetap utuh, ketidakubahan konteks asli, label versus ID query,
pilihan opt-in pada BM25/Semantic, identitas varian dan isolasi default produksi.
Tes memakai graph/corpus sintetik dan embedder stub, bukan ukuran akurasi model.

Run pertama dihentikan preflight karena label_provenance harus memakai enum
`single-source-review`; metadata tersebut diperbaiki sebelum inference tanpa
mengubah labels. Run analisis memakai folder baru setelah preflight berhasil.

Keputusan adopsi harus mempertimbangkan perbaikan dan regresi pada seluruh
kasus, recall, MAP serta jumlah kandidat union. Empat kasus ini saja tidak
membuktikan generalisasi. Jika hasil tidak konsisten, default tetap dipertahankan.

## Hasil dan keputusan

Semua 8 sel kasus/varian (16 analisis) menghasilkan prediksi dan evaluasi identik
byte pada dua pengulangan. Fingerprint sama antar varian, analysis ID berbeda.
Keempat hasil `full` juga identik dengan prediksi Hybrid benchmark sebelumnya.
Anchor label/path cocok dengan BPMN, corpus coverage sama, dan taxonomy/elemen
perubahan tidak berubah. Tidak ada labels/setting diubah setelah run dimulai.

| Kasus | Varian | Precision@5 | Recall@5 | MAP | File union top5/node |
|---|---|---:|---:|---:|---:|
| Tambah satu task | Full | 60,00% | 100% | 1,0000 | 5 |
| Tambah satu task | Multi-hop4 | 60,00% | 100% | 1,0000 | 5 |
| AND | Full | 33,33% | 83,33% | 0,5879 | 12 |
| AND | Multi-hop4 | 36,67% | 91,67% | 0,8155 | 11 |
| Gabungan | Full | 37,14% | 85,71% | 0,6468 | 13 |
| Gabungan | Multi-hop4 | 40,00% | 92,86% | 0,7664 | 12 |

Kedua kontrol identik tidak menghasilkan perubahan/kandidat. Recall union file
tetap 100%; precision union AND naik 16,67% -> 18,18%, gabungan 23,08% -> 25%.
Ini penilaian file/node, bukan bukti function atau required-action tepat.

Target awal tercapai: file store untuk MasterChecksJoin naik 7 -> 1 pada kedua
kasus; CombineValidation naik 7 -> 2 pada AND dan 7 -> 3 pada gabungan.
Tetapi ada regresi: operations integration test untuk CheckWarehouse turun
5 -> 7 di kedua kasus; beberapa test lain bergeser turun meski tetap top 5.

**Keputusan: tetap opt-in eksperimen.** Hasil mendukung hipotesis konteks baseline
membantu cluster baru, tetapi dua kasus positif masih berhubungan erat dan
memiliki reference sementara. Belum ada pengujian proses/repository lain atau
review independen. Jangan menjadikan hasil ini klaim peningkatan umum atau
menganggap precision union 25% sudah menyelesaikan seleksi kandidat.

Artefak: `outputs/baseline-context-study/`, termasuk `comparison.md` dengan posisi
setiap file acuan sebelum/sesudah. Verifikasi:

```powershell
python experiments/simple-erp/summarize_baseline_context.py outputs/baseline-context-study
```

Tes Go seluruh package akhirnya lulus dengan `go test -p 1 -timeout 120s ./...`
dan GOMAXPROCS=2. Percobaan test paralel ketika embedding berjalan gagal karena
kehabisan memori saat kompilasi, lalu dihentikan. Verifikasi diulang berurutan
setelah embedding selesai; tidak mengubah model atau hasil eksperimen.
Tes query, siklus/scope, isolasi default dan Markdown anchor lulus. Baseline
traceability tetap diverifikasi; ERP tidak diubah, tidak ada commit/push.

Langkah berikutnya: kunci opsi ini, uji pada kebutuhan P2P atau modul berbeda
yang belum dipakai mengembangkan perluasan, lalu audit regresi test CheckWarehouse
sebelum mempertimbangkan default produksi atau threshold seleksi.
