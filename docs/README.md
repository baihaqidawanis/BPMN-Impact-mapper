# Peta dokumentasi

Mulai dari [README utama](../README.md) untuk menjalankan web. Untuk melanjutkan pengembangan lintas GPT/Claude, baca [pegangan bersama](../collab/CLAUDE-GPT.md) dan [status terbaru](../collab/STATUS.md).

Panduan agent aktif: [Focused combined review](FOCUSED-COMBINED-REVIEW.md). Catatan bertanggal lainnya menjelaskan versi historis, bukan konfigurasi model yang harus dipakai sekarang.

| Keperluan | Dokumen |
|---|---|
| Input demo Staff terbaru | [current-staff](../experiments/simple-erp/current-staff-v2/README.md) |
| Mapping before task ke file/function | [Staff P2P L3](../experiments/simple-erp/baseline-traceability/README.md) |
| Status perapian BPMN Staff | [normalisasi Staff](STAFF-BPMN-NORMALIZATION.md) |
| Konvensi seluruh role | [Staff, Supervisor, Super Admin](ROLE-BPMN-NORMALIZATION.md) |
| Elemen BPMN yang tercakup dan batasnya | [matriks kemampuan](BPMN-CAPABILITY-MATRIX.md) |
| Cara kerja metode | [metodologi](METHODOLOGY.md) |
| Struktur backend/frontend | [arsitektur](ARCHITECTURE.md) |
| Kesiapan model lab dan source connections | [pre-LLM readiness](PRE-LLM-READINESS.md) |
| Kesiapan agent sebelum konek lab | [kontrak dan verifikasi offline](AGENT-INTEGRATION-READINESS.md) |
| Membaca seluruh repo dengan Qwen | [Full repository review](FULL-REPOSITORY-REVIEW.md) |
| Alur Workspace, Mapping dan Planning | [perapian UI/UX terbaru](WORKSPACE-UX-2026-11-02.md) |
| Review plan Qwen inventory terbaru | [temuan dan perbaikan source evidence](QWEN-PLAN-REVIEW-2026-11-02.md) |
| Evaluasi dan keterbatasan acuan | [evaluasi](EVALUATION.md) |
| Eksperimen terbaru dan lama | [peta eksperimen](../experiments/README.md) |
| Keputusan dan pekerjaan tersisa | [PLAN](../PLAN.md) |

`ground-truth/` menyimpan acuan evaluasi; jangan dijadikan input retrieval.
Laporan audit dan catatan bertanggal tetap disimpan untuk penelusuran keputusan.
Dokumen lama tidak otomatis menjelaskan perilaku aplikasi versi sekarang.

- [Local BPMN library](BPMN-LIBRARY.md): saving and reusing versioned diagrams.
- [Role convention recheck](ROLE-BPMN-AUDIT-2026-11-02.md): strict gateway edge findings.
- [Prepared gateway cleanup](ROLE-BPMN-GATEWAY-CLEANUP-2026-11-02.md): user-approved canonical update and post-apply verification.
- [Sidebar, Overview and result UX](WORKSPACE-REDESIGN-2026-11-02.md).
- [Handoff GPT/Claude](../collab/CLAUDE-GPT.md): konteks TA, kedua repo, commands dan protokol update.

- [Audit paket agent tanpa VPN](OFFLINE-AGENT-PAYLOAD-AUDIT-2026-11-09.md): pemeriksaan 21 variasi transport dan batas klaim pengujiannya.
