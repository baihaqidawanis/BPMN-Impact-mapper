# TA Direction Conclusion

## Proposed title

**Evidence-Constrained Agentic Change Planning from BPMN Changes Using Deterministic Impact Mapping**

Versi Indonesia:

**Perencanaan Perubahan Perangkat Lunak Berbasis Agentic AI dari Perubahan BPMN Menggunakan Deterministic Impact Mapping Berbasis Evidence**

## Research objective

Penelitian membangun alat pihak ketiga yang mendeteksi perubahan BPMN, mengambil kandidat file dan symbol perangkat lunak yang mungkin terdampak, memvalidasi kandidat berdasarkan evidence, lalu menghasilkan rencana perubahan lintas layer. Output akhirnya adalah change plan, bukan modifikasi source code otomatis.

## Final architecture

```text
Repository + BPMN Lama + BPMN Baru
                |
                v
        BPMN Graph Differencing
                |
                v
      Automatic Retrieval Methods
       1. BM25-only
       2. Semantic-only
       3. BM25 + Semantic Fusion
       4. Graph-Aware Hybrid
                |
                v
 Canonical Impact Report + Evidence
                |
                v
 Evidence-Constrained Context Retrieval
                |
                v
       Impact Validation Agent
                |
                v
       Change Planning Agent
                |
                v
         Plan Review Agent
                |
                v
 change-plan.json + change-plan.md
```

## Position of RAG

RAG diperlukan pada lapisan agentic sebagai mekanisme penyedia konteks, tetapi bukan metode impact mapping kelima dan bukan dependency wajib bagi pipeline deterministik.

Retrieval untuk agent harus dibatasi pada:

- BPMN change dan graph context terkait;
- kandidat file/symbol dari canonical impact report;
- bounded evidence excerpts;
- dependency atau call relation yang telah didukung index;
- kontrak API, test, schema, dan artifact terkait yang ditemukan melalui retrieval terkontrol.

Vector database tidak wajib. Tahap awal dapat memakai indexed artifacts dan deterministic retrieval yang sudah tersedia. Semantic embedding dapat ditambahkan sebagai salah satu retriever, tetapi seluruh file repository tidak boleh langsung dimasukkan ke prompt.

## Automatic mapping methods

### 1. BM25 Retrieval

Baseline lexical retrieval. Perubahan BPMN menjadi query untuk meranking file dan symbol berdasarkan token label, path, identifier, route, state, table, test, dan dokumentasi yang diizinkan.

### 2. Semantic Retrieval

Baseline embedding retrieval untuk vocabulary mismatch, misalnya label BPMN `Approve purchase request` dan symbol code `AuthorizeProcurementDemand`. Model, checksum, preprocessing, chunking, threshold, serta runtime harus dipin agar reproducible.

Cara kerjanya:

1. Graph diff menghasilkan deskripsi perubahan BPMN dari change type, label lama/baru, actor, tipe elemen, predecessor/successor, kondisi, state, dan data requirement.
2. Source index membentuk dokumen terbatas per file atau symbol dari relative path, module, layer, language, nama symbol, komentar, dan source excerpt.
3. Model embedding lokal mengubah query BPMN dan dokumen source menjadi vektor angka.
4. Cosine similarity meranking file/symbol; tie diurutkan deterministik.
5. Kandidat di bawah threshold ditolak. Kandidat yang lolos selalu membawa `SEMANTIC_SIMILARITY` evidence dan action `VERIFY` karena similarity bukan bukti kausal.

Semantic Retrieval bukan RAG. Ia hanya mengambil kandidat. Pipeline menjadi RAG-like ketika evidence terpilih kemudian diberikan kepada LLM untuk membuat `change-plan.md`.

### 3. BM25 + Semantic Fusion

Metode otomatis yang menggabungkan ranked list BM25 dan Semantic menggunakan Reciprocal Rank Fusion (RRF), bukan menjumlahkan cosine dan BM25 mentah yang skalanya berbeda. Implementasi memakai `k=60`, menyimpan kontribusi rank sebagai `RRF_FUSION` evidence, dan selalu menghasilkan action `VERIFY`. Nilai RRF adalah konsensus ranking, bukan probabilitas file pasti berubah.

### 4. Graph-Aware Hybrid

Metode utama yang menambahkan exact trace ID, symbol/path similarity, structured evidence, BPMN graph context, dan static source relations pada hasil retrieval. Implementasi Hybrid sekarang menggabungkan ranking graph-aware tersebut dengan Semantic menggunakan RRF. Kandidat yang hanya berasal dari Semantic tetap `VERIFY`; action kuat hanya dipertahankan bila didukung sisi graph-aware. Implementasi ini masih harus diuji dengan hidden ground truth sebelum diklaim lebih akurat.

