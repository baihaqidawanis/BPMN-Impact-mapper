# Skenario Skip, Redo, Switch dan Non-Free Choice pada Simple-ERP

Setiap folder berisi salinan BPMN Level 3 dari Simple-ERP sebagai `before.bpmn` dan rancangan perilaku yang diusulkan sebagai `after.bpmn`. Repository untuk mapping adalah `C:\Simple-ERP`. Uji satu pasangan per analisis dengan **Hybrid**, **Top 10**, dan konfigurasi model yang sama. Untuk putaran awal di Planning, pilih **All changes** dan **Ultra Light** agar rangkaian lintas aksi ikut direview. Setelah perubahan diagram atau instruksi agent, jalankan mapping baru dan mulai sesi review baru.

## Skip

[Before](skip/before.bpmn) · [After](skip/after.bpmn)

[Ground truth keputusan Skip](skip/reference.json) · [Bukti source dan kriteria penerimaan](skip/reference-notes.md)

Setelah validasi DRAFT dan versi, `CheckZeroDelta` memeriksa selisih setiap baris. Jika seluruh baris nol, `CloseWithoutAdjustment` menyimpan `NO_ADJUSTMENT`, versi baru, dan audit dalam transaksi submit sebelum respons sukses. Jalur ini melewati antrean approval. Posting ledger tetap aksi terpisah. Satu baris nonnol, termasuk pasangan selisih berlawanan, masuk `SUBMITTED`. Versi usang gagal tanpa mutasi. Plan perlu menjelaskan migrasi maju yang mengizinkan status baru.

## Redo

[Before](redo/before.bpmn) · [After](redo/after.bpmn)

Reviewer yang berwenang dapat memilih **Request revision** dengan catatan. Sistem menyimpan `REVISION_REQUIRED`, catatan, audit, dan versi baru sebelum menunggu Staff. Pada request berikutnya, pembuat memperbaiki serta mengirim ulang PR ber-ID sama ke `SUBMITTED`. Panah `BackToReview` kembali ke antrean review. Reject tetap terminal. Uji penolakan permintaan revisi tanpa catatan, pengiriman ulang oleh selain pembuat, dan versi usang. Operasi yang gagal tidak mengubah status atau versi.

## Switch

[Before](switch/before.bpmn) · [After](switch/after.bpmn)

Pada approve yang lolos validasi review, backend membaca status tiap produk sebelum menulis hasil. Satu produk nonaktif mengalihkan ke jalur `REJECTED` dengan alasan otomatis, sementara reject manual tetap memakai alasan reviewer. Audit mencerminkan hasil penolakan ketika cabang beralih. Uji approve biasa, approve yang dialihkan, reject manual, dan validasi atau pembacaan produk yang gagal tanpa commit.

## XOR–AND–Stack (fixture historis)

[Before](xor-and-stack/before.bpmn) · [After](xor-and-stack/after.bpmn)

Saat Supervisor menyetujui PR, XOR memilih jalur standar atau bernilai tinggi. Jalur tinggi membuka dua pemeriksaan independen, `CheckBudget` dan `CheckDocuments`. AND join menunggu keduanya. Jika keduanya lolos, sistem menyimpan persetujuan tingkat pertama dan membuka antrean Super Admin. PR baru menjadi `APPROVED` setelah persetujuan kedua. Jika pemeriksaan gagal, PR tetap di statusnya. Transaksi dan lock review pertama berakhir sebelum menunggu keputusan berikutnya. Uji jalur standar, kedua jenis kegagalan pemeriksaan, dan penolakan pada tahap kedua. Cek transisi purchasing, penyimpanan tahap approval, UI masing-masing peran, dan tes integrasi.

## Non-Free Choice

[Before](non-free-choice/before.bpmn) · [After](non-free-choice/after.bpmn)

`LockApprovalRoute` menetapkan `STANDARD` untuk total PR di bawah ambang konfigurasi dan `ESCALATED` mulai dari nilai ambang itu. Simpan rute bersama status `SUBMITTED` untuk PR yang sama. Pada aksi approve atau reject yang sudah ada, `LoadLockedApprovalRoute` membaca rute tersebut sebelum mutasi. Dalam skenario ini, STANDARD memberi hak review kepada Supervisor dan ESCALATED kepada Super Admin. Status, versi, dan maker-checker tetap diperiksa. Uji total tepat pada ambang, perubahan ambang setelah submit, role yang keliru, dan PR lama tanpa rute valid. Before mencakup submit. After juga memperlihatkan titik penerapan aturan baru pada command review yang sudah ada.

## Representasi invisible task

After memakai nama dan tipe `serviceTask` agar perubahan langkah otomatis bisa ditelusuri. Dalam proyeksi event yang mencatat hasil bisnis, Skip melewati antrean approval, Redo kembali ke review setelah resubmit, dan Switch mengalihkan keputusan awal ke penolakan. NFC menguji ketergantungan pada rute yang sebelumnya disimpan. Sifat invisible pada ketiga pola prime bergantung pada event yang dipilih dalam proyeksi penelitian. Proyeksi belum diverifikasi terhadap event log aktual Simple-ERP.

Acuan file Skip mencakup satu gateway. Untuk Redo, Switch, dan NFC, kriteria di atas dipakai menilai isi plan. Bekukan acuan file dan fungsi secara independen sebelum menghitung precision dan recall. Untuk alur lintas aksi, satu diagram mencakup beberapa request aplikasi.

Generator dapat dijalankan kembali dari root repository.

```powershell
python -B experiments/tools/build_poc_pattern_cases.py --repo C:/Simple-ERP --output experiments/simple-erp/poc-patterns
go run ./cmd/bpmn-check --path experiments/simple-erp/poc-patterns --research-convention
```
