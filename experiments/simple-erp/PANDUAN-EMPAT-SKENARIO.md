# Panduan empat skenario Staff Simple-ERP

**Baseline empat kasus ini dibekukan.** BPMN Staff kanonis di Simple-ERP sudah
dirapikan menjadi satu start/end. Gunakan file baseline historis pada isian di
bawah untuk mutasi lama; jangan mencampurnya dengan baseline kanonis terbaru.
Detail: [perapian Staff](../../docs/STAFF-BPMN-NORMALIZATION.md).

## Catatan lanjutan setelah presentasi

Hybrid produksi memakai dua pencarian utama: BM25 dan Semantic dengan konteks
BPMN, lalu RRF menggabungkan posisi ranking. RRF bukan pencarian ketiga.
ID, path, dan nama function merupakan bukti pendukung, bukan bobot tambahan.

Top candidates masih batas keluaran yang dipilih pengguna, bukan perkiraan
otomatis jumlah file wajib berubah. Lanjutan: audit false positive dan uji
penyaringan relevansi tanpa ambang arbitrer, menjaga Recall sambil memperbaiki
Precision/MAP pada beberapa kasus. Qwen dapat meninjau kandidat kemudian;
kenaikan kualitas harus diukur sebelum/sesudah, bukan diasumsikan.

Contoh report inventory `1f7e0e4fb37f419cd0c1c925ed7878f0`: tiga file acuan
berada di posisi 1, 3, 5; P@5=0.6, R@5=1, AP=0.7556, RR=1,
R-Precision=2/3. Ini hasil satu perubahan utama, bukan rata-rata empat skenario.

MAP merata-ratakan AP antar kasus; AP memperhitungkan Precision pada setiap
posisi jawaban benar. MRR merata-ratakan 1/posisi jawaban benar pertama.
R-Precision menghitung Precision pada R hasil teratas, dengan R jumlah file acuan.
Tabel Saved test results di Research merata-ratakan empat kasus positif;
kontrol negatif dinilai terpisah. Tabel tersebut bukan evaluasi otomatis report
terbaru atau input pengguna umum.

## Isian bersama di web

- Repository path: `C:\Simple-ERP`
- Repository name: `Simple-ERP`
- Revision: boleh kosong; menganalisis file lokal saat ini, bukan checkout commit.
- Mode: Product.
- Method: Hybrid / BPMN-Context-Aware Hybrid Retrieval.
- Top candidates: `10` agar sama dengan konfigurasi pengujian tersimpan.
- Compare mapping methods: nonaktif untuk demo satu metode. Untuk perbandingan,
  pilih Research pada form dan aktifkan perbandingan dengan pasangan input sama.

Jalankan satu pasangan per analisis. Jangan memuat synthetic demo karena data
di sini memakai repository Simple-ERP. Ground truth bukan input retrieval.

## 1. Ubah aturan harga purchase request

Perubahan: harga harus lebih dari nol; perubahan label/aturan task, bukan task baru.

Baseline:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\frozen-staff-baselines\role-staff-l3-pr-create.bpmn
```
New BPMN:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\staff-p2p\role-staff-l3-pr-create-positive-price.bpmn
```
Acuan: `docs/ground-truth/staff-pr-positive-price-reference.json` dan `.md`.

## 2. Tambah task alasan inventory adjustment

Perubahan: tambah user task "Isi alasan penyesuaian stok" beserta penghubungnya.
Ini skenario yang disarankan untuk demo visual singkat.

Baseline:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\frozen-staff-baselines\role-staff-l3-adj-submit.bpmn
```
New BPMN:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\staff-inventory\role-staff-l3-adj-submit-reason-task.bpmn
```
Acuan: `docs/ground-truth/staff-inventory-submit-reason-task-reference.json` dan `.md`.

## 3. Tambah task konfirmasi total purchase request

Perubahan: tambah langkah konfirmasi total pada antarmuka pengguna.

Baseline:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\frozen-staff-baselines\role-staff-l3-pr-create.bpmn
```
New BPMN (gunakan v2):
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\staff-p2p\role-staff-l3-pr-create-confirm-total-v2.bpmn
```
Acuan: `docs/ground-truth/staff-p2p-total-confirmation-reference.json` dan `.md`.

## 4. Ubah kondisi gateway inventory adjustment

Perubahan: validasi selisih stok tidak boleh nol. Ini mengubah kondisi gateway
yang sudah ada, bukan menambahkan gateway.

