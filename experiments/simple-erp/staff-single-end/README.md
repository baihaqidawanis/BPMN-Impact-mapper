# Staff: satu start/end dan perubahan XOR + AND + task

Paket ini berada di Mapper. Simple-ERP tidak diubah. Bukan pengganti otomatis
baseline tervalidasi atau empat kasus benchmark lama.

## File dan isian web

| Isian | Nilai |
|---|---|
| Repository path | Path lokal Simple-ERP, biasanya `C:\Simple-ERP` |
| Repository name | Simple-ERP |
| Revision | Kosong untuk menganalisis working tree sekarang |
| Baseline BPMN | `before.bpmn` dari folder ini |
| New BPMN | `after.bpmn` dari folder ini |
| Mode | Product / Hybrid |
| Top candidates | 10 |

`original.bpmn` adalah salinan baseline Staff L3 submit adjustment dari ERP.
Untuk uji perubahan bisnis, jangan membandingkan original dengan after: itu
akan mencampur perapian end event dengan penambahan aturan bisnis.

`before.bpmn` adalah contoh perapian source-reviewed: satu start, satu end,
XOR merge, lalu aktivitas `Kirim respons hasil aksi (sukses atau gagal)`.
Hasil sukses/gagal tetap tercantum pada cabang masuk. Jalur gagal tidak melewati
perubahan status atau commit pada jalur sukses. Ini abstraksi pengiriman
respons HTTP, bukan menambah operasi baru pada ERP.

Bukti source: `backend/internal/inventory/adapter/http/handler.go`, `Handler.action`
memanggil `respond` dengan status 409 ketika service gagal, atau 200 ketika
berhasil. `backend/internal/inventory/adapter/postgres/store.go`,
`TransitionAdjustment` mengembalikan error sebelum commit ketika guard gagal.
Hash source dan BPMN ada di `provenance.json`. Ini inspeksi statis, bukan
pembuktian kesetaraan seluruh jalur runtime atau seluruh role.

## Perubahan pada after

1. XOR: alasan wajib diisi; jika kosong, siapkan respons penolakan.
2. AND split: cek ulang warehouse aktif dan seluruh produk aktif.
3. AND join: kedua pemeriksaan mengembalikan hasil sebelum dilanjutkan.
4. Activity setelah join: gabungkan hasil pemeriksaan tanpa mengubah stok/status.
5. XOR: kedua hasil valid -> validasi/submit lama; tidak valid -> respons penolakan.
6. Semua hasil alternatif bergabung melalui XOR merge, kirim respons, satu end.

Ada **empat gateway baru dan lima task baru**. XOR lama tetap ada. AND bukan
instruksi untuk menambah goroutine: requirement-nya kedua pemeriksaan wajib
selesai. Kesalahan pemeriksaan dalam skenario ini menjadi hasil gagal yang
dikumpulkan, bukan meninggalkan salah satu cabang join menunggu selamanya.

Tidak ada ground truth terukur untuk gabungan ini. `source_hashes` adalah bukti
inspeksi baseline, bukan daftar jawaban file yang harus berubah. Jangan pakai
angka Precision/Recall dari empat kasus lama untuk mengklaim akurasi kasus ini.

## Aturan dosen dan batas semantik

- Satu start/end diterapkan per scope proses/subprocess, bukan satu untuk seluruh
  collaboration yang mungkin berisi banyak peserta.
- Activity setelah join adalah konvensi penelitian, bukan kewajiban umum BPMN.
- Jangan menambah task `Selesai` yang tidak melakukan apa-apa hanya agar lolos.
- XOR merge untuk jalur alternatif; AND join menunggu semua cabang wajib.
- Jangan mengubah error/terminate end menjadi end biasa secara otomatis.
- Perapian harus diterapkan pada kedua sisi pasangan sebelum mutasi bisnis.
- Konvensi ini diperiksa oleh CLI terpisah, belum menjadi tombol pemeriksa di web.
  Tidak mengubah ranking, bobot, atau hasil benchmark tersimpan.

Referensi primer:
[XOR merge](https://docs.camunda.io/docs/components/modeler/bpmn/exclusive-gateways/),
[AND split/join](https://docs.camunda.io/docs/components/modeler/bpmn/parallel-gateways/),
[none end events](https://unsupported.docs.camunda.io/8.2/docs/components/modeler/bpmn/none-events/).

## Pemeriksaan

Dari root Mapper:

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-check --path experiments/simple-erp/staff-single-end --research-convention
```

Tanpa `--research-convention`, CLI hanya memberi temuan struktur dan hal yang
perlu direview. Exit 1 untuk kegagalan pembacaan/parser; temuan konvensi tidak
menolak BPMN. Pemeriksa bukan simulator token atau verifier deadlock penuh.

Audit read-only Staff pada `staff-audit.json`: 32 file dapat diparse, 23 temuan
SINGLE_END, 5 temuan ACTIVITY_AFTER_JOIN, tidak ada SINGLE_START. Jumlah temuan
bukan jumlah bug software. Lima diagram join yang perlu ditinjau:

- `inventory/level-1/role-staff-l1-inventory.bpmn`
- `inventory/level-2/role-staff-l2-inv-monitor.bpmn`
- `inventory/level-2/role-staff-l2-inv-sources.bpmn`
- `level-0/role-staff-l0.bpmn`
- `p2p/level-2/role-staff-l2-correction.bpmn`

Diagram lain belum dinormalisasi massal: aktivitas bersama di level 0/1/2 harus
sesuai abstraksi level tersebut, bukan otomatis memakai respons HTTP L3.

Tes Go: konvensi terpisah dari validitas, scope subprocess, join, deterministic
warnings, seluruh primary changes gabungan, tidak ada jalur dari penolakan baru
ke state update sukses, dan self-diff kosong. Tes browser lokal: before/after
terimpor tanpa warning, semua shape tersedia dan semua edge punya panjang.
Tes ini tidak membuktikan ketepatan retrieval atau kesetaraan perilaku ERP.
