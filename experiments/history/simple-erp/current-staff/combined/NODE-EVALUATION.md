# Evaluasi seluruh node baru pada skenario gabungan

Cakupan diperluas dari satu gateway menjadi **7 node: 4 gateway dan 3 task**.
Referensi lama `reference.json` dipertahankan. Label baru `node-reference.json`
disertai kebutuhan/alasan per node dan hash source/BPMN di
`node-reference-review.json`.

**Status:** acuan sementara dari review source oleh AI, diperluas setelah
benchmark lama tersedia. Ini evaluasi retrospektif development set yang sudah
terekspos, bukan blind/independent ground truth atau held-out validation.

## Acuan

| Node | File acuan | Alasan |
|---|---|---|
| ReasonDecision | Inventory panel, inventory store, operations integration test | UI submit mengirim alasan kosong; backend belum mewajibkan alasan pada submit |
| MasterChecksSplit | Store, integration test | Kedua pemeriksaan wajib sebelum submit; tidak berarti wajib goroutine |
| CheckWarehouse | Store, integration test | Warehouse yang aktif saat create bisa nonaktif pada waktu submit |
| CheckProducts | Store, integration test | Produk perlu diperiksa ulang saat submit |
| MasterChecksJoin | Store, integration test | Hasil kedua pemeriksaan harus tersedia sebelum perubahan status |
| CombineValidation | Store, integration test | Gabungkan hasil, gagal berarti tidak mengubah status/stok |
| MasterDecision | Store, integration test | Kedua hasil harus valid; guard DRAFT/version/akses dan rollback tetap berlaku |

Endpoint dan field Reason sudah ada. Handler tidak otomatis harus diubah; tidak
ada kebutuhan migration atau tabel baru pada skenario ini. Source aplikasi dan
test ERP tidak dimodifikasi atau dieksekusi. Tujuh label memiliki tiga file unik;
hubungan berulang di node berbeda bukan tujuh kebutuhan bisnis independen.

## Hasil pada prediksi tersimpan yang sama

| Metode | Precision@5 | Recall@5 | MAP node |
|---|---:|---:|---:|
| BM25 | 14,29% | 30,95% | 0,1946 |
| Semantic | 40,00% | 95,24% | 0,6603 |
| BM25 + Semantic | 31,43% | 71,43% | 0,5087 |
| Hybrid dengan konteks BPMN | 37,14% | 85,71% | 0,6468 |

Semua tujuh node diberi acuan. **11 perubahan sequence flow dikecualikan**, tidak
dianggap negatif atau dinyatakan tidak berdampak. Ini complete node coverage,
bukan complete BPMN change coverage. Function dan ketepatan aksi MODIFY/VERIFY
tidak disamakan dengan relevansi file.

## Gabungan file, setiap file dihitung sekali

Ambil top 5 **file unik dari setiap node**, kemudian union. Precision memakai
jumlah file union yang benar-benar dikembalikan, bukan penyebut 5.

| Metode | Acuan unik | Kandidat unik | Precision union | Recall union |
|---|---:|---:|---:|---:|
| BM25 | 3 | 22 | 13,64% | 100% |
| Semantic | 3 | 13 | 15,38% | 66,67% |
| BM25 + Semantic | 3 | 16 | 18,75% | 100% |
| Hybrid | 3 | 13 | 23,08% | 100% |

Semantic kehilangan frontend pada union top 5 karena panel berada di peringkat 8
untuk ReasonDecision. Pada Hybrid, store berada di peringkat 7 untuk
MasterChecksJoin dan CombineValidation, sehingga dua kebutuhan node belum
tercakup pada top 5 walau file yang sama ditemukan melalui node lainnya.

Poin ini menjelaskan mengapa recall union 100% belum membuktikan setiap node
dipetakan benar. Sebaliknya, membesar-besarkan daftar kandidat memperbesar recall
tetapi menambah pekerjaan review: top 10 per node Hybrid menghasilkan 23 file
unik untuk tiga acuan, precision union hanya 13,04%.

File ekstra berarti di luar daftar acuan perubahan implementasi/test yang
dinyatakan. Sebagian dokumentasi atau file pendukung mungkin tetap layak VERIFY;
label ini tidak membuktikan semua kandidat ekstra sama sekali tidak relevan.

## Verifikasi dan penggunaan

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-impact-node-eval --output outputs/compound-node-new-run
```

Output harus folder baru. Command tidak melakukan inference. Ia memverifikasi
hash source/BPMN/acuan, hubungan input dengan manifest benchmark lama, hash
prediksi/analysis ID/fingerprint, dan coverage semua node. JSON dan Markdown
ringkasan dihasilkan dari hasil evaluasi yang sama. Dua pengulangan evaluasi
menghasilkan artefak identik; ini reproduksi evaluator, bukan dua inference baru.

Artefak: `outputs/compound-node-evaluation-final/`. CLI ini khusus node scope.
Jangan upload `node-reference.json` ke evaluator web yang mensyaratkan label
seluruh perubahan atomik; guard complete coverage web sengaja tidak dilonggarkan.

Lanjutan yang dibenarkan hasil ini: uji pengayaan konteks terbatas untuk node
baru yang hanya dikelilingi node baru, khususnya join/agregasi. Bandingkan sebelum/
sesudah dengan label tetap dan skenario lain; jangan langsung mengubah bobot,
menambah top-k atau mengubah acuan agar Hybrid terlihat menang. Belum ada
perubahan ranking/model pada evaluasi ini.
