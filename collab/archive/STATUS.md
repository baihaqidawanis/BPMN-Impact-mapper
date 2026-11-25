# Status bersama

Checkpoint handoff UI: 21 November 2026, ditulis GPT/Codex. Entri di bawah bagian terbaru menyimpan riwayat pengembangan.

## Handoff UI Gemini

- Scope aktif untuk Gemini adalah UI/UX. Mulai dari [brief Gemini](GEMINI-UI-HANDOFF.md) dan [DESIGN.md](../DESIGN.md). Arahan desain terakhir mempertahankan sidebar charcoal full-height rata kiri, workspace terang, dan aksen biru.
- Untuk kontinuitas kerja, Gemini membaca `CLAUDE-GPT.md`, STATUS, WORKLOG, dan dokumentasi kedua repo sejak awal. Jika Haqi meminta takeover setelah Codex berhenti, lanjutkan tugas yang dipilih Haqi dari catatan dan diff terbaru. Simple-ERP tetap menjadi studi kasus read-only untuk pekerjaan Mapper.
- Planning sekarang punya Focused Standard, Light, Ultra Light, dan Full repository review. Light adalah pilihan awal. Opsi yang disimpan bisa dipulihkan ketika membuka sesi terdahulu. Status, draft, dan progress berasal dari API sesi. Refresh/deep link perlu memulihkan sesi yang sama.
- Stopwatch memakai waktu aktif backend sejak sesi mulai. Waktu berhenti saat pause dan terakumulasi saat resume. Sesi lama tanpa data waktu menampilkan status unavailable. Estimasi mode Focused muncul setelah setidaknya dua batch selesai dan masih ada pekerjaan batch. Labelnya menunjukkan waktu menuju final review.
- Ultra Light terbaru bisa melakukan pemeriksaan tes tambahan, satu batch cadangan kecil per perubahan, dan satu koreksi terarah. Batas lookup awal, pengaturan mode, dan aturan review dimiliki backend. Gunakan sesi dummy atau saved analysis untuk preview UI. Saved analysis Gateway yang dipakai pada dev server lokal: http://localhost:3001/?analysis=87ab0d43f1710f3910205478ab870d56
- Handoff ini menemani perubahan kerja yang masih ada di workspace. Periksa diff lebih dulu. Kepemilikan perubahan backend dan eksperimen tetap dikoordinasikan Haqi.

## Pekerjaan aktif dan ownership

Latest UI request: implemented Change Explorer in ReportView, replacing repeated full-report change cards with sticky change navigation and a selected-change reading pane. Show all changes and By file preserve complete report views; filters/export data unchanged. Initial selection handles grouped flow edits. CSS includes desktop/tablet/phone layouts; original sidebar retained. Typecheck/encoding passed; browser verification not claimed. No model/API/backend/agent-state edits for this UI task.

18 November session9e701586 audit: Planner PAUSED at7turns/4tools due output token exhaustion, not a demonstrated VPN failure. Added typed output-capacity error, automatic multi-target Planner partition and Focused Analyst split, Reviewer capacity fallback, and clearer API error. Single-target exhaustion remains paused without repeating format repair. Focused protocol v12 requires NEW session after BE restart. Related agent tests passed incl new split/no-retry regressions; no live inference performed. UI sidebar restored flush-left on owner request, editorial page-title font added using installed fonts; no visual approval claimed.

18 November UI continuation: inspected image.png reference, applied more distinctive composition rather than only CSS refinement. Overview now asymmetric welcome/current-analysis section and saved reports/library work area. Sidebar inset, empty Mapping split explanation/live checklist. All values remain actual workspace data and actions reuse existing callbacks. Typecheck/encoding passed; visual acceptance still Haqi's explicit requested role. No model/agent/backend/session changes in this UI task.

Latest UI redesign: full-width collapsible analysis setup replaces left rail; report is a single reading surface with ruled groups, Planner output uses label/content columns, overview and stage controls recomposed. All old handlers retained. Paused-stage label and automatic retry for failed status reads added after inspecting live session3121cdf: PAUSED, analyst3turns/1tool, output6144token limit reached before JSON. On-all Thinking remains unproven; no session/model settings changed. TypeScript/encoding checks passed. Visual acceptance remains with Haqi as requested; no browser/build claimed. If trying Thinking all-on, output8192 is available but not guaranteed, requires new session due config binding. A smaller trial can use only Planner Thinking.

