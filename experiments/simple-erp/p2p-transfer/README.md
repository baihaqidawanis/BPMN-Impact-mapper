# P2P submit checks: transfer dan seleksi kandidat

Eksperimen offline tambahan, belum menjadi perilaku web produksi. Simple-ERP
dibaca saja. Acuan disusun dari requirement dan source sebelum inference, tetapi
masih berupa review satu AI, bukan ground truth independen.

## Perubahan proses

Baseline merupakan salinan Staff P2P L3 PR-submit. Sebelum validasi submit lama,
tambahkan dua pemeriksaan: produk pada baris PR masih aktif dan setiap estimasi
harga lebih besar dari nol. AND split/join menggabungkan hasilnya; activity
mengumpulkan hasil, kemudian XOR menentukan lanjut atau gagal. Kegagalan tidak
mengubah DRAFT. AND adalah kewajiban kedua pemeriksaan, bukan kewajiban goroutine.

`checks/before.bpmn` dan `checks/after.bpmn` memiliki diagram DI lengkap. Ada enam
node baru, beserta perubahan flow. `unchanged-control/` tidak memiliki perubahan.
Manifest mengikat hash BPMN/acuan; `source-review.json` mengikat snapshot source.

Untuk mencoba di web: repository path menunjuk checkout Simple-ERP lokal,
repository name `Simple-ERP`, revision boleh kosong untuk kondisi working tree.
Upload pasangan `checks/before.bpmn` dan `checks/after.bpmn`, pilih Product/Hybrid,
top candidates 10. **Web memakai konteks produksi; hasil eksperimen multi-hop4
dan seleksi berikut hanya tersedia melalui CLI.** Label revision manifest adalah
nama snapshot eksperimen, bukan bukti git checkout tertentu.

## Acuan dan batas evaluasi

Dua file existing yang diperkirakan perlu diubah:

- `backend/internal/purchasing/adapter/postgres/store.go`: revalidasi line/master
  saat submit di transaksi existing, tetap menjaga actor/state/version.
- `backend/test/integration/operations_test.go`: uji kedua kegagalan, gabungan,
  submit valid dan guard existing.

Kedua file diberi label pada semua enam node baru. Total 12 pasangan node-file,
tetapi hanya dua file unik. Route/form/schema baru tidak menjadi requirement.
File test yang dipilih bersifat provisional; alternatif organisasi test mungkin
valid. Flow-only changes dikeluarkan dari evaluasi node, bukan dianggap negatif.
Kontrol tidak berubah harus menghasilkan nol changes/candidates.

## Hasil ranking

| Varian | Precision@5 | Recall@5 | MAP |
|---|---:|---:|---:|
| Konteks produksi (`full`) | 36,67% | 91,67% | 0,9127 |
| Konteks baseline sampai 4 langkah | 40,00% | 100% | 0,9444 |

Delapan analisis (dua kasus, dua varian, dua pengulangan) menghasilkan prediction
dan evaluation identik per pengulangan. Semua empat BPMN lolos checker dan empat
tes import canvas. Diagram after juga diperiksa lewat screenshot browser.

## Seleksi otomatis eksperimental

`internal/selection` hanya membaca posisi kedua daftar retrieval. Tidak membaca
ground truth atau menentukan perubahan source. Pilihan diuji pada data Inventory
terdahulu: tanpa filter, atau kandidat muncul pada posisi <=5/10/20 di kedua
daftar. Pilih precision tertinggi yang mempertahankan seluruh hit top-10 awal.
Policy dikunci sebelum evaluasi P2P; tidak disetel ulang dari hasil P2P.

- `full`: semua filter yang diuji kehilangan hit development, sehingga tetap
  tanpa filter. Hasil negatif ini dipertahankan.
- `multi-hop4`: batas posisi 20 di kedua daftar. Pada P2P, 60 pasangan kandidat
  menjadi 51, semua 12 hit tetap ada. Precision aktual 20% -> 23,53%; recall 100%.
  Jumlah per node berubah 7-10, bukan selalu 10. File unik turun 19 -> 15.

Precision aktual tersebut berbeda dari Precision@5 pada tabel ranking. Kandidat
tertunda tetap disimpan beserta alasannya; action/score asli tidak diubah. Nol
kandidat juga diperbolehkan. Rank agreement bukan probabilitas kepastian.

Belum diaktifkan di web/default: precision masih rendah, enam node saling
berkaitan, acuan provisional, dan cakupan proses/arsitektur sangat terbatas.
Perlu skenario independen serta audit false positive sebelum adopsi umum.

## Reproduksi

Jalankan dari root Mapper, runtime embedding lokal sudah tersedia. Gunakan output
directory baru untuk run study. Jangan menjalankan builder untuk menimpa label
setelah melihat prediction.

```powershell
go run ./cmd/bpmn-impact-selection-study --calibrate-only --output outputs/selection-calibration
go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/p2p-transfer/manifest.json --output outputs/p2p-transfer-study --variants full,multi-hop4 --node-scope --timeout 1h
python experiments/simple-erp/summarize_baseline_context.py outputs/p2p-transfer-study
go run ./cmd/bpmn-impact-selection-study --policy outputs/selection-calibration/policy.json --output outputs/selection-p2p-validation-final
go run ./cmd/bpmn-impact-selection-study --policy outputs/selection-calibration/policy.json --output outputs/selection-p2p-validation-repeat
python experiments/simple-erp/summarize_selection_study.py outputs/selection-calibration/policy.json outputs/selection-p2p-validation-final outputs/selection-p2p-validation-repeat
```

Kalibrasi menggunakan `outputs/baseline-context-study` dari eksperimen Inventory
terdahulu. Output mesin berada di `outputs/` (di-ignore), termasuk `policy.json`,
`validation.json`, `validation-bindings.json`, `selection-summary.md` dan study
`comparison.md`. Saat mereproduksi urutan ilmiah, kunci policy setelah data
development tersedia dan sebelum membuka acuan/prediction transfer P2P.
