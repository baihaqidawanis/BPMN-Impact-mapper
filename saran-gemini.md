# Audit Modularitas & Kualitas Kode — `C:\BPMN-Impact-Mapper`

- **Tanggal:** 2026-11-22
- **Sifat audit:** Read-only (berdasarkan pengukuran struktur paket, jumlah baris file/fungsi, dan rasio tes).

---

## 1. Ringkasan Singkat

**Verdict:** Repo **bukan spaghetti code** — arsitekturnya sudah **bersih dan modular** (~8/10 untuk proyek TA). Tidak disarankan refactor besar-besaran di tengah eksperimen TA; cukup lakukan pemecahan inkremental saat menyentuh hotspot di bawah.

### Metrik Struktur Repo
- **Go internal packages:** 32 paket dengan boundary tanggung jawab tunggal (`bpmn`, `diff`, `ingest`, `index`, `retrieval`, `semantic`, `fusion`, `planning`, `agenttools`, `llm`, `api`, `filestore`, `architecture`, dll.) dan 1 composition root (`internal/wiring`).
- **Rasio tes Go:** **169 file `_test.go`** vs **135 file Go produksi** (> 1:1).
- **Frontend Next.js:** 45 file TS/TSX, 24 CSS Modules terisolasi per komponen, **21 E2E spec Playwright**.

---

## 2. Hotspot yang Mulai Gemuk (Bukan Spaghetti, Tapi Perlu Dijaga)

### Backend Go
| File | Baris | Fungsi Terpanjang | Catatan Modularitas |
|---|---|---|---|
| `internal/llm/ollama_agent.go` | **1.190 L** | `Turn()` (378 L) | File paling gemuk; menggabungkan penyusunan prompt, context/budget trimming, HTTP transport, dan parsing/repair JSON. |
| `internal/planning/agents.go` | 772 L | `apply()` (242 L), `Run()` (216 L) | State-machine agent inti yang menampung banyak cabang transisi & validasi sekaligus. |
| `internal/diff/diff.go` | 728 L | — | Strukturnya masih bersih, tetapi menyatukan structural matching, element diff, flow diff, dan pattern grouping dalam satu file. |
| `internal/retrieval/retrieval.go` | 665 L | — | Menggabungkan `BM25Only`, legacy `Attach`, query building, inflection matching, dan evidence excerpting. |
| `internal/planning/full_repository.go` | 464 L | `Run()` (389 L) | Loop orkestrasi full-repo panjang (scan → batch plan → consistency review → pause/resume). |
| `internal/planning/focused.go` | 343 L | `Run()` (245 L) | Sudah mulai baik karena sebagian dipecah ke `focused_light.go` dan `focused_review.go`. |
| `internal/index/constructor_calls.go` | — | `Extract()` (272 L) | Fungsi ekstraksi wiring Go cukup panjang. |
| `internal/agenttools/source.go` | — | `Execute()` (202 L) | Dispatcher tool source mulai memanjang seiring tambahan jenis lookup. |

### Frontend (`web/src/`)
| File | Baris | Catatan Modularitas |
|---|---|---|
| `web/src/components/ReportView.tsx` | 519 L | Menggabungkan header summary, navigator change, `ChangeCard`, `ImpactRow`, dan helper label/skor. |
| `web/src/components/QwenReview.tsx` | 505 L | Menggabungkan polling session, kontrol rail kiri, render status/progress, dan grup log audit. |
| `web/src/app/page.tsx` | 438 L | Memegang orkestrasi state lintas 7 view (`overview`, `mapping`, `canvas`, `library`, `history`, `research`, `guide`). |
| `web/src/components/QwenReview.module.css` | 968 L | CSS module terbesar; beberapa rule lama (mis. `.delivery`, `.coverage`, `.roleModels`) bisa dipangkas saat touch berikutnya. |
| `web/src/components/ReportView.module.css` | 939 L | Cukup besar karena mencakup explorer 2-kolom + detail evidence + tabel skor. |

---

## 3. Saran Inkremental (Aman, Tanpa Risiko Regresi Besar)

1. **Jangan lakukan big-bang refactor** selama fase eksperimen/sidang TA — arsitektur sekarang sudah sehat dan teruji oleh 169 test Go + 21 E2E.
2. **Pecah hanya saat file disentuh (boy-scout rule):**
   - Jika menyentuh `internal/llm/ollama_agent.go`, pisahkan *prompt builder / context budget trimming* ke file terpisah (mengikuti pola `evidence_transport.go` & `review_transport.go` yang sudah rapi).
   - Jika menambah pola diff/grouping baru di `internal/diff/diff.go`, taruh logika pattern grouping di file terpisah (mis. `internal/diff/patterns.go`) agar `diff.go` fokus pada core graph comparison.
   - Jika menyentuh `internal/planning/focused.go`, teruskan pola pemecahan ke helper file (`focused_light.go`, `focused_review.go`).
   - Di frontend, jika `QwenReview.tsx` atau `ReportView.tsx` ditambah fitur baru, ekstrak sub-komponen (`ImpactRow.tsx`, `AuditLogs.tsx`) dan bersihkan class CSS yang sudah tidak dipakai di `QwenReview.module.css`.
3. **Pertahankan guard arsitektur & test discipline:** jalankan `go test ./...` dan `npm --prefix web run typecheck` + `check:encoding` setiap kali memecah file agar boundary paket dan kontrak E2E tetap terjaga.
