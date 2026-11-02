# Evaluasi satu kasus inventory dan pencarian repo

Sesi tersimpan: `d0897ce0da2185870b3911defa4cc334`. Acuan tersembunyi untuk
evaluasi: `experiments/simple-erp/current-staff/add-task/reference.json`.
Acuan tersebut memuat tiga file dan berstatus provisional source-derived;
bukan ground truth independen dari perubahan implementasi nyata.

Hanya ADD_TASK / ReasonReview dievaluasi. Dua perubahan sequence flow tidak
dimasukkan sebagai positif atau negatif. Acuan tidak dikirim ke agent.

| Daftar file | Precision | Recall |
|---|---:|---:|
| Hybrid top 3 | 100% | 100% |
| Hybrid top 5 | 60% | 100% |
| Hybrid top 10 | 30% | 100% |
| File RELEVANT dari Qwen (3 file) | 100% | 100% |

Qwen mempertahankan persis tiga kandidat awal: nol tambahan dan nol penghapusan.
Maka belum ada kenaikan terhadap Hybrid top 3. Membandingkan accepted-set
precision Qwen dengan Precision@10 Hybrid tanpa menjelaskan jumlah keluaran
akan menyesatkan. Pemilihan file yang cocok juga tidak membuktikan rencana
perubahannya benar; plan sesi ini masih NEEDS_REVIEW dan memiliki kesalahan
yang dibahas di QWEN-PLAN-REVIEW-2026-11-02.md.

Sesi lama melakukan delapan source lookup tetapi nol search_source lintas repo.
Ia belum menunjukkan kemampuan menemukan file yang terlewat Hybrid.

## Perbaikan

Kontrak analyst Ollama sekarang meminta satu pencarian repo independen sebelum
keputusan akhir, selama budget tool tersedia. Setelah itu ia memeriksa kandidat
dan dapat mengubah query atau melanjutkan halaman search menggunakan offset.
Pencarian mengunjungi semua artifact dalam snapshot repo yang diizinkan ingestion,
bukan hanya file Hybrid. Setiap halaman memuat maksimal tiga file berbeda;
fungsi-fungsi dalam satu file tidak lagi menghabiskan seluruh halaman.

Agent tetap boleh menolak hasil search dan menyatakan bukti belum cukup. Ia tidak
dianggap membaca setiap file secara penuh. Budget dan batas penemuan tetap berlaku
(default delapan tool, maksimal tiga kandidat baru per perubahan). Tidak ada
klaim cakupan exhaustif. Secret, laporan evaluasi dan dokumen leakage dikecualikan.

Tes offline mencakup pencarian lima file dalam dua halaman, sumber di luar
kandidat awal, alur HTTP search-inspect-plan-review, dan penolakan offset negatif.
Belum ada uji Qwen lab ulang atas kebijakan pencarian baru.

## Mengulang evaluasi tanpa model

```powershell
python experiments/evaluate_agent_case.py --report .data/analyses/fb60e3d69eaffe113771ec45fd9338e3/impact-report.json --session .data/agent-sessions/d0897ce0da2185870b3911defa4cc334.json --reference experiments/simple-erp/current-staff/add-task/reference.json --element-id ReasonReview --output outputs/agent-case-evaluation/result.json
```

Script hanya membaca keluaran inference dan acuan setelah proses selesai.
Evaluasi file tidak menilai action MODIFY versus VERIFY: kontrak agent saat ini
mempertahankan suggested action dan tidak boleh mempromosikan VERIFY ke MODIFY.
Skor MAP/MRR Qwen tidak dihitung karena keputusan RELEVANT bukan ranking baru.
