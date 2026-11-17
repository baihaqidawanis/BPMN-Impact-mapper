# Hasil pemeriksaan offline - 4 November 2026

Paket: current-staff-v2. Empat skenario positif dan satu kontrol identik, repository Simple-ERP lokal read-only. Model embedding lokal digunakan; tidak ada panggilan Qwen/lab. Top K = 10; pengulangan = 2.

## Cakupan

Acuan positif hanya melabeli satu perubahan utama per skenario. Tabel berikut menghitung macro-average empat perubahan utama berlabel, bukan seluruh task/flow di seluruh proses. Kontrol identik dilaporkan terpisah. Acuan berasal dari satu review source dan belum independen.

| Metode | Precision@5 | Recall@5 | Recall@10 | MAP | MRR | R-precision |
|---|---:|---:|---:|---:|---:|---:|
| BM25 | 0.3500 | 0.5834 | 0.7500 | 0.3084 | 0.2500 | 0.2500 |
| Semantic | 0.5500 | 1.0000 | 1.0000 | 0.7320 | 0.8750 | 0.6250 |
| BM25 + Semantic | 0.5500 | 1.0000 | 1.0000 | 0.9306 | 1.0000 | 0.8334 |
| Hybrid dengan konteks BPMN | 0.5500 | 1.0000 | 1.0000 | 0.7986 | 0.8750 | 0.7084 |

## Per skenario

| Skenario | Metode | Precision@5 | Recall@5 | MAP | R-precision |
|---|---|---:|---:|---:|---:|
| add-and | BM25 | 0.0000 | 0.0000 | 0.0000 | 0.0000 |
| add-and | BM25 + Semantic | 0.4000 | 1.0000 | 1.0000 | 1.0000 |
| add-and | Hybrid dengan konteks BPMN | 0.4000 | 1.0000 | 0.7500 | 0.5000 |
| add-and | Semantic | 0.4000 | 1.0000 | 0.5000 | 0.5000 |
| add-task | BM25 | 0.6000 | 1.0000 | 0.4778 | 0.3333 |
| add-task | BM25 + Semantic | 0.6000 | 1.0000 | 1.0000 | 1.0000 |
| add-task | Hybrid dengan konteks BPMN | 0.6000 | 1.0000 | 1.0000 | 1.0000 |
| add-task | Semantic | 0.6000 | 1.0000 | 0.7556 | 0.6667 |
| add-xor | BM25 | 0.4000 | 0.6667 | 0.3778 | 0.3333 |
| add-xor | BM25 + Semantic | 0.6000 | 1.0000 | 0.8056 | 0.6667 |
| add-xor | Hybrid dengan konteks BPMN | 0.6000 | 1.0000 | 0.8056 | 0.6667 |
| add-xor | Semantic | 0.6000 | 1.0000 | 0.7556 | 0.6667 |
| combined | BM25 | 0.4000 | 0.6667 | 0.3778 | 0.3333 |
| combined | BM25 + Semantic | 0.6000 | 1.0000 | 0.9167 | 0.6667 |
| combined | Hybrid dengan konteks BPMN | 0.6000 | 1.0000 | 0.6389 | 0.6667 |
| combined | Semantic | 0.6000 | 1.0000 | 0.9167 | 0.6667 |

## Temuan

- Semua metode mengembalikan nol perubahan pada kontrol identik. Kontrol tidak dimasukkan sebagai recall nol ke rata-rata kasus positif.
- Hybrid memperoleh Recall@5 = 1 pada keempat perubahan utama. Precision@5 rata-rata 0.55 sesuai acuan 2-3 file per kasus.
- BM25 + Semantic memperoleh MAP lebih tinggi daripada Hybrid di paket ini. Konteks BPMN membantu beberapa pencarian, tetapi menambah query tidak menjamin posisi paling atas selalu membaik.
- BM25 gagal menemukan dua anchor AND dalam Top 10 pada perubahan gateway berlabel. Kemungkinan penyebabnya adalah kecocokan istilah query yang terbatas; hasil ini sendiri belum membuktikan penyebab tersebut.
- Status required_action dari retrieval umumnya VERIFY, sedangkan acuan meminta MODIFY; recall tinggi tidak berarti klasifikasi aksi sudah tepat.
- Tidak ada penyesuaian bobot, threshold atau acuan berdasarkan hasil ranking. Referensi inti tetap didukung source.

## Eksekusi dan batasnya

Batch pertama mencapai batas waktu total 8 menit ketika menjalankan combined. Hasil tiga skenario pertama telah tersimpan. Combined dan kontrol diselesaikan lewat batch terpisah dengan konfigurasi sama; benchmark batch kedua melaporkan reproducible=true untuk seluruh delapan pasangan skenario/metode. Tidak ada hasil batch pertama yang diklaim sebagai satu benchmark lengkap.

Artifact lokal: outputs/offline-review-2026-11-04 dan outputs/offline-review-remaining-2026-11-04. Benchmark batch kedua: 1a2779b1e86ff213ffeff7f570162bff. Laporan ini menggabungkan hasil kasus yang berbeda, tidak menghitung combined dua kali. Latency embedding meliputi biaya proses/load lokal dan bukan prediksi latency Qwen.

Tes Go planning, agentjobs, llm dan API lulus, mencakup Stop/Resume, kegagalan koneksi, validasi output, partial plan dan transport combined. Runtime model tiruan menguji protokol, bukan kualitas model lab.

## Berikutnya

Gunakan current-staff-v2/add-task untuk pemeriksaan model pertama, atau combined untuk kebutuhan gabungan. Restart backend, buka tunnel saat VPN tersedia, buat sesi AI baru. Periksa apakah file rekomendasi dan langkah implementasinya didukung evidence; evaluasi keluaran AI terhadap acuan yang sama secara terpisah. Hasil di atas belum mengukur manfaat AI.
