# Pengujian indeks potongan teks

Pertanyaan development: apakah perbedaan ukuran dokumen membuat file frontend
tertinggal dari function Go yang sudah diindeks secara terpisah? Ini percobaan
granularitas indeks, bukan metode kelima di web dan bukan JavaScript AST.

## Dua cakupan yang diuji

- `generic-windows`: full file plus windows untuk semua file non-Go, termasuk
  dokumentasi/schema/config. Pada kasus penghapusan, integration test justru
  turun 5 -> 7 ketika dokumen mendapatkan tambahan windows. Jangan menganggap
  chunking selalu memperbaiki ranking.
- `code-windows`: windows hanya untuk JavaScript, TypeScript, Java, Python, C#
  yang dikenali adapter bahasa. Go tetap memakai `go/parser`; dokumentasi,
  schema/config dan bahasa belum dikenali memakai adapter existing. Cakupan ini
  dikunci sebelum melihat hasil perluasan kondisi pada generic-windows.

Kedua opsi menyimpan full-file artifact, lalu menambahkan maksimal delapan
potongan berurutan yang mencakup seluruh teks tanpa overlap. Minimum 1200 rune;
file besar memakai potongan lebih besar agar jumlahnya tetap dibatasi. File
pendek tidak diduplikasi. Nama function/class tidak ditebak; tiap potongan hanya
memiliki rentang baris asli. Semua kemampuan non-Go tetap TEXT_ONLY, ditambah
regex HTTP-route evidence yang memang tersedia, bukan klaim AST.

Corpus BM25 dan panjang dokumen berubah sehingga IDF/ranking juga berubah.
BM25 tetap k1=1.2/b=.75, semantic tetap MiniLM, RRF tetap k60. Query/context,
threshold dan top10 tidak disetel. Semantic masih memakai passage/token budget
yang sama; full text tersedia untuk BM25 tidak berarti semua token di-embed.

## Acuan dan desain

Manifest di folder ini merupakan subset unchanged dari
`selection-challenges/manifest.json`: perluasan kondisi Copy PR dan kontrol.
Hash BPMN/acuan tidak diedit. Delapan analisis (dua kasus, full/code-windows,
dua pengulangan) untuk tahap fokus. Generic-windows diuji pada ketiga challenge
dengan 12 analisis. Ini data development yang sudah terlihat, bukan independent
benchmark. Jangan menyebut provisional extras sebagai file pasti salah.

```powershell
go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/selection-challenges/manifest.json --output outputs/generic-window-challenges --variants full,generic-windows --node-scope --timeout 1h
python experiments/simple-erp/summarize_baseline_context.py outputs/generic-window-challenges --candidate-variant generic-windows
go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/code-window-study/manifest.json --output outputs/code-window-condition --variants full,code-windows --node-scope --timeout 1h
python experiments/simple-erp/summarize_baseline_context.py outputs/code-window-condition --candidate-variant code-windows
```

## Keputusan

Tidak ada perubahan default sampai hasil dan regresi ditinjau. Jika frontend
masih terlewat atau file acuan lain turun, jangan mempromosikan varian hanya
karena satu angka naik. Pengujian berikutnya dibatasi oleh hasil kasus fokus.

Hasil `generic-windows` (12 analisis identik per pengulangan) menolak adopsi:

| Kasus | Full Recall@5 / MAP | Generic windows Recall@5 / MAP |
|---|---:|---:|
| Hapus pemeriksaan | 100% / 0,7000 | 50% / 0,6429 |
| Perluas kondisi | 66,67% / 0,5000 | 33,33% / 0,4444 |

Frontend tetap di luar top10, integration test turun 5 -> 7 pada removal dan
4 -> 6 pada perluasan kondisi. Tidak ada SEMANTIC_PASSAGE_LIMIT pada run ini.
Prototype pertama ini dibuat sebelum kompatibilitas adapter window dengan
regex HTTP-route extractor ditambahkan. Perbaikan tersebut diuji pada source
final dan dipakai oleh run code-windows; generic-window run lama tidak diklaim
sebagai bukti kesetaraan capability route. Seluruh varian tetap research-only.

Hasil `code-windows` (delapan analisis identik per pengulangan) juga belum
memenuhi syarat adopsi. Pada perluasan kondisi, Full dan Code windows sama-sama
P@5=40%, R@5=66,67%, MAP=0,5000. Store tetap rank1, integration test rank4,
frontend tetap di luar top10. Kontrol menghasilkan nol perubahan/kandidat.
Adapter route compatibility yang telah diperbaiki aktif pada run ini.

**Kedua varian tidak dipasang di default/web.** Pengujian lebih luas untuk
rollout ditunda karena kasus target belum membaik. Tidak ada label, threshold,
top-k atau bobot yang diubah untuk membuat hasil tampak naik. Delapan baseline
full dari ketiga studi sama persis dengan report historis; binding SHA ada di
`outputs/query-window-regression-bindings.json`.

Pengembangan berikutnya perlu menelusuri dependensi source yang lebih spesifik
antara operasi backend dan pemanggil frontend, termasuk keterbatasan literal
route extractor terhadap URL dinamis. Itu hipotesis berikutnya, belum hasil
terbukti dan tidak memerlukan graph database atau mengganti metode publik.
