# Hasil pengembangan Staff ternormalisasi

Run `b1142fe45846f7f21e3d279d420df29b`. Empat kasus positif dengan satu anchor
berlabel per kasus, satu kontrol identik, empat metode, dua pengulangan, top 10.
20 sel metode/kasus dan 40 analisis; semua prediksi identik byte antar pengulangan.
Hash prediksi/evaluasi serta kesamaan fingerprint antar metode diperiksa oleh
`summarize_current_staff.py`.

| Metode | Precision@5 | Recall@5 | Recall@10 | MAP | MRR | R-precision |
|---|---:|---:|---:|---:|---:|---:|
| BM25 | 35,0% | 58,34% | 75,0% | 0,3084 | 0,2500 | 0,2500 |
| Semantic | 45,0% | 83,34% | 100,0% | 0,6586 | 0,8750 | 0,6250 |
| BM25 + Semantic / Fusion | 55,0% | 100,0% | 100,0% | 0,9028 | 1,0000 | 0,8334 |
| Hybrid dengan konteks BPMN | 55,0% | 100,0% | 100,0% | 0,8472 | 0,8750 | 0,7917 |

Ini rata-rata diagnostik empat anchor yang terpilih, bukan agregat seluruh
perubahan dalam diagram. Angka berasal dari metrik per kasus yang sudah dibulatkan
engine. Empat kontrol metode memiliki nol perubahan dan kandidat.

## Temuan

- AND: BM25 tidak menemukan kedua file acuan di top 10. Semantic dan kedua fusion
  menemukan keduanya di top 5. Hybrid AP 0,75; Fusion AP 1.
- Tambah task alasan: Hybrid dan Fusion AP 1, semua tiga file acuan pada tiga
  posisi pertama.
- XOR alasan: Hybrid Recall@5 1, AP 0,6389. Fusion AP 0,8056.
- Gabungan: Hybrid AP 1 untuk anchor ReasonDecision; Fusion AP 0,8056. Hasil ini
  belum membuktikan akurasi setiap task/gateway baru di diagram gabungan.

Fusion memiliki MAP lebih tinggi dalam paket ini. Tidak ada perubahan default,
bobot atau reference label untuk membuat Hybrid terlihat selalu menang.
Kasus AND BM25 menunjukkan keterbatasan pencarian kata; kandidat UI/master-data
mendahului inventory store dan integration test. Penilaian file tidak membuktikan
function yang ditampilkan benar; misalnya Fusion bisa memilih function berbeda
dalam file store yang sama.

## Batas dan langkah berikutnya

Acuan dibuat sebelum inference dari inspeksi source oleh satu reviewer AI.
Belum ada implementasi nyata atau review independen. Dua kasus alasan merupakan
variasi representasi satu kebutuhan. Paket ini development set yang terekspos,
bukan bukti generalisasi atau pengujian independen. Action classification juga
tidak disamakan dengan relevansi file: VERIFY tetap bisa wajar saat perubahan
implementasi belum dibuktikan oleh mapper.

Berikutnya perlu label semua perubahan primer gabungan dan variasi kebutuhan
lain sebelum memilih default berdasarkan perbedaan kecil MAP. Tidak menyimpulkan
perlu mengganti embedding atau menambah metode dari empat anchor ini saja.

Artefak lengkap: `outputs/current-staff-benchmark-final/`. Ringkasan terverifikasi:
`selected-primary-summary.json` dan `.md`. Tabel historis di web tetap memakai
hasil run historis, bukan angka baru ini.

Percobaan awal berhenti saat preflight karena penulisan JSON Windows CRLF membuat
hash acuan tidak cocok. Penulisan byte UTF-8/LF diperbaiki sebelum inference;
isi label tidak diubah. Run final memakai folder baru dan selesai penuh.