Walaupun fusion dan Hybrid sudah diimplementasikan, Semantic Retrieval tetap harus dievaluasi sebagai metode independen. Pemisahan ini diperlukan untuk mengukur apakah Semantic benar-benar menambah recall atau hanya menambah false positive.

## Current implementation status

- BM25-only: implemented dan dapat dipilih melalui CLI/API/web;
- Semantic-only: implemented secara offline dengan local embedding model;
- BM25 + Semantic Fusion: implemented dengan deterministic RRF;
- Graph-Aware Hybrid: implemented sebagai fusion antara graph-aware ranking dan Semantic;
- Configuration-Driven: hanya compatibility adapter internal, bukan metode utama TA;
- LLM/agentic planning: belum diimplementasikan dan tetap berada setelah evaluasi retrieval.

Synthetic smoke test, unit test, dan browser E2E membuktikan pipeline dapat berjalan. Hasil tersebut belum membuktikan akurasi atau generalisasi.

## Recommended next development

1. Bekukan dataset evaluasi: repository revision, BPMN lama, BPMN mutasi, Top-K, model, dan konfigurasi preprocessing.
2. Buat hidden ground truth file/symbol terdampak melalui dua reviewer setelah inference, kemudian lakukan adjudication.
3. Jalankan empat metode pada input identik dan ukur Recall@k, Precision@k, MRR/MAP, false discovery proportion, unresolved rate, latency, dan reproducibility.
4. Analisis error per jenis perubahan BPMN: label, penambahan/penghapusan task, gateway branch, sequence flow, actor, state, dan data requirement.
5. Setelah metode retrieval terpilih berdasarkan evidence eksperimen, implementasikan evidence-constrained planning agent untuk menghasilkan `change-plan.json` dan `change-plan.md` tanpa mengubah source code.

## Agent responsibilities

### Impact Validation Agent

- menerima canonical report, bukan repository mentah;
- memeriksa kandidat dan evidence secara read-only;
- dapat meminta retrieval konteks tambahan secara terbatas;
- menghasilkan verdict `CONFIRMED_RELEVANT`, `NEEDS_REVIEW`, `INSUFFICIENT_EVIDENCE`, atau `REJECTED`;
- tidak boleh mengarang file atau symbol.

### Change Planning Agent

- menerima hanya kandidat tervalidasi dan unresolved impacts;
- menyusun urutan perubahan lintas backend, API, database/schema, frontend, style, test, configuration, dan documentation jika didukung evidence;
- menjelaskan tujuan perubahan pada setiap file/symbol;
- mencatat dependency order, risiko, asumsi, dan kebutuhan test;
- tidak menghasilkan atau menerapkan source-code patch.

### Plan Review Agent

- memastikan setiap langkah menunjuk file/symbol yang ada;
- memastikan setiap klaim memiliki evidence reference;
- menolak unsupported suggestion;
- memisahkan fakta, asumsi, dan unresolved question;
- memastikan plan tidak keluar dari scope BPMN change.

## Final output

```text
outputs/<analysis-id>/
  impact-report.json
  impact-report.md
  agent-validation.json
  change-plan.json
  change-plan.md
  plan-review.json
```

Isi minimal `change-plan.md`:

1. repository revision dan input fingerprint;
2. ringkasan BPMN changes;
3. validated impacts per software layer;
4. ordered implementation steps;
5. file, symbol, required action, dan change intent;
6. evidence dan graph/path justification;
7. tests yang perlu dibuat atau diperbarui;
8. dependency order dan rollout consideration;
9. assumptions, rejected suggestions, dan unresolved questions;
10. explicit non-impacts jika dapat dibuktikan.

## Evaluation design

Eksperimen dibagi menjadi dua bagian agar scope tetap terkendali.

### Experiment A: impact retrieval

Bandingkan BM25-only, Semantic-only, BM25 + Semantic Fusion, dan Graph-Aware Hybrid pada input dan hidden ground truth yang sama menggunakan Recall@k, Precision@k, MRR/MAP, file/module/layer accuracy, action classification, unresolved rate, false discovery proportion, latency, dan reproducibility.

### Experiment B: agentic planning

Gunakan satu frozen Hybrid report sebagai input tetap, kemudian bandingkan:

1. deterministic impact report tanpa agent;
2. satu planning agent dengan retrieved evidence;
3. validation, planning, dan review agents dengan retrieved evidence.

Evaluasi plan menggunakan grounding rate, unsupported-file rate, evidence coverage, plan completeness, reviewer agreement, correctness of ordering, latency, dan cost. Jangan mengalikan empat mapping methods dengan seluruh konfigurasi agent karena scope eksperimen menjadi terlalu besar.

