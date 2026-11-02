# Uji seleksi: hapus pemeriksaan dan ubah kondisi Copy PR

Kebijakan seleksi dari eksperimen Inventory dikunci dan dipakai ulang tanpa
kalibrasi baru. Ini diagnostik development dengan acuan satu review AI dari
source, bukan benchmark independen. Simple-ERP tidak dieksekusi atau diubah.

## Pasangan BPMN

| Folder | Requirement hipotetis | Node perubahan utama | File acuan |
|---|---|---|---|
| `remove-check` | Copy PR REJECTED boleh memakai produk nonaktif; guard status tetap | REMOVE_TASK dan MODIFY_GATEWAY_CONDITION | purchasing store, operations integration test |
| `change-condition` | Copy menerima PR REJECTED atau CANCELLED; guard produk aktif tetap | MODIFY_GATEWAY_CONDITION | purchasing store, operations integration test, purchase-request-panel.tsx |
| `unchanged-control` | Tidak ada perubahan | Tidak ada | Tidak ada |

Setiap folder berisi `before.bpmn`, `after.bpmn`, dan `reference.json`. Baseline
berasal dari salinan `role-staff-l3-pr-copy.bpmn`: label diterjemahkan dan task
yang menggabungkan pemeriksaan status/master diperinci menjadi dua task. Ini
representasi ulang logic existing, **bukan salinan baseline kanonis tanpa edit**.
Seluruh penyuntingan ada di Mapper; baseline asli ERP tetap sama.

Untuk mencoba di web, repo path menunjuk Simple-ERP lokal, nama `Simple-ERP`,
revision boleh kosong untuk working tree saat itu. Upload before/after dari
folder yang sama, metode Product Hybrid, top candidates 10. Angka di bawah
berasal dari snapshot source yang hash-nya dicatat dalam `source-review.json`.
Snapshot berbeda dapat memberikan hasil berbeda.

Penghapusan task tidak otomatis berarti menghapus file/function. Store.CopyRequest
masih dibutuhkan; yang berubah hanya satu guard di function bersama tersebut.
Perluasan kondisi juga mengubah tombol Copy yang sebelumnya tampil hanya untuk
REJECTED. Signature service/domain/route/schema tidak wajib berubah menurut
requirement ini. Penentuan file test masih provisional.

## Hasil ranking pada top 10

| Kasus | Precision@5 | Recall@5 | MAP | Posisi file acuan |
|---|---:|---:|---:|---|
| Hapus pemeriksaan | 40% | 100% | 0,700 | store 1; integration test 5 pada kedua node |
| Ubah kondisi | 40% | 66,67% | 0,500 | store 1; integration test 4; frontend di luar top 10 |
| Kontrol | n/a | n/a | n/a | nol perubahan/kandidat |

Full dan multi-hop4 identik pada kedua kasus ini. Perluasan baseline sampai empat
langkah hanya berlaku untuk klaster node baru, sehingga memang tidak menambah
konteks untuk penghapusan/perubahan kondisi. Dua varian kali tiga kasus kali dua
pengulangan menghasilkan 12 analisis yang byte-identical per pengulangan.

## Seleksi dengan kebijakan lama

| Varian | Kandidat sebelum/sesudah | Hit sebelum/sesudah | Precision aktual | Recall aktual |
|---|---:|---:|---:|---:|
| full, tanpa filter | 30 -> 30 | 6 -> 6 | 20% -> 20% | 85,71% -> 85,71% |
| multi-hop4, kedua rank <=20 | 30 -> 27 | 6 -> 6 | 20% -> 22,22% | 85,71% -> 85,71% |

Ini agregat **pasangan node-file** dengan tujuh pasangan acuan, bukan rata-rata
Recall@5 dua kasus. Ada tiga node utama; filter menghasilkan 8, 10 dan 9 kandidat.
Tidak ada abstention atau hit yang dibuang filter. Frontend yang sudah tidak
masuk top 10 tetap terlewat: selection tidak melakukan retrieval tambahan.
Kebijakan tidak diketatkan supaya precision tampak tinggi.

## Audit kandidat tambahan

Tool `audit_selection_errors.py` menghasilkan JSON dan Markdown terikat hash:

- acuan tidak ditemukan dalam kandidat raw;
- acuan ditemukan raw tetapi hilang karena filter;
- acuan dipertahankan;
- kandidat terpilih/tertunda yang tidak tercantum dalam acuan;
- posisi ranking, indexed layer dan evidence terbatas untuk setiap kandidat
  tambahan. Evidence menjelaskan asal retrieval, bukan keputusan wajib berubah.

Pada varian filter terdapat 21 pasangan tambahan di luar acuan: 9 dokumentasi,
4 adapter, 4 persistence, 3 domain, 1 frontend. Ini **bukan 21 file unik**.
Sebutan false positive masih provisional karena label tidak independen dan
fokusnya implementasi wajib. Kontrak/dokumentasi bisa tetap relevan sebagai
pendukung. Audit ini tidak menambahkan bobot untuk membenarkan kunci jawaban.

