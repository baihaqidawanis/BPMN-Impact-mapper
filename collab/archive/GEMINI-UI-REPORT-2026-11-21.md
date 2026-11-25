# Laporan Perubahan UI/UX Gemini (2026-11-21)

Dokumen ini merangkum seluruh perubahan UI/UX yang dikerjakan Gemini (3 iterasi sesuai arahan Haqi) mengacu pada [GEMINI-UI-HANDOFF.md](GEMINI-UI-HANDOFF.md) dan [STATUS.md](STATUS.md) agar Codex dapat melakukan *code review*, verifikasi visual/Playwright, atau melanjutkan iterasi dari checkpoint yang sama.

## 1. Scope dan Batas yang Dijaga

- **Dalam scope:** hanya file presentasi di `web/src` (JSX presentasi, CSS Modules, design tokens) serta dokumentasi di `collab/WORKLOG.md` dan laporan ini.
- **Di luar scope (tidak disentuh):** backend Go (`internal/`, `cmd/`), `C:\Simple-ERP` (hanya dibaca secara read-only), `.data/`, konfigurasi model/prompt, skrip evaluasi, dan proses dev server lokal yang sedang berjalan di port `3001`.
- **Perilaku yang dipertahankan 100%:**
  - Sidebar charcoal full-height rata kiri beserta perilaku collapse & mobile (`WorkspaceShell`).
  - Kontrak API, handler submit/export, save/restore `?analysis=...` & `?agentSession=...`, polling status sesi, pause/resume, aturan stopwatch `active_ms` & estimasi (`done >= 2 && remaining > 0`), serta urutan/nilai skor retrieval.
  - Tanpa penambahan dependensi baru di `web/package.json`.

## 2. Evolusi Arah Desain pada Sesi Ini

1. **Iterasi 1:** perapian tokens, form grid, Change Explorer monospace path, dan 4-tile telemetri Planning.
2. **Iterasi 2 (Redesign Struktural):** atas feedback Haqi bahwa iterasi 1 terlalu halus, layout dirombak menjadi arsitektur 2-kolom di **Overview**, **Mapping Setup** (2-column split launchpad), dan **Planning** (2-column split workbench dengan *Before vs After Behavior Diff* di hasil plan).
3. **Iterasi 3 (Estetika Apple / macOS, Kurangi Card Berlapis):** atas feedback Haqi bahwa terlalu banyak *boxed card* terasa seperti AI slop dan kurang *smooth*, seluruh card berlapis, kotak border kasar, dan aksen garis kiri tebal dihilangkan — diganti permukaan putih tunggal yang tenang di atas kanvas `#f5f5f7`, pemisah *hairline* transparan, tombol pill lembut, *inset grouped list*, dan *frosted glass sticky bar*.

## 3. Daftar File yang Diubah dan Kondisi Terakhir

### A. Global Tokens, Shell, & Kontrol Bersama
- `web/src/app/globals.css`: palet Apple-inspired (`--canvas: #f5f5f7`, `--ink: #1d1d1f`, `--muted: #6e6e73`, `--blue: #0071e3`, `--line: rgba(0,0,0,0.075)`), token status semantik (`ok`, `warn`, `danger`), shadow lembut, radius `18px`/`10px`/`8px`, dan transisi halus (`cubic-bezier(0.25, 0.1, 0.25, 1)`).
- `web/src/app/page.module.css`: sticky `.workflowBar` dengan *frosted glass* (`backdrop-filter: saturate(180%) blur(20px)`), segmented control `Mapping / Canvas / Planning`, `.inputBar` hairline-separated tanpa card, dan `.draftNotice` lembut.
- `web/src/components/ExportAction.module.css`: tombol ekspor pill halus tanpa border kasar.
- `web/src/components/Disclosure.module.css`: trigger disclosure hairline transparan dengan pill label biru lembut.

### B. Workspace Overview
- `web/src/components/WorkspaceOverview.tsx` & `web/src/components/WorkspaceOverview.module.css`
  - Menghapus *marketing hero* 88px rata tengah dan `ProcessPreview`.
  - Menggantinya dengan header command bar ringkas + layout 2 kolom: tabel **Saved reports** dalam satu permukaan putih di kiri; **Active session**, statistik (`Saved analyses`, `Process diagrams`), dan ringkasan alur kerja duduk bersih langsung di atas kanvas di kanan dengan pemisah hairline (tanpa card bertumpuk).

### C. Mapping Setup & Input Readiness
- `web/src/components/AnalysisForm.tsx` & `web/src/components/AnalysisForm.module.css`
  - Layout **2-Column Split Launchpad** dalam satu permukaan putih menyatu: kiri (`01 Repository` + `02 Process versions` dengan panah `Before -> After`), kanan (`03 Search settings` + `Semantic runtime` + `Readiness strip` `Repository / Baseline BPMN / New BPMN` + tombol pill `Run impact analysis`).
  - Input filled lembut ala macOS (`#f5f5f7`) dengan focus ring biru halus; `repo_path` dan `revision` memakai monospace; file picker BPMN dengan indikator siap hijau lembut.
- `web/src/components/MappingEmptyState.tsx` & `web/src/components/MappingEmptyState.module.css`
  - Menghapus SVG dekoratif; hanya menampilkan daftar kesiapan input yang ringkas.