## Scope boundary

Penelitian berhenti pada evidence-backed change planning. Tool tidak mengeksekusi repository, tidak memodifikasi source code, tidak membuat migration, tidak melakukan deployment, dan tidak mengirim seluruh repository ke provider eksternal.

Automatic code modification dapat menjadi pekerjaan lanjutan. Memasukkannya sekarang akan mengubah tujuan penelitian dari impact mapping dan planning menjadi autonomous software modification, meningkatkan risiko keselamatan, memperbesar evaluasi, dan mempersempit perbedaan dengan topik agentic code-generation lain.

## Final decision

Configuration-Driven JSON dikeluarkan dari metode utama dan UI produk karena membutuhkan pemetaan manual sebelum inference, sedangkan tujuan produk adalah analisis otomatis dari repository dan dua BPMN. Adapter lama boleh dipertahankan sementara hanya untuk kompatibilitas internal, bukan bagian eksperimen utama.

RAG digunakan secara terbatas untuk memberi agent konteks yang telah difilter. Automatic retrieval dan graph evidence tetap menjadi sumber kandidat utama. AI agents bertugas memvalidasi evidence dan membuat repo-wide change plan yang dapat diaudit. Tidak ada perubahan source code otomatis dalam scope TA saat ini.

## Open-access literature, 2021-2026

Tidak ditemukan satu paper yang sudah menyelesaikan pipeline persis `BPMN change -> semantic source-file retrieval`. Karena itu landasan disusun dari tiga bagian yang berdekatan dan batas klaimnya harus disebutkan.

### Direct BPMN semantics

1. Di Martino et al. (2023), *A tool for the semantic annotation, validation and optimization of business process models*. Menggunakan word embedding dan semantic similarity pada label task, event, data object, dan participant BPMN. Ini mendukung representasi semantik label BPMN, tetapi tidak mengambil source-code impact. [Open-access PDF](https://onlinelibrary.wiley.com/doi/epdf/10.1002/spe.3184) · [DOI](https://doi.org/10.1002/spe.3184).

### BPMN-to-source traceability

2. Vega-Márquez et al. (2022), *Mapping Study on Traceability Between BPMN Models and Source Code*. Memetakan state of the art BPMN-to-code traceability dan menyebut graph sebagai teknik yang berkembang untuk versioning/change impact. Ini mendukung problem dan research gap, bukan algoritma embedding. [Open-access PDF](https://ijaseit.insightsociety.org/index.php/ijaseit/article/download/16170/pdf_2013/39081) · [DOI](https://doi.org/10.18517/ijaseit.12.1.16170).

### Semantic and hybrid code retrieval

3. Li et al. (2022), *CodeRetriever: A Large Scale Contrastive Pre-Training Method for Code Search*. Mendukung semantic natural-language-to-code retrieval pada beberapa bahasa dan granularitas kode. [Open-access PDF](https://aclanthology.org/2022.emnlp-main.187.pdf) · [ACL Anthology](https://aclanthology.org/2022.emnlp-main.187/).

4. Li et al. (2023), *Structure-Aware Language Model Pretraining Improves Dense Retrieval on Structured Data*. Mendukung dense retrieval yang sadar struktur dan dievaluasi pada code search, tetapi bukan BPMN change impact. [Open-access PDF](https://aclanthology.org/2023.findings-acl.734.pdf) · [ACL Anthology](https://aclanthology.org/2023.findings-acl.734/).

5. Lee et al. (2023), *On Complementarity Objectives for Hybrid Retrieval*. Mendukung alasan sparse lexical dan dense semantic retrieval digabung karena membawa sinyal yang saling melengkapi. Datasetnya bukan BPMN/source impact sehingga manfaat pada TA tetap harus diuji. [Open-access PDF](https://aclanthology.org/2023.acl-long.746.pdf) · [ACL Anthology](https://aclanthology.org/2023.acl-long.746/).

6. Xiangyang Li et al. (2025), *CoIR: A Comprehensive Benchmark for Code Information Retrieval Models*. Menunjukkan code retrieval perlu evaluasi lintas task/domain dan bahwa leaderboard tunggal tidak membuktikan generalisasi. [Open-access PDF](https://aclanthology.org/2025.acl-long.1072.pdf) · [ACL Anthology](https://aclanthology.org/2025.acl-long.1072/).

Literatur ini membuat Semantic Retrieval layak sebagai metode eksperimen, bukan membuktikan bahwa model saat ini akurat. Klaim akhir tetap memerlukan hidden ground truth, blind review, ablation per metode, dan metrik retrieval yang telah ditentukan sebelum melihat hasil.
