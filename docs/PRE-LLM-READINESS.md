# Batas sebelum menghubungkan model lab

Pipeline menerima repo read-only dan before/after BPMN, menghasilkan ranked
impacts, source connections, serta paket planning dengan whitelist file dan
evidence ID. Provider tetap disabled; belum ada panggilan model generatif.
Ini kesiapan teknis integrasi, bukan klaim akurasi 100% atau evaluasi independen.

## Bahan penelitian yang dikunci

[Freeze penelitian](../experiments/simple-erp/research-freeze/README.md) mengikat
10 kasus development/regression, source review, label dan kode metode. Semua
kasus tersebut sudah terekspos; independent test set masih kosong. Jangan
menggunakan freeze ini sebagai input retrieval atau konteks penilai LLM.

Audit required-change mencatat kebutuhan, alasan per file dan kontrak yang
tetap dapat dipakai, tanpa membaca output kandidat dan tanpa mengganti label
lama. Mapping before tiga modul/tiga role tetap menjadi acuan implementasi
existing, bukan daftar otomatis file yang harus berubah. Acuan perubahan
masih provisional dari review AI, bukan implementation diff after.

Verifikasi freeze sebelum eksperimen berikutnya. Jika source/kode metode/BPMN
berubah, buat versi freeze baru setelah review; jangan menimpa versi ini.
Hash manifest bukan backup source atau pembekuan runtime Python/model. Setiap
inference tetap harus merekam identitas model sebenarnya dan fingerprint.

## Perubahan terbaru

- URL template JS/TS yang sederhana dapat dihubungkan ke satu handler Go yang
  pola route dan HTTP method-nya cocok. Registrasi ganda, parameter/ekspresi
  tidak didukung, komentar/string contoh dan method dinamis tidak dipaksakan.
- Constructor langsung seperti `return Service{store}` dapat menunjukkan
  implementasi interface yang dipasang. Binding berbeda/tidak diketahui,
  direct construction lain dan mutasi field menghentikan asosiasi tambahan.
  Ini observed wiring, bukan pembuktian seluruh kemungkinan runtime.
- Maksimal lima file kandidat awal menjadi sumber pencarian function dengan
  kecocokan nama unik minimal dua kata. Maksimal tiga reverse source edges,
  128 node yang dikunjungi, dan tiga file terkait per perubahan. Tidak ada
  graph database atau eksekusi source.
- File terkait ditampilkan terpisah dengan action VERIFY, tanpa skor tambahan.
  File yang juga berada di ranked list mencatat posisinya. Semua ranking,
  retrieval score dan evaluasi Precision/Recall tetap memakai `impacts` saja.
- Paket planner memakai maksimal sepuluh kandidat ranked dan tiga file terkait
  yang belum ada dalam batas itu, maksimal delapan evidence per kandidat dan
  240 rune per excerpt. File terkait tidak diberi nilai nol palsu: score diomit.
- Before/after element, actor/state/data/event metadata dan arah perubahan
  kondisi gateway disediakan. Konteks panjang dibatasi dan ditandai. Warning
  analisis ikut dibawa. Requirement terpotong/tidak lengkap atau BPMN partial
  memaksa NEEDS_REVIEW, meskipun provider reviewer memberikan approval.
- Model tetap dilarang membuat file/symbol/evidence ID di luar whitelist.
  Output hanya rencana JSON/Markdown; tidak menulis ulang source aplikasi.

## Bukti pada kasus ERP yang sama

`outputs/pre-llm-production-final` memuat dua run pipeline produksi pada
perluasan kondisi Copy PR dan unchanged control. Prediction, evaluation dan
planning-context JSON identik antar pengulangan. Ranked impacts tetap sama
persis dengan baseline lama. Caller frontend ditemukan lewat:

`Store.CopyRequest -> Service.CopyRequest -> Handler.prAction -> purchase-request-panel.tsx`

Bukti mencakup constructor di composition root, route berparameter, pemanggilan
API dan baris guard tombol Copy yang masih memakai REJECTED. Tiga file acuan
provisional tersedia dalam paket planner; sebelumnya frontend di luar top10.
Ini **coverage paket konteks pada satu kasus**, bukan kenaikan Recall@5 atau
bukti bahwa setiap related file wajib diubah. Kasus hapus pemeriksaan juga bisa
memiliki caller terkait meskipun requirement tidak mewajibkan perubahan UI.

Run awal `outputs/pre-llm-source-callers` adalah studi association sebelum
penambahan directed branch context v4; jangan dicampur dengan run final.
Canonical analysis ID berubah karena versi kebijakan/registry berubah.
Source ERP tidak diubah. Acuan tetap hasil review source sementara.

## Memeriksa di web

Restart backend setelah perubahan Go, refresh frontend. Jalankan kasus yang
sama lagi; laporan lama tetap tersimpan tetapi tidak ditambahkan source links
secara retroaktif. Buka **Related source connections** pada report, kemudian
**Planning** untuk melihat before/after, kondisi gateway dan kandidat Not ranked.
Input file/draft Canvas masih perlu dipilih ulang setelah refresh; hasil analisis
tersimpan dan dapat dibuka lewat Saved analyses.

Ekspor paket tanpa model atau membaca ulang repo:

```powershell
go run ./cmd/bpmn-impact-mapper planning-context --report outputs/pre-llm-production-final/selection-challenge-change-condition/production/run1/impact-report.json --output outputs/planning-context-review.json
```

Output harus file baru. Paket yang identik menghasilkan context ID identik.

## Yang masih diperlukan dari lab

Endpoint/protokol API, nama model, autentikasi bila ada, serta batas context dan
concurrency. Setelah tersedia, baru uji connector, timeout, respons terstruktur,
evidence validation, penolakan hallucination dan kualitas rencana. Qwen/Mistral
atau model lain dapat memakai kontrak provider yang sama jika API-nya cocok.
Tidak perlu memilih framework agent baru untuk menjalankan pipeline saat ini.

Keterbatasan: source associations saat ini terbatas ke pola JS/TS + Go yang
diimplementasikan. Bahasa/arsitektur lain tetap lexical/semantic dan adapter
fallback. Alias/router grouping/ekspresi URL kompleks, reflection dan runtime DI
tidak dijamin. Generalisasi dan kualitas rencana belum terbukti lintas repository.
