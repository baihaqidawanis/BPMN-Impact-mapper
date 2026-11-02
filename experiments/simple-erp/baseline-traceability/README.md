# Acuan mapping before Simple-ERP

**Paket lengkap terbaru:** [tiga modul × tiga role](all-roles/README.md).
Mencakup P2P, Inventory, Master Data/Konfigurasi untuk Staff, Supervisor dan
Super Admin: 72 proses L3, 354 aktivitas, serta indeks 40 diagram L0–L2.
Setiap aktivitas L3 memiliki hubungan ke file/function dengan bukti source.
Paket memuat JSON utama, sembilan tabel Markdown dan hash 112 BPMN serta 34
source/test files. Ini hasil review statis sementara, bukan klaim seluruh
perilaku sudah diuji 1:1 pada runtime.

Verifikasi paket baru:

```powershell
python experiments/simple-erp/build_role_traceability.py --repo C:\Simple-ERP --output experiments/simple-erp/baseline-traceability/all-roles --verify
python -m unittest discover -s experiments/simple-erp -p test_role_traceability.py -v
```

## Snapshot Staff P2P awal (dipertahankan)

Buka **[tabel mapping](staff-p2p-l3.md)** untuk membaca hubungan task ke file dan
function. Data utamanya [staff-p2p-l3.json](staff-p2p-l3.json); Markdown dirender
dari object JSON tersebut.

Cakupan: 10 proses, 50 aktivitas dari BPMN Staff P2P L3 Simple-ERP yang sudah
mengikuti satu start/end. Setiap aktivitas memiliki nama, tipe, actor lane,
file/function implementasi, peran hubungan dan bukti baris source. Sepuluh BPMN
dan 12 source/test files dibekukan melalui SHA-256; tidak menyalin seluruh repo.
Git HEAD dan keadaan dirty dicatat. Snapshot hash mendeteksi perubahan referensi.

Cara membaca:

- **Input pengguna:** form/tombol pada frontend yang mengirim kebutuhan task.
- **Validasi aturan:** pemeriksaan yang dijalankan function backend.
- **Penyimpanan data/state:** query insert/update yang menerapkan efek proses.
- **Audit dan commit:** pencatatan audit serta finalisasi transaksi.
- **Respons:** handler/helper mengirim hasil atau error ke pemanggil.
- **Rantai operasi pendukung:** API -> application service -> store; tidak
  dianggap seluruhnya direct implementation untuk setiap task.
- **RELATED_TEST_NOT_EXECUTED:** source test yang memanggil operasi terkait.
  Delapan proses memiliki hubungan test yang ditinjau. Test ERP tidak dijalankan
  dan daftar ini bukan bukti semua guard/jalur UI teruji. Cancel PR/GR belum
  ditautkan ke test khusus dalam review ini.

Event/gateway dicatat sebagai elemen kontrol. Pemetaan file tersendiri dan analisis
detail kondisi/branch belum disimpulkan untuk elemen tersebut.

## Cara memverifikasi snapshot

Dari root Mapper:

```powershell
python experiments/simple-erp/build_baseline_traceability.py --repo C:\Simple-ERP --output experiments/simple-erp/baseline-traceability --verify
```

Pemeriksaan mencocokkan ulang hash dan bukti source, 50 aktivitas, data JSON serta
hasil render Markdown. Jika gagal karena drift, tinjau perubahan source/BPMN;
jangan langsung overwrite referensi untuk menyesuaikan hasil prediksi.

Script memakai keputusan review yang eksplisit untuk memilih file/function dan
potongan kode; pencocokan string hanya menemukan posisi bukti, bukan algoritma
otomatis untuk menemukan mapping. Tidak memakai output kandidat mapper atau
menjalankan source Simple-ERP. Semua path dalam artefak relatif terhadap repo.

## Peran dalam evaluasi

Ini **baseline implementation traceability**, acuan sementara dari satu review
source oleh AI. Belum ada review independen, validasi runtime atau metrik akurasi
mapping before. Tidak ada skor confidence buatan pada hubungan-hubungan ini.

Untuk perubahan BPMN, nilai kebutuhan baru secara terpisah. Sebagai contoh,
handler yang sudah menerima field alasan bisa tetap sama meski UI/validasi perlu
berubah. Hubungan existing tidak otomatis menjadi daftar file wajib diubah.
Pemetaan after saat source after belum ada merupakan perkiraan kebutuhan.

File ini hanya untuk penelitian/evaluasi. Repository inference tetap Simple-ERP;
jangan memasukkan referensi ini ke repository yang dianalisis, configuration
retrieval atau konteks LLM penilai sebelum menghasilkan prediksi. Jika acuan
dipakai sebagai input metode, eksperimennya harus dinyatakan sebagai eksperimen
dengan mapping eksplisit dan memakai evaluasi independen.

Perluasan modul/role tersedia pada paket terbaru di atas. Review independen,
validasi runtime dan pemetaan gateway/branch secara mendalam masih terpisah
dari kelengkapan tabel aktivitas. Label required-change untuk skenario harus
disusun dari kebutuhan perubahan, bukan menyalin semua hubungan existing.
