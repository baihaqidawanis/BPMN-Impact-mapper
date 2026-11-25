# Rencana perubahan submit stock adjustment

Revisi manual dari [plan Gateway `575de4c7`](../../.data/plans/575de4c746785beab9e2d0592c1fc480/change-plan.md), sesi `ff1d7fa3305f0b5787e4c7555fa076e9`. Ruang lingkup edit dikunci pada tiga file `MODIFY` dari kondisi gateway `role_staff_l3_adj_submit_valid`. Hasil run asli mencatat precision 100% dan recall 100% untuk kondisi tersebut. Revisi ini adalah rencana implementasi untuk ditinjau developer. Hasil agent tetap tersimpan dengan status `NEEDS_REVIEW`.

## Perilaku yang dituju

Staff bisa submit adjustment jika statusnya `DRAFT`, versi cocok, dan request berisi alasan. Alasan yang sebelumnya tersimpan boleh kosong. Submit yang berhasil menyimpan alasan baru, menaikkan versi satu kali, dan mencatat audit. Jika aturan submit gagal, respons menyampaikan penyebab penolakan tanpa mengubah status, versi, atau audit submit.

## Perubahan per file

1. **`backend/internal/inventory/adapter/postgres/store.go`** — `Store.TransitionAdjustment` membaca record dengan `FOR UPDATE` dan memeriksa versi sebelum dispatch. Cabang `submit` sudah memeriksa `DRAFT`. Tambahkan pemeriksaan `strings.TrimSpace(reason) == ""` di cabang `submit`, sebelum `d.Status` diubah. Parameter `reason` menerima nilai dari request, sedangkan parameter `request` adalah ID untuk audit. Gunakan pemeriksaan versi yang sudah ada. Pada kegagalan validasi submit, kembalikan error yang membungkus `ErrConflict` dengan pesan singkat sesuai penyebabnya, misalnya versi tidak cocok, status bukan `DRAFT`, atau alasan kosong. Pertahankan penanganan error query serta konflik pada aksi lain. Jalur sukses sudah memakai `reason` untuk UPDATE dan audit dalam transaksi yang sama.
2. **`frontend/features/inventory/inventory-adjustment-panel.tsx`** — `action("submit")` saat ini mengirim `reason: ""`. Perluas label input alasan agar mencakup submit dan teruskan nilai input pada aksi itu. Tampilkan pesan pada form jika alasan kosong atau hanya spasi. Biarkan backend menentukan kecocokan versi dari angka yang sudah dikirim dengan `Number(version)`. Alur reject, cancel, dan reverse tetap memakai input yang sama sesuai perilakunya saat ini.
3. **`backend/test/integration/operations_test.go`** — Isi alasan valid pada submit di `TestInventoryApprovalRejectsChangedSnapshot` dan pada submit awal serta submit pengganti di `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot`. Pertahankan asersi persetujuan dan pemulihan snapshot di kedua tes. Tambahkan tes terpisah untuk submit gagal yang membaca kembali record agar status dan versi yang diperiksa berasal dari database.

## Hubungan dengan perubahan BPMN

- **Task pengecekan dan kondisi gateway.** Lock, versi, status, dan alasan diputuskan sebelum mutasi. Satu edit pada jalur submit mencakup perubahan aturan tersebut.
- **Task penolakan.** Store menyampaikan penyebab konflik. `Handler.action` sudah meneruskan `e.Error()` melalui respons HTTP `409` dengan bentuk `{"message": "…"}`. Service sudah meneruskan alasan dan versi ke store. Kedua jalur ini mendukung perubahan tanpa edit tambahan pada handler atau service.
- **XOR join dan sequence flow.** Jalur sukses dan gagal berakhir pada penanganan respons yang sudah ada. Sukses mengirim HTTP `200` setelah commit. Gagal mengirim HTTP `409` tanpa commit submit. Join menggambarkan bahwa salah satu hasil dikirim ke pengguna dan tidak menambah operasi transaksi.

## Acceptance checks

- `DRAFT` dengan versi cocok, alasan lama kosong, dan alasan baru valid berhasil submit. Baca ulang record untuk memastikan status `SUBMITTED`, versi naik satu kali, dan alasan baru tersimpan. Pastikan audit submit berisi alasan yang sama.
- Alasan kosong, alasan berisi spasi saja, versi tidak cocok, dan status selain `DRAFT` diuji sebagai kasus terpisah. Tiap kasus mengembalikan konflik dengan pesan sesuai penyebab. Status dan versi setelah dibaca ulang sama seperti sebelumnya dan tidak ada audit submit baru.
- Form meneruskan alasan yang diisi saat submit, menampilkan pesan saat alasan kosong, dan menampilkan pesan respons jika backend menolak submit karena status atau versi.
- Dua tes snapshot yang sudah ada tetap mencapai pemeriksaan persetujuan atau pemulihan yang menjadi tujuan awalnya.

Lihat [BPMN After](../../experiments/simple-erp/current-staff-v2/add-gateway/after.bpmn) dan [acuan file](../../experiments/simple-erp/current-staff-v2/add-gateway/reference.json).