Latest pre-Thinking repair after e2aa4594 / sa5921b: actual Go declaration facts (parameter order including grouped parameters, receiver declared fields) now accompany named source reads/searches from immutable snapshot; source tool identity v13. Reviewer transport adds BEFORE vs PROPOSED_AFTER frame and file responsibility hints (documentation/tests/implementation); no verdict fabricated or reference used. Planner/Reviewer instructions allow documentation edits to their own rules, not Go/TS edits under Markdown. Declaration/frame regression tests and agenttools/llm/planning/api/agentjobs tests passed. Restart BE and NEW review needed. No live inference or measured accuracy improvement yet. UI source edits exist and localhost3000 responds Next.js200; active-browser CSS/version not confirmed. Haqi approves a more radical UI later, but explicitly prioritizes agent readiness for Thinking tonight.

UI update (Haqi selected Anti Slop during): palette preserved, source/CSS changes across app shell, Mapping form/results, Planning/agent settings/progress, Canvas, overview/history/library and Research. DESIGN.md and docs/UI-REFINEMENT-2026-11-17.md record direction and checks. Typecheck, encoding, 23 CSS parse and diff checks passed; base text contrast checked. No browser visual approval claimed; user explicitly will inspect. No backend restart, inference or agent-state mutation performed for UI. Complete visual gate remains pending owner review.

Latest offline repair after plan b2d7ca61 (s71e480): Focused v11 shares prior same-file proposal contracts with later Planner batches, binding-checked and never promoted to source. Reviewer transport now retains current_behavior claims instead of hiding them. Mixed review can correct clean REVISE targets while leaving NEED_EVIDENCE targets unresolved; filtered feedback and other proposed contracts reach the correction. Analyst gap criteria and experimental Reviewer before/after evaluation instructions tightened. Full go test ./... passed; final correction regression and vet planning/llm passed. No live inference or new precision claim. Restart BE, new session, same saved analysis; thinking optional, output/context budgets unchanged, 10-minute turn timeout remains. UI held for agent work.

Latest: hasil 81e4d3a / sesi 37b2e0 sudah diaudit: 55 turns, 41 tools, semua 5 batch prepared tetapi combined dan per-file review CONTEXT_LIMIT. Added focused v10 target-level fallback after file overflow, with full source/steps/checks and mandatory final contracts. Generic Planner/Reviewer instructions now check explicit SQL constraints, declared fields, same-requirement consistency and operation scope. Regression tests cover all targets, rejection blocking approval and cached resume. Offline only; live improvement remains unproven. No automatic semantic deduplication or silent repair of conflicting recommendations.

Update offline setelah audit: GPT memperbaiki merge evidence (exact function dahulu; named source boleh memperkaya whole-file; whole-file source tidak boleh memperkaya function sembarang), memproyeksikan final review ke semua target usulan sambil mempertahankan semua requirement BPMN, dan memperjelas prompt owner/operation/caller evidence. Protocol sekarang focused v9 / agent-loop v12. Tes Go menyeluruh lulus; belum ada inference baru. Restart BE dan mulai sesi baru dari saved analysis untuk validasi live, bukan Resume checkpoint lama. Retrieval weights dan source ERP tidak diubah. Konflik proposal tetap ditampilkan untuk review, tidak dihapus otomatis agar terlihat berhasil.

- Editor terakhir: GPT/Codex. Tugas terbaru: audit hasil plan 32d544b0f51abd63fb27ceb3be719b4a terhadap source ERP dan acuan primer.
- Saat berganti alat: baca git diff, konfirmasi tidak ada edit/inference lain yang aktif, lalu ganti bagian ownership ini. Haqi mengatur pergantian GPT/Claude. Subscription/limit tidak memicu handoff otomatis.
- Sesi 9B `6b59a2816212154c5d577bd6c012bd6b` dihentikan atas instruksi Haqi; API mengonfirmasi PAUSED, checkpoint tersimpan.
- PUT settings backend berhasil: ketiga role Qwen3.5 35B-A3B Q4, Thinking Off, Coding, 32768/6144. Checkpoint terbaru membuktikan Haqi sudah menjalankan trial dengan kebijakan Focused v8 (`combined_review_only: true`). Jangan meminta restart/upload ulang tanpa perubahan baru; periksa keadaan worker sebelum tindakan berikutnya.

## Perubahan terbaru, belum di-commit

HEAD yang terakhir diperiksa: `94cddc0 Improve agent review recovery and web model configuration`. Jangan menganggap remote menyimpan perubahan berikut.

