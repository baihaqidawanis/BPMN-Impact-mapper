# Konvensi BPMN tiga role

Kelanjutan 30 Oktober 2026: Supervisor dan Super Admin mengikuti konvensi Staff.
File kanonis tetap di `Simple-ERP/docs/bpm-haqi/roles/<role>/`. Perapian mengganti
file yang sama, tanpa membuat varian baru di folder role.

| Role | File diperiksa | Perubahan diterapkan |
|---|---:|---:|
| Staff | 32 | Sudah selesai pada 29 Oktober; dipertahankan |
| Supervisor | 31 | 29; dua sudah memenuhi konvensi |
| Super Admin | 49 | 49 |

Setiap proses memiliki satu start/end. Hasil alternatif dipertahankan pada flow
menuju XOR merge, diikuti aktivitas yang bermakna lalu end. Satu jalur tidak
ditambahi join yang tidak diperlukan. End umum berarti proses selesai, bukan
semua hasil dianggap sukses. AND join tidak digunakan untuk hasil eksklusif.

Pada L3, aktivitas bersama adalah pengiriman respons hasil/error oleh sistem.
Pada L0–L2, aktivitas bersama adalah role pemilik diagram memeriksa hasil/status
yang sudah tampil di UI. Aktivitas ini tidak menambah approval, commit, perubahan
state atau request API baru. Guard dan hak akses existing tetap pada model.

Verifier membandingkan ID/atribut node bisnis, kondisi flow, actor lane,
`calledElement`/namespace QName, tata letak node lama, isolasi jalur keluar dari
operasi bisnis lama, serta kelengkapan shape dan waypoint. Ini verifikasi source
dan struktur statis, bukan pembuktian seluruh perilaku runtime atau concurrency.

Hasil final: seluruh 112 file kanonis lolos parser dan checker tanpa temuan
struktur/review/konvensi. Tes browser untuk 80 diagram Supervisor/Super Admin
lulus 80/80: semua shape terimpor, semua edge memiliki panjang dan tidak ada
warning import. Hash file kanonis cocok dengan versi review. Sebanyak 120 hash
source backend Go dan BPMN Staff tetap sama. Tes Go parser/diff/checker dan
whitespace check kedua repository lulus. Tidak ada commit atau push.

Review dan backup:

- `outputs/supervisor-normalization-2026-10-30/`
- `outputs/super-admin-normalization-2026-10-30/`
- `outputs/other-roles-review-2026-10-30/`
- `outputs/all-role-generator-review-2026-10-30/`

Normalisasi memakai `experiments/simple-erp/normalize_staff_bpmn.py` dengan
`--role supervisor` atau `--role super-admin`; nama script lama dipertahankan
agar workflow Staff yang sudah ada tetap berlaku. Verifier yang sama dipakai
pada ketiga role. Generator dokumen dan helper konvensinya diperluas untuk
seluruh role; generator diuji pada salinan di Mapper, bukan menimpa diagram
kanonis dengan hasil regenerate.

Salinan generator menghasilkan 112 BPMN yang lolos konvensi; normalisasi ulang
memberi byte yang sama dan seluruh preview SVG dapat diparse. Dua script yang
diterapkan di Simple-ERP adalah `scripts/bpm-haqi/build.py` dan
`scripts/bpm-haqi/staff_convention.py`. Nama helper historis tetap dipakai untuk
kompatibilitas meskipun sekarang mencakup tiga role.

Tahap berikutnya: bekukan baseline dan susun pemetaan task-to-file/function
sebagai acuan penelitian. JSON menjadi data utama; Markdown dibuat dari data
yang sama. Pemetaan implementasi existing tidak otomatis merupakan acuan file
yang wajib berubah pada suatu skenario; kebutuhan perubahan dinilai terpisah.

## Recheck 2026-11-02

The historical pass above covers the earlier single start/end and activity-after-
join convention. It does not cover a prohibition on every gateway-to-gateway
edge. The stricter recheck found 158 such edges in 87 of 112 canonical role
diagrams. See [current audit](ROLE-BPMN-AUDIT-2026-11-02.md). No case-study files
were rewritten in this recheck.