Baseline:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\frozen-staff-baselines\role-staff-l3-adj-submit.bpmn
```
New BPMN:
```text
C:\BPMN-Impact-Mapper\experiments\simple-erp\staff-inventory\role-staff-l3-adj-submit-nonzero-delta.bpmn
```
Acuan: `docs/ground-truth/staff-inventory-nonzero-delta-reference.json` dan `.md`.

## Cara membaca dan mengevaluasi

Setelah Run impact analysis, lihat perubahan BPMN, ranking file, dan bukti source.
Unduh JSON sebelum mengganti skenario. Skor BM25/Semantic/RRF bukan probabilitas
file harus berubah. Precision, Recall, AP/MAP dan MRR memerlukan file acuan.
Research menampilkan hasil pengujian tersimpan; itu bukan otomatis evaluasi
analisis baru. File lokal yang berubah dapat menghasilkan ranking berbeda.

Semua acuan ini sementara, disusun berdasarkan pemeriksaan source oleh satu
assistant; belum dibuktikan melalui implementasi perubahan nyata atau validasi
independen. Jangan menganggap file di luar acuan pasti tidak relevan.

## Cakupan tambahan yang disarankan (belum dibuat dalam panduan ini)

Update: paket pasangan lanjutan sekarang tersedia terpisah di
[`structural-staff/README.md`](structural-staff/README.md): hapus task, tambah
XOR, hapus gateway dengan perubahan bisnis, kontrol hapus gateway visual,
perubahan role, AND dan OR. Itu belum menambah jumlah skenario yang dievaluasi
pada tabel empat skenario di web. Acuan union file dan hash tersedia per kasus;
konversi ke evaluasi atomic-change masih perlu dikerjakan sebelum benchmark baru.

### Checklist gateway dan rencana perubahan

- [ ] XOR: satu cabang dipilih berdasarkan kondisi; tuliskan kondisi dan jalur default.
- [ ] AND/parallel: semua cabang dijalankan; tentukan titik sinkronisasi dan kebutuhan menunggu hasil cabang.
- [ ] OR/inclusive: satu atau beberapa cabang aktif; jelaskan pemilihan dan penggabungan cabang aktif.
- [ ] Untuk tiap jenis: periksa tipe XML, diff, konteks, kandidat file, rencana perubahan dan test secara terpisah.
- [ ] Pasangan sebelum/sesudah harus memiliki diagram lengkap dan acuan sebelum melihat prediksi.

Ini backlog, bukan klaim validasi lengkap perilaku gateway. Event-based gateway
adalah pengujian lanjutan tersendiri. Rencana Markdown nantinya memuat kebutuhan
bisnis, file dan bukti, perubahan perilaku per layer, test dan ketidakpastian.
Task/gateway baru tidak otomatis membutuhkan endpoint atau migration baru.

Prioritas berikutnya untuk kasus Simple-ERP dengan acuan source:

1. Hapus task: sambungkan kembali alur, jelaskan kewajiban bisnis yang dihapus.
2. Tambah gateway: buat keputusan dan cabang dengan kondisi serta tujuan jelas.
3. Hapus gateway: tentukan cabang/aturan mana yang dipertahankan atau dihapus.
4. Pindah task ke lane/role lain: uji perubahan tanggung jawab dan otorisasi.
5. Kontrol negatif: perubahan tata letak atau label tampilan tanpa perubahan
   kewajiban bisnis seharusnya tidak menuntut perubahan source.

Tambah lane kosong berbeda dari perubahan tanggung jawab task. Taxonomy saat ini
memiliki MODIFY_ACTOR_ROLE, bukan ADD_LANE. Jangan mengklaim tambah lane sebagai
jenis perubahan khusus yang telah didukung/dievaluasi. Jika task dipindahkan ke
lane baru, periksa bahwa parser/diff mendeteksi perubahan aktor, lalu evaluasi
mapping-nya. Perubahan visual saja tidak otomatis berdampak ke source.

Daftar tersebut melengkapi variasi struktural, bukan seluruh kemungkinan BPMN.
Data requirement, state/outcome, dan perubahan ambigu dapat menjadi perluasan
berikutnya. Untuk setiap kasus baru: tetapkan kebutuhan bisnis, periksa baseline
source, susun acuan sebelum melihat prediksi, lalu uji semua metode pada input
yang sama. Uji deteksi diff dan ketepatan mapping sebagai dua hal berbeda.

Fixture sintetis tambah gateway/hapus task sudah terpisah dari empat kasus Staff
ini; keberhasilan fixture tidak membuktikan akurasi pada source Simple-ERP.