1. Model dropdown lokal, tombol disclosure/download yang lebih jelas (web/components AgentSettings, Disclosure, ExportAction, QwenReview, ReportView, PlanningContextView).
2. Readiness error kategori spesifik dan aman: model belum terdaftar, thinking tidak didukung, embedding-only, remote alias, identitas berubah. Gagal check bukan otomatis VPN gagal.
3. Source tool v12: Go AST memberi keterangan function pemilik source windows, ditambah call literal facts yang sudah ada; target file tidak otomatis dipromosikan ke function.
4. Kompresi transport lossless untuk excerpt panjang identik melalui source_excerpts/excerpt_ref. Tidak membuang bukti; paket unik terlalu besar tetap ditolak.
5. **Focused v8 combined review**: child Analyst/Planner menyiapkan draft tervalidasi struktur, status child NEEDS_REVIEW/awaiting_combined_review. Parent mengumpulkan; Reviewer hanya pada fase gabungan. Split file dilakukan bila konteks memang terlalu besar. Maksimal satu koreksi Planner pada temuan REVISE beridentitas target tepat, tanpa Analyst ulang atau tools; NEED_EVIDENCE tetap unresolved. Review ulang bisa tetap menghasilkan NEEDS_REVIEW dengan draft dipertahankan. Full-repository mode belum mengikuti perubahan ini.
6. UI membedakan batches_prepared dari approval. Validasi JSON/evidence dan batas konteks tetap aktif. Prompt/source/engine identity berubah; butuh sesi baru.

File inti perubahan fase review: internal/planning/agents.go, focused.go, focused_review.go; tes focused_review_test.go, focused_test.go, review_failure_test.go. Bukti/context: internal/agenttools/function_scope.go, internal/llm/evidence_transport.go. Update UI: web/src/lib/agents.ts dan components/QwenReview.tsx.

## Verifikasi yang benar-benar sudah dilakukan

- `go test ./...` lulus setelah perubahan fase review; setelah guard tambahan NEED_EVIDENCE, paket planning/agentjobs/api diuji ulang dan lulus.
- `go vet` pada planning/agentjobs/agenttools/llm/api lulus.
- npm typecheck, check:encoding dan git diff --check lulus.
- Pengembangan offline tidak menjalankan browser atau inference. Haqi kemudian mengirim hasil live phase v8; hasilnya diringkas di bawah. Belum ada pengukuran latency terkontrol atau bukti peningkatan kualitas rencana keseluruhan.
- Simple-ERP dibaca untuk audit, tidak diedit. Tidak ada file handoff berisi credential atau token VPN.

## Hasil lama yang harus dibandingkan dengan hati-hati

| Hasil | Konfigurasi | Temuan | Batas |
|---|---|---|---|
| 5 November, sesi 25ce84d91e6de0a3d3244e004fa28e9c | Qwen3.5 9B Q8, prompt v6 | 3 file acuan dari 6 proposal: P50%, R100%, F1 66,7% | Draft NEEDS_REVIEW; snapshot statis di web/public/benchmarks/agent-add-task-2026-11-05.json |
| Plan c69f9ec91b090febdab3a8306f121ed5, sesi ecc875a6ff3c554560e4ea0bd7deba93 | Ketiga role Qwen3.5 35B, pipeline lama | 1/7 batch disetujui, 122 turns; backend salah NO_ACTION; satu frontend edit di sequence flow, tidak ada proposal ekspor untuk primary ADD_TASK | NEEDS_REVIEW; bukan controlled model comparison |
| Sesi 6b59a2816212154c5d577bd6c012bd6b | Percobaan 9B, sebelum phase v8 | Saat dicek 166 turns/50 tools/12 revisions; 4/10 kandidat, review/context berulang | Dihentikan, bukan final benchmark; baca checkpoint untuk angka terakhir |

Angka 55/100, 40/100 dll adalah penilaian manual lama tanpa rubrik konsisten; jangan jadikan benchmark/tren akurasi. Precision 1/1 terbaru dengan recall 1/3 mengabaikan scope change, sehingga tidak setara dengan primary ADD_TASK lama. Untuk primary ADD_TASK hasil ekspornya kosong: recall 0, precision tidak terdefinisi, bukan klaim tidak ada dampak. Dataset provisional belum memberi ground truth independen lengkap.

## Audit live terbaru ? 17 November, setelah handoff awal

Plan `32d544b0f51abd63fb27ceb3be719b4a`, sesi `0d7436b7f1a56c35a58fe44a5830f169`, model Qwen3.5 35B-A3B, status NEEDS_REVIEW. Focused v8 terkonfirmasi: 10/10 kandidat diproses, 7 batch, 5 prepared/2 planning issues, 83 turns/63 tools/0 revisions. Jumlah giliran lebih rendah dari trial 35B sebelumnya (122), tetapi konfigurasi/pipeline berubah dan ini bukan benchmark latency.

