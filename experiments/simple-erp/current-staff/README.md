# Snapshot historis: Staff satu start/end

Untuk percobaan baru setelah perapian gateway, gunakan [current-staff-v2](../current-staff-v2/README.md). Paket ini dipertahankan bersama hasil lamanya.

**Run selesai:** [hasil dan keterbatasan evaluasi](RESULTS.md).

Kelanjutan: [evaluasi ketujuh node baru pada kasus gabungan](combined/NODE-EVALUATION.md).
Acuan ini terpisah dari label satu anchor sebelumnya dan tidak mengganti hasil historis.

Paket historis untuk mereproduksi percobaan sebelumnya. `before.bpmn` disalin dari Staff kanonis yang
sudah dinormalisasi; `after.bpmn` diturunkan dari salinan yang sama. Semua file
memiliki BPMN DI agar task dan garis muncul di Canvas/bpmn.io.

## Isian web

- Repository path: `C:\Simple-ERP` (sesuaikan lokasi laptop).
- Repository name: Simple-ERP. Revision boleh kosong untuk working tree sekarang.
- Mode Product / Hybrid. Top candidates 10.
- Baseline: `before.bpmn` dalam salah satu subfolder berikut.
- New BPMN: `after.bpmn` dari **subfolder yang sama**.
- Untuk perbandingan empat metode, aktifkan Compare mapping methods di Research.

| Folder | Perubahan kebutuhan |
|---|---|
| add-task | Staff mengisi dan memeriksa alasan sebelum submit adjustment |
| add-xor | Alasan wajib: ada -> validasi submit, kosong -> respons gagal |
| add-and | Cek ulang warehouse aktif DAN seluruh produk aktif, kumpulkan kedua hasil sebelum keputusan |
| combined | Alasan wajib melalui XOR, lalu AND pemeriksaan master, keputusan hasil dan submit |
| unchanged-control | Before/after identik; seharusnya tidak ada perubahan atau kandidat |

Kegagalan pemeriksaan tidak boleh mengubah status/stock. Pemeriksaan AND dalam
skenario ini mengembalikan hasil valid/gagal agar kedua jalur dapat selesai;
bukan instruksi wajib goroutine. State/actor/version guards lama tetap berlaku.

## Acuan dan evaluasi

`reference.json` ditulis sebelum inference dari inspeksi source ERP. Ini acuan
sementara dari satu review source, bukan perubahan implementasi nyata atau
penilaian independen. `source-review.json` menyimpan hash source dan alasan.
Jangan upload kedua file ini sebagai input mapping.

Empat kasus positif dinilai pada **satu perubahan utama terpilih per kasus**.
Kasus gabungan memakai gateway alasan sebagai anchor. Perubahan lain tetap
ditampilkan dalam report, tetapi belum dilabeli; tidak dianggap false positive
atau negatif. Skor ini tidak membuktikan seluruh perubahan gabungan tertangani.
Penilaian utama pada tingkat file; menemukan file yang benar belum membuktikan
function yang ditampilkan tepat. Klasifikasi aksi dinilai terpisah: mapper bisa
tetap memberi VERIFY meskipun acuan kebutuhan mengusulkan MODIFY. Dua kasus
alasan memakai kebutuhan yang sama dengan representasi berbeda, sehingga empat
kasus ini tidak dianggap empat proses bisnis independen.
Control identik memiliki cakupan lengkap. Hasil subset harus dibaca per kasus,
terpisah dari agregat benchmark lengkap dan dari tabel historis di web.

Run: `manifest.json`, empat metode, top 10, dua pengulangan. Canonical JSON dan
evaluasi tersimpan di `outputs/current-staff-benchmark-final`. Hasilnya dilaporkan
setelah run benar-benar selesai; tidak memperbarui tabel web secara otomatis.

Setelah selesai, validasi hash dan rangkum hasil dengan:

```powershell
python experiments/simple-erp/summarize_current_staff.py outputs/current-staff-benchmark-final
```

```powershell
$env:GOCACHE='C:\BPMN-Impact-Mapper\.gocache'; go run ./cmd/bpmn-impact-mapper benchmark --manifest experiments/simple-erp/current-staff/manifest.json --output outputs/current-staff-new-run --timeout 1h
```

Output harus folder baru. Ini pengujian pengembangan yang sudah terekspos,
bukan held-out benchmark untuk klaim generalisasi lintas repository/bahasa.