### D. Change Explorer
- `web/src/components/ReportView.tsx` & `web/src/components/ReportView.module.css`
  - Header ringkasan laporan dan 4 metrik tabular duduk langsung di kanvas dengan pemisah hairline; kontrol filter (`All / Code / Tests / Docs`) dan grouping (`By BPMN change / By file`) memakai segmented control.
  - Sticky `.changeNav` (`Process changes`) transparan di atas kanvas (gaya sidebar Finder) dengan sorotan biru lembut pada perubahan aktif dan tombol `Show all changes`.
  - Baris kandidat menampilkan path file monospace (`.fileDir` muted + nama file tebal, tetap satu string utuh), badge aksi pill, **score meter bar** visual di samping angka tabular, dan blok *technical evidence* bersih (`#fafafa`, `overflow-x: auto`).
  - `<summary>` pada `Show technical evidence` tetap non-grid (memenuhi asersi `report-results.spec.ts`).

### E. Planning & Telemetry Sesi Agent
- `web/src/components/AgentProgress.tsx` & `web/src/components/AgentProgress.module.css`
  - `TelemetryStrip` 4 kolom borderless dengan pemisah hairline vertikal menggunakan data sesi backend:
    1. **Active time** (`session.active_ms` live saat aktif, label `Paused` saat jeda, atau `Active time unavailable for this session` untuk sesi lama).
    2. **Est. to final review** (`About N min to final review` bila `live && recorded > 0 && done >= 2 && remaining > 0`).
    3. **Batch progress** (`batches_reviewed/batches_total` + `batches_prepared` vs `batches_needing_review`).
    4. **Candidate coverage** (`candidates_processed/candidates_total` atau `files_reviewed/files_total` & `chunks_reviewed/chunks_total`).
  - Mempertahankan tahapan `Analyst -> Planner -> Reviewer` (`aria-label="AI agent progress"` dan `data-state`).
- `web/src/components/QwenReview.tsx` & `web/src/components/QwenReview.module.css`
  - **2-Column Split Workbench**:
    - **Left Sticky Control Rail (`340px`):** duduk bersih di kanvas (status dot koneksi, model 3 role, path repo, **Apple inset grouped radio list** untuk 4 mode review, `Model settings`, dan tombol aksi sesi `Run / Start new AI review`, `Stop`, `Resume`, `Refresh`).
    - **Right Main Stage:** satu *document sheet* putih berisi empty workbench state sebelum run, status header, telemetri, pipeline 3 tahap, delivery bar, daftar proposal, dan grup `.auditLogs` sekunder.
- `web/src/components/AgentPlanResults.tsx`
  - Menambahkan tampilan **side-by-side Behavior Diff** (`Current behavior` vs `Required behavior`) dengan tint lembut di dalam `FileAssessment`.
  - Membungkus `.auditTable` (`Reviewer checks`) dalam `.tableWrap` (`overflow-x: auto`) agar bebas overflow di layar kecil.
- `web/src/components/PlanningContextView.module.css`
  - Melebarkan `.root` agar 2-column workbench Planning mendapat ruang penuh.

## 4. Verifikasi yang Sudah Dijalankan

| Pemeriksaan | Hasil | Catatan |
|---|---|---|
| `npm.cmd --prefix web run typecheck` | **Lulus** (exit 0) | Dijalankan ulang setelah Iterasi 3. |
| `npm.cmd --prefix web run check:encoding` | **Lulus** (exit 0) | UTF-8 bersih, tanpa mojibake. |
| `git diff --check -- web collab` | **Lulus** (exit 0) | Tanpa whitespace error. |
| Cek visual browser (`http://localhost:3001/?analysis=87ab0d43f1710f3910205478ab870d56`) | **Belum berhasil terhubung** | `DevToolsActivePort` Chrome tidak aktif; verifikasi visual dilakukan langsung oleh Haqi. |

## 5. Feedback Terbaru Haqi & Langkah Berikutnya

1. **Pengurangan kepadatan teks (prioritas UI berikutnya):** Haqi menilai masih terlalu banyak tulisan/penjelasan panjang di UI sehingga terasa seperti blog/dokumen teks. Langkah selanjutnya adalah memangkas microcopy yang berulang (mis. catatan *read-only* berulang di Mapping, paragraf disclaimer panjang di Change Explorer & Planning) menjadi label singkat 3–6 kata, dan memindahkan penjelasan metodologi detail ke disclosure atau tab **Guide** tanpa menghilangkan teks yang dibutuhkan tes E2E.
2. **Cek tes E2E Playwright (`web/tests/e2e/`):**
   - `web/tests/e2e/qwen-review.spec.ts` baris 30 masih mengharapkan default radio `Focused review`, sedangkan default di `QwenReview.tsx` adalah `"light"` (sesuai `STATUS.md`). Perlu sinkronisasi ekspektasi tes bila default `Light` sudah final.
   - Jalankan `report-results.spec.ts` dan `qwen-review.spec.ts` untuk memverifikasi seluruh selector dan layout mobile 390px.
3. **Layar pendukung (opsional):** selaraskan `AnalysisHistory`, `BpmnLibrary`, dan `BpmnEditor` dengan token Apple-inspired terbaru bila diperlukan.
