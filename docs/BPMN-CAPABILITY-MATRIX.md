# Kemampuan BPMN dan kasus pada screenshot pilot

Audit terhadap `Screenshot 2026-10-29 143029.png`. Slide membatasi subset pilot
teman; batas proyek tersebut tidak otomatis sama dengan batas Impact Mapper.
Mapper membandingkan spesifikasi dan mencari kandidat source, bukan menjalankan
workflow. Jumlah input/output gateway tidak dipakai untuk menebak AND/XOR:
jenisnya dibaca dari tag XML BPMN.

| Elemen/kasus | Yang sudah dilakukan | Batas yang harus diperhatikan |
|---|---|---|
| Satu pool, lane manusia/sistem | Actor dibaca dari lane; perubahan role dapat dideteksi | Hak akses runtime tidak dibuktikan |
| Start/end biasa | Dibaca; perubahan hasil akhir tertentu terdeteksi | Konvensi satu start/end adalah policy penelitian, bukan syarat universal input |
| User/service/send task | Dibaca; add/remove/label change dapat dideteksi | Tipe dikenal tidak menjamin kandidat file benar |
| Sequence flow dan XOR | Endpoint, kondisi teks, label, direction/default dibaca/dibandingkan | Kondisi tidak dieksekusi atau dibuktikan benar |
| AND dan OR | Jenis gateway, penambahan/penghapusan dan topology terdeteksi; fixture tersedia | Tidak ada simulator token, pembuktian sinkronisasi atau deadlock; OR belum punya evaluasi retrieval lengkap |
| Loop berupa sequence flow kembali | Adjacency map menyimpan siklus; diff kondisi dalam siklus diuji | Tidak mensimulasikan pengulangan atau membuktikan terminasi |
| Standard-loop/multi-instance marker | File masih dapat dibaca; ada peringatan partial support | Detail jumlah instance, sequential/parallel dan completion condition belum dibandingkan |
| Subprocess biasa | Elemen internal dan parent scope dibaca | Tidak mengeksekusi atau membuktikan kesetaraan perilaku subprocess |
| Call activity | Node dibaca | calledElement tidak di-resolve/dibandingkan; model eksternal tidak dimuat. Ada warning |
| Event-based/complex gateway | Jenis node/topology dibaca | Race event/activation rules belum dimodelkan. Ada warning |
| Timer/message/boundary event | Tipe event definition, parent dan attachment tertentu dibaca | Durasi/ref/payload/cancelActivity belum dibandingkan. Ada warning |
| Event subprocess/compensation | Struktur dan beberapa tipe definition tetap terbaca | Trigger/compensation execution belum dianalisis. Ada warning |
| Multi-pool/collaboration | Beberapa process/participant dibaca dengan konteks process | Message flow/dependency komunikasi belum didiff. Ada warning |
| XOR dengan predicate JSON bertipe seperti slide | Dibaca sebagai teks conditionExpression; perubahan teks terdeteksi dalam tes bersiklus | Tidak ada interpreter, schema/type checker predicate atau evaluasi boolean |

Perubahan format di dalam JSON/expression dapat terdeteksi sebagai perubahan
teks walaupun artinya sama. Belum ada normalisasi/ekuivalensi logika expression.

## Cara membaca report baru

`BPMN_PARTIAL_SUPPORT` mencatat fitur pada baseline/changed yang belum tercakup
semantik diff. Keluaran parsial masih berguna untuk perubahan yang diketahui,
tetapi **nol perubahan terdeteksi tidak berarti proses pasti sama**. Contohnya,
penambahan multi-instance dapat belum muncul sebagai change, namun sekarang
report memberi warning eksplisit. Metadata policy ikut berubah untuk input parsial
agar identitas analisis tidak disamakan dengan perilaku diagnosis lama.

Warning yang sama muncul dalam canonical JSON, Markdown, checker CLI dan web.
Web memakai judul analisis dengan keterbatasan dan label detected changes.
Tidak mengonversi jenis gateway secara diam-diam, mengarang file implementasi,
atau menambahkan score relevansi karena adanya warning. Penandaan ini bukan
validator XSD penuh dan bukan daftar lengkap seluruh extension BPMN/vendor.

Tes baru memeriksa siklus dengan perubahan condition JSON, multi-instance,
standard loop, boundary/timer/message, event subprocess, event-based gateway,
call reference, collaboration/message flow, determinisme warning dan report,
serta penerusan warning ke Markdown ketika semantic-only edit belum didiff.
Tes sintetik memakai embedder stub bila diperlukan, bukan bukti akurasi embedding.

## Status penelitian sebelum Qwen

Snapshot mapping Staff P2P L3 tetap diverifikasi melalui `--verify`; JSON/Markdown
baseline tidak dimasukkan ke retrieval. Notasi sendTask respons pada baseline
dipertahankan sebagai representasi pengiriman hasil pada batas request sistem;
bukan bukti listener/message broker atau BPMN runtime pada ERP. Perubahan notasi
berikutnya harus menjadi snapshot baru, bukan diam-diam menimpa pasangan evaluasi.

AND/OR/siklus tidak perlu ditolak hanya karena pilot lain belum mendukungnya.
Untuk mengklaim kualitas mappingnya, tetap perlu acuan dampak per skenario yang
disiapkan sebelum inference. Evaluasi empat anchor yang tersedia belum melabeli
seluruh perubahan primer pada diagram gabungan dan bukan held-out validation.

Urutan berikutnya: lengkapi label primer gabungan, tambah variasi kebutuhan yang
belum dipakai menyetel pencarian, lakukan error analysis dan bandingkan metode
dengan acuan tetap. Tidak perlu menambah metode atau Qwen untuk menutupi gap
dukungan semantik yang sekarang sudah terlihat.
