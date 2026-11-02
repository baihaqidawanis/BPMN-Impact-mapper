# Query khusus perubahan kondisi gateway

Hipotesis development: query yang panjang mencampurkan syarat lama yang tidak
berubah dengan requirement baru. Coba nama gateway plus kata kondisi yang
berbeda, tanpa bobot task ID/path, model baru atau perubahan RRF.

## Implementasi

- Bandingkan token kondisi lama/baru untuk setiap outgoing flow dengan ID yang
  sama, di gateway yang sama. Simpan symmetric difference terurut di
  `context.condition_delta`. Kata di kedua sisi sama pentingnya: kondisi yang
  dihapus juga harus dapat ditemukan dalam implementasi lama.
- Hanya opsi offline `condition-focus` yang membuat anotasi dan memakainya.
  Query memakai nama gateway serta delta; process name dan immediate-neighbour
  expansion tidak digunakan untuk node tersebut. Konteks lengkap tetap ada di
  canonical JSON. Jenis change dan required action tidak diganti.
- Jika tidak ada delta kata, perubahan operator saja, perubahan boolean saja,
  atau ada branch ditambah/dihapus/dipindahkan, gunakan query konteks lengkap.
  Ini bukan solver kondisi BPMN dan tidak menilai benar/salah ekspresi.
- Perubahan task/gateway lain tetap memakai query semula. Model MiniLM, source
  index, threshold semantic, kandidat top10 dan RRF k60 tidak diubah.

Engine produksi tidak memakai opsi ini. Metadata experimental mencatat versi
query dan analysis ID berbeda; input fingerprint tetap sama untuk input yang
sama. Tidak ada acuan, nama file wajib, framework atau rule Simple-ERP di query.

## Protokol

Pakai manifest [selection-challenges](selection-challenges/README.md) dan acuan
yang sudah dibekukan. Dua varian (full/condition-focus), tiga kasus, dua
pengulangan: 12 analisis. Acuan sudah pernah terlihat pada development sebelumnya,
jadi hasil ini tidak boleh dilaporkan sebagai pengujian independen.

Perbaikan fallback untuk target-flow dan boolean-only dilengkapi saat eksperimen
berjalan; keduanya tidak muncul pada manifest ini sehingga query kasus yang
diukur tidak berubah. Semua fallback harus lulus unit test pada source final.

```powershell
go run ./cmd/bpmn-impact-context-study --manifest experiments/simple-erp/selection-challenges/manifest.json --output outputs/condition-focus-challenges --variants full,condition-focus --node-scope --timeout 1h
python experiments/simple-erp/summarize_baseline_context.py outputs/condition-focus-challenges --candidate-variant condition-focus
```

Output run dan tabel hasil otomatis berada di `outputs/condition-focus-challenges`.
Jangan mengganti label atau memperbesar top-k untuk menaikkan metrik kasus ini.

## Hasil dan keputusan

| Kasus | Varian | Recall@5 | MAP |
|---|---|---:|---:|
| Hapus pemeriksaan | full | 100% | 0,7000 |
| Hapus pemeriksaan | condition-focus | 100% | 0,8500 |
| Perluas kondisi | full | 66,67% | 0,5000 |
| Perluas kondisi | condition-focus | 66,67% | 0,3333 |

Seluruh 12 prediction/evaluation berulang identik. Semua full baseline match
byte dengan studi sebelumnya. Pada penghapusan, integration test di gateway
naik 5 -> 2; pada perluasan kondisi, store turun 1 -> 2 dan frontend tetap tidak
masuk top10. Union top5 penghapusan juga membesar 6 -> 7 file: naiknya MAP tidak
berarti seluruh aspek precision membaik.

**Ditolak untuk default.** Delta token saja menghilangkan konteks syarat lama
yang juga dibutuhkan untuk menemukan implementasi existing. Query bukan solver
logika atau bukti dependensi. Kode tetap tersedia untuk reproduksi offline,
bukan metode kelima di dropdown atau pengganti Hybrid produksi.