File-level primary ADD_TASK / ReasonReview: tiga unique proposed-edit files setelah exclude NO_ACTION dan dedup function/file assessments; semuanya sesuai tiga file acuan. P=100%, R=100%, F1=100% untuk acuan sementara ini. Ini **bukan** function correctness, kualitas rencana, atau approval; jangan dijadikan klaim sistem 100% akurat. MAP/MRR agent tidak tersedia karena tidak ada ranking relevansi usulan.

Defect penting yang diverifikasi dari source:
- Store proposal masih salah ke ReverseAdjustment untuk kebutuhan submit. ReverseAdjustment sudah menolak reason kosong dan hanya menangani reversal; operasi submit berada di TransitionAdjustment. Ada usulan `reason nil` padahal Go string.
- Backend NO_ACTION pada flow lain mengasumsikan validasi ada di domain tanpa bukti, bertentangan dengan kebutuhan alasan wajib.
- Proposal domain mengarang call adapter ke Adjustment.Submit dan field a.Reason; field tersebut tidak ada, dan pencarian saat audit hanya menemukan caller Submit pada domain test. Acceptance checks juga bertentangan dengan signature baru yang diusulkan.
- Migration NO_ACTION tetapi checks meminta NOT NULL/ALTER TABLE yang tidak ada. Requirement submit tidak otomatis mengharuskan mandatory reason pada semua status.
- Frontend forwarding reason saat submit benar secara arah; tests perlu menjaga success recovery scenario, menambah negative test, bukan mengubah recovery menjadi gagal.
- Final combined review berhenti: receipt `...-consistency.json` CONTEXT_LIMIT. Tidak ada persetujuan semantic akhir. Ada overlapping proposals dan dua planning batches unresolved.

Prioritas: (1) perbaiki asosiasi evidence ke owner function; periksa merge di planning/agents.go karena file-level read dapat digabung ke candidate fungsi pertama dalam file yang sama; (2) kecilkan paket review gabungan tanpa membuang requirement/source penting; (3) satukan proposal duplikat secara aman dan tampilkan konflik; (4) ulangi dengan model tetap, nilai selection terpisah dari correctness. Tidak ada perubahan source atau inference baru selama audit ini.

## Langkah berikutnya

1. Trial baru sudah tersedia dan diaudit di atas. Benahi defect konkret sebelum meminta Haqi mengulang. Jika source/BPMN tetap, saved analysis tetap dapat dipakai; kode/policy berubah memerlukan BE restart dan sesi baru.
2. Audit hasil Qwen35 phase v8: source yang diakses, fungsi tepat, tiga file inti, reason benar-benar diteruskan dan divalidasi sebelum submit, tests valid/invalid, unrelated actions dipertahankan.
3. Periksa apakah shared excerpt refs dipahami model dan apakah pembatasan review mengurangi revisi/calls. Tes lossless tidak membuktikan comprehension model. Koreksi bila bukti tercampur atau paket tetap sering overflow.
4. Prioritaskan defect konkret; jangan ganti banyak model/prompt/metrik sekaligus. Setelah stabil, evaluasi kasus add-xor/add-and/combined/control dengan acuan dan batas yang tetap. Tambahan skip-condition dapat menyusul; invisible-event-log discovery bukan scope aktif.
5. Bekukan implementasi dan eksperimen sebelum tenggat. Public SaaS, agent coding, KG, fine-tuning, model baru bukan prioritas saat ini.

## Data dan perubahan pengguna yang harus dijaga

- `.data/` dan `outputs/` bukan target cleanup. Planinng - Answer/ menyimpan laporan manual terbaru. Jangan mengubah laporan lama agar metrik terlihat lebih baik.
- BASELINE-METHODS.md sudah terhapus sebelum pekerjaan handoff ini; jangan restore diam-diam. README sempat menunjuk file tersebut; tautan diganti ke docs/METHODOLOGY.md.
- Semua perubahan kerja ditinjau sebelum commit; jangan stage seluruh repo. Belum ada permintaan push pada tahap ini.

## Template pembaruan status

Saat melanjutkan, ubah checkpoint/editor/tugas aktif dan tulis: selesai, belum selesai, hasil verifikasi, file disentuh, blocker dan aksi Haqi. Tambahkan detail ke WORKLOG; jangan menghapus hasil historis atau menyatakan worker berhenti hanya dari snapshot ini.