Pemeriksaan source menemukan:

- Sales store memiliki guard copy/master yang mirip, tetapi requirement hanya
  mengubah PR. Kemiripan implementasi tidak membuat sales wajib berubah.
- Migration/schema memuat purchase request/status, tetapi tidak perlu perubahan
  struktur: CANCELLED sudah ada dan tidak ada field/constraint baru diminta.
- Domain `PurchaseRequest.Copy` membuat DRAFT tanpa guard REJECTED. Relevan
  secara konsep, tetapi guard yang berubah berada di store.
- `docs/STATE-MACHINES.md` mendeskripsikan rejected-copy. Ia sah sebagai
  dokumentasi terkait; jangan menghapus semua dokumentasi dari corpus hanya
  untuk menaikkan precision terhadap acuan kode.
- Frontend memiliki guard REJECTED dan tombol Copy di tail passage yang memang
  dibaca semantic. Teks CANCELLED belum ada di file itu. Belum ada bukti bahwa
  memperbanyak passage atau mengganti model otomatis menyelesaikan ranking.

Diagnostik terpisah top-50 menemukan frontend pada **rank 17**, BM25 rank 15,
semantic rank 25. Evidence semantic langsung menunjuk baris tombol Copy yang
dibatasi REJECTED. Sepuluh kandidat pertama identik dengan run top-10, begitu
juga source coverage serta old/new element dan konteksnya. Label revision run
diagnostik adalah `unspecified`, berbeda dari `selection-challenge-frozen`,
sehingga input fingerprint tidak identik; perbedaan ini tidak disembunyikan.
Binding hash dan pemeriksaan kesamaan tersimpan di
`outputs/selection-challenge-wide-diagnostic/diagnostic-binding.json`.

Temuan tambahan: jika daftar diperluas, aturan kedua rank <=20 justru akan
menunda frontend valid ini karena semantic rank-nya 25. Jadi mempertahankan hit
top-10 belum membuktikan filter aman pada budget kandidat lain. Hasil top-50
**tidak** menggantikan angka recall/precision top-10 pada tabel di atas.

## Keputusan dan pekerjaan berikutnya

**Tidak mengaktifkan filter otomatis di web/default.** Pengurangan kandidat kecil,
masih ada banyak tambahan, dan frontend wajib terlewat. Kebijakan menyaring hanya
hit yang sudah ditemukan, bukan membuktikan seluruh dampak berhasil dipetakan.

Langkah berikutnya adalah diagnostik ranking lintas layer (backend/frontend),
kemudian uji perbaikan umum pada development dan kasus lain. Jangan mempromosikan
metode dari satu kasus atau mengubah label setelah melihat prediction. Qwen belum
digunakan; hasil yang lemah tidak diserahkan begitu saja kepada LLM.

## Reproduksi

Output directory harus baru. Runtime embedding lokal harus tersedia; tanpa
dependency download atau network. Jalankan pekerjaan embedding, browser, dan
kompilasi secara berurutan pada mesin dengan memori terbatas.

```powershell
go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/selection-challenges/manifest.json --output outputs/selection-challenge-study --variants full,multi-hop4 --node-scope --timeout 1h
python experiments/simple-erp/summarize_baseline_context.py outputs/selection-challenge-study
go run ./cmd/bpmn-impact-selection-study --policy outputs/selection-calibration/policy.json --validation outputs/selection-challenge-study --output outputs/selection-challenge-validation
go run ./cmd/bpmn-impact-selection-study --policy outputs/selection-calibration/policy.json --validation outputs/selection-challenge-study --output outputs/selection-challenge-validation-repeat
python experiments/simple-erp/summarize_selection_study.py outputs/selection-calibration/policy.json outputs/selection-challenge-validation outputs/selection-challenge-validation-repeat
python experiments/simple-erp/audit_selection_errors.py --study outputs/selection-challenge-study --selection outputs/selection-challenge-validation --output outputs/selection-challenge-error-audit-verified
python -m unittest discover -s experiments/simple-erp -p test_selection_audit.py
```

Perintah diagnostik (bukan evaluasi budget utama):

```powershell
go run ./cmd/bpmn-impact-mapper analyze --repo C:\Simple-ERP --old-bpmn experiments/simple-erp/selection-challenges/change-condition/before.bpmn --new-bpmn experiments/simple-erp/selection-challenges/change-condition/after.bpmn --output outputs/selection-challenge-wide-diagnostic --method hybrid --top-k 50 --timeout 20m
```

Hasil mesin berada di `outputs/` yang di-ignore. `comparison.md` memuat ranking;
`selection-summary.md` seleksi; `audit.md` dan `audit.json` memuat audit. Jangan
menjalankan builder lagi untuk mengganti label setelah inference. Enam BPMN lolos
checker konvensi dan tes import canvas; tiga unit test auditor lulus.
