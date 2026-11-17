# Perapian workspace

## Audit dan pembersihan 10 November 2026

- Dihapus: tiga executable/backup hasil build di bin yang terakhir dibuat September dan tidak mencerminkan source sekarang (sekitar 24,6 MiB). Manifest nama, ukuran dan SHA-256 ada di .data/cleanup-2026-11-10/removed-binaries.json. Build ulang melalui command README bila diperlukan.
- Dihapus: planner_probe_live_test.go, harness follow-through lama yang tumpang tindih dengan role_probe_live_test.go dan uji API saved-case terbaru. Hasil percobaan historis tetap disimpan; ini bukan penghapusan hasil riset.
- Harness lab/checkpoint dipisahkan memakai build tag labdiagnostics. Tes biasa tidak perlu mengompilasi harness khusus eksperimen. Opt-in environment tetap diwajibkan untuk tes yang menghubungi model.
- Helper filesystem dipindahkan dari analyzer.go ke internal/app/files.go tanpa mengubah API atau perilaku analisis. Facade app masih memakai I/O langsung; tidak diklaim menjadi lapisan yang sepenuhnya murni.
- Tes batas dependency diperluas ke adapter LLM, source tools dan persistence, serta larangan I/O langsung di inti domain/mapping/planning/agentjobs.
- scripts/verify-agents.ps1 kini memakai proses/memori terbatas, dependency network off, dan memulihkan environment setelah selesai.
- Diperbarui: docs/ARCHITECTURE.md agar tidak lagi mendeskripsikan API/LLM sebagai Phase-1 lama.

Dipertahankan: dua report pilihan Haqi di root, seluruh saved analysis/session di .data, benchmark, ground truth, BPMN aktif, history/freeze, cache yang masih berguna, dan dependency frontend. File tidak dihapus hanya karena tua atau tidak diimpor produksi. Tidak ada perubahan Simple-ERP/POC TA, model inference, browser/server startup, commit, atau push.

Verifikasi 10 November: go test ./..., go vet ./..., gofmt dan git diff --check lulus. Harness bertag labdiagnostics berhasil dikompilasi dengan -run '^$' tanpa inference. Script verify-agents.ps1 lulus, termasuk pemeriksaan tipe frontend dan encoding. Browser tidak dijalankan; belum ada klaim verifikasi visual atau peningkatan akurasi model.

## Riwayat susunan (2026-11-07)

- experiments/simple-erp/current-staff-v2: skenario before/after dan acuan evaluasi aktif.
- experiments/simple-erp/baseline-traceability: mapping before ke source.
- experiments/tools: pembuat fixture, audit dan tes penelitian.
- experiments/history/simple-erp: paket lama, freeze, studi dan fixture regresi.

Perapian sempat berhenti setelah pemindahan, lalu diselesaikan: referensi kode,
tes dan tautan panduan diperbarui. Snapshot JSON lama tidak ditulis ulang demi
mempertahankan identitas/hash penelitian. Reader freeze mendukung lokasi paket
lama. Instruksi historis dapat merekam path asal; gunakan panduan paket aktif
untuk percobaan baru. Manifest pemindahan beserta hash asal tetap tersedia di
outputs/workspace-cleanup-2026-11-05/moves.json.

Dua report terakhir tetap di root sesuai permintaan Haqi: impact-report-89a7...
dan change-plan-8351.... File agent-context lama berada di outputs/manual-inputs.
Paket aktif, acuan evaluasi, hasil unik dan seluruh perubahan pengguna dipertahankan.
Cache Python dapat dibuat ulang dan dibuang setelah verifikasi. Data .data,
embedding/cache Go serta source Simple-ERP tidak diubah.

## Verifikasi

- Seluruh tes cmd/internal Go dan go vet lulus.
- 17 tes Python tools penelitian lulus.
- TypeScript dan pemeriksaan encoding UI lulus.
- 10 tes browser: workspace, hasil tersimpan, refresh, kedua Canvas,
  laporan AI historis/null, benchmark dan unduhan evaluasi lulus.
- Paket Staff aktif: 10 BPMN, baseline hash cocok ERP, nol temuan konvensi.

Tes UI AI memakai respons simulasi, bukan model lab. Tidak ada perubahan metode,
inference baru, commit, push, atau modifikasi Simple-ERP. Hasil Qwen terdahulu
masih NEEDS_REVIEW; verifikasi ini bukan klaim kualitas model sudah meningkat.
