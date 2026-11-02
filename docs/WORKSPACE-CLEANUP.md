# Perapian workspace

Root sekarang berisi source, konfigurasi Go/Git, README, PLAN dan dua dokumen
utama arah TA/metode. Mulai dari README, lalu pilih panduan paket Staff aktif.

Sepuluh file lepas dipindah ke `outputs/manual-inputs/`: dua BPMN manual, satu
impact report, lima gambar dan dua executable. File tidak dihapus. Manifest
berisi nama asal, lokasi baru dan SHA-256; hash dicek setelah pemindahan.

Executable tersebut adalah hasil build lama; command `go run` pada README tidak
membutuhkannya. Upload/report baru dari pengguna sebaiknya masuk folder ini juga.

Source `cmd/`, `internal/`, `web/`, fixture `testdata/`, data `.data/`, cache dan
semua hasil eksperimen lama dipertahankan. Tidak melakukan reset Git, menghapus
perubahan pengguna, commit atau push. Tidak mengubah arsitektur paket backend.

Dokumen teknis tetap di `docs/` dengan indeks `docs/README.md`. Eksperimen di
`experiments/` memiliki indeks sendiri yang membedakan paket aktif dan historis.
Nama file eksperimen lama tetap agar referensi dan jejak keputusan tidak rusak.
