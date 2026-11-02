# Audit kandidat inventory: tambah alasan submit

Input audit: `outputs/manual-inputs/impact-report-1f7e0e4fb37f419cd0c1c925ed7878f0.json` (dipindah dari root tanpa mengubah isi).
Audit ini membaca prediksi dan acuan sesudah inference; bukan ground truth baru.
Tidak menjalankan inference ulang atau mengubah source/ranking.

## Hasil satu kasus

Tiga file acuan berada pada posisi 1, 3, 5: integration operations test,
inventory adjustment panel, inventory backend store. P@5=0.6, R@5=1,
P@10=0.3, R@10=1, AP=0.7556, RR=1, R-Precision=0.6667.
Ini bukan rata-rata empat skenario. Acuan tetap provisional source review.

## Konteks yang benar-benar tercatat

- Process: `Submit stock adjustment - Staff`.
- Before: `Kebutuhan aksi` (startEvent).
- After: `Kirim adjustment untuk review` (userTask).
- Pattern: INSERT_TASK.

Before bukan task bisnis, melainkan startEvent berlabel umum. Jangan mengklaim
bahwa konteks selalu satu task bisnis sebelum dan sesudah. Kata umum seperti
`aksi` berpotensi menambah noise; ini hipotesis untuk uji ablation, bukan bukti
bahwa satu kata tersebut menyebabkan seluruh kesalahan.

## Kandidat tambahan di luar acuan

| Posisi | File | Bukti pada report | Penilaian terbatas |
|---|---|---|---|
| 2 | frontend/app/inventory/[section]/page.tsx | Jalur inventory | Relevansi proses masuk akal; kewajiban mengubah file belum terbukti |
| 4 | frontend/features/inventory/inventory-workspace.tsx | Workspace inventory | Bisa konteks pendukung; bukan otomatis target perubahan |
| 6 | frontend/features/purchasing/purchase-order-panel.tsx | BM25 rank 5; Semantic rank 17; excerpt form PO | Dugaan noise dari pola form/aksi lintas modul |
| 7 | backend/internal/sales/adapter/postgres/store.go | BM25 `case "submit"`; Semantic SQL stock_balances | Kata submit umum dan akses stok juga ada di sales; belum membuktikan kebutuhan perubahan sales |
| 8 | backend/migrations/000005_operations.up.sql | BM25 rank 26; Semantic rank 3; schema stok | Mirip domain inventory, tetapi task alasan tidak otomatis memerlukan migration |
| 9 | frontend/features/purchasing/purchase-request-panel.tsx | BM25 rank 4, excerpt `Aksi ${name}`; Semantic rank 28 | Kata generik dapat mendapat posisi tinggi di lexical branch |
| 10 | backend/migrations/000005_operations.down.sql | DROP trigger stock_movements | Hubungan teks ke stok, tanpa bukti perubahan rollback schema |

Di luar acuan dihitung sebagai false positive dalam evaluasi tertutup ini,
tetapi bukan pembuktian bahwa file tersebut pasti tidak perlu berubah.

## Eksperimen berikutnya

Opsi internal sudah disiapkan: `retrieval.ContextBM25{OmitStartEndLabels: true}`.
Default tetap false dan opsi ini belum dihubungkan ke CLI/web. Opsi mengosongkan
label tetangga bertipe startEvent/endEvent pada salinan query saja, bukan
menghapus hubungan graph, label event yang menjadi perubahan utama, atau
task/gateway lain. Ini menyaring berdasarkan tipe XML, bukan daftar kata atau
nama modul. Ia dapat membuang label event yang bermakna juga, sehingga perlu
pengujian sebelum diaktifkan. Semantic tetap tidak berubah.

Metadata strategi mencatat `query_ablation=omit-neighbour-start-end-labels-v1`.
Harness eksperimen mendatang juga harus memasukkan opsi ini ke identitas
konfigurasi sebelum menyimpan atau membandingkan report. Belum ada klaim
kenaikan Precision/Recall pada Simple-ERP dari opsi ini.

1. Bekukan hasil dan acuan sekarang; jangan menambah acuan karena kandidat muncul.
2. Bandingkan konteks penuh dengan konteks yang tidak menyertakan label event
   generik. Jangan blacklist kata/domain/path khusus Simple-ERP.
3. Pertahankan model, snapshot source, file-level fusion, top-k, dan input lain.
4. Catat peringkat file acuan, P@5, R@5, AP dan R-Precision per kasus, lalu agregat.
5. Uji semua kasus pengembangan dan kasus baru terpisah sebelum memilih default.
6. Jika menguji penyaringan relevansi, laporkan file benar yang ikut terbuang.

Tidak menetapkan cutoff cosine/RRF dari satu kasus ini. Tidak memblokir semua
file migration, sales, atau purchasing: ketiganya dapat relevan pada perubahan
bisnis lain. Targetnya membedakan kemiripan teks dari kebutuhan perubahan,
bukan menaikkan nilai similarity atau menjadikan P@5 kasus ini melebihi batas
3/5 yang ditentukan jumlah acuannya.
