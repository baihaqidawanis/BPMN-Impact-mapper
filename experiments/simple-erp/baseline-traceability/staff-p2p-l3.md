# Mapping baseline Staff P2P L3

Snapshot: `df5ed08f8e69212904abf9f865a2faee`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; dirty: True.

10 proses, 50 aktivitas. Acuan sementara dari inspeksi source oleh satu reviewer AI; belum review independen atau bukti runtime.
Tabel menunjukkan implementasi existing. Hubungan ini bukan daftar file yang otomatis wajib berubah. Tidak digunakan sebagai input retrieval.

JSON adalah data utama; Markdown ini dirender dari object JSON yang sama. Seluruh path relatif terhadap root Simple-ERP. Nomor baris berlaku pada hash snapshot.

## Catat invoice supplier - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-ap-invoice.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_ap_invoice_input<br>Isi invoice supplier dan referensi PO<br>userTask / Staff | frontend/features/finance/finance-workspace.tsx | FinancePanel | Input pengguna | L136: async function createInvoice(event: FormEvent) {<br>L150: await command("/supplier-invoices", { purchaseOrderId: source, supplierInvoiceNumber: externalNumber, invoiceDate: today, dueDate: today, currency: "IDR", replacesInvoiceId: replacement?.id \|\| "", lines }); |
| role_staff_l3_ap_invoice_check<br>PO RECEIVED; metadata/lines valid; jika pengganti, invoice terakhir harus VOID dan ID-nya dirujuk<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierInvoice | Validasi aturan | L87: if len(input.Lines) == 0 \|\| input.SupplierInvoiceNumber == "" \|\| input.InvoiceDate == "" \|\| input.DueDate == "" \|\| input.Currency == "" {<br>L96: if err = tx.QueryRow(ctx, `SELECT supplier_id,status FROM purchase_orders WHERE id=$1 FOR UPDATE`, input.PurchaseOrderID).Scan(&amp;supplierID, &amp;status); err != nil \|\| status != "RECEIVED" {<br>L103: if !replacementValid {<br>L113: if line.Quantity &lt;= 0 \|\| line.UnitPrice &lt; 0 { |
| role_staff_l3_ap_invoice_effect0<br>Simpan invoice MATCH_PENDING dan line invoice<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierInvoice | Penyimpanan data/state | L118: _, err = tx.Exec(ctx, `INSERT INTO supplier_invoices(id,document_no,supplier_invoice_number,purchase_order_id,supplier_id,replaces_invoice_id,status,invoice_date,due_date,currency,total,outstanding,created_by)<br>L119: VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,'MATCH_PENDING',$7,$8,$9,$10,$10,$11)`, id, no, input.SupplierInvoiceNumber, input.PurchaseOrderID, supplierID, input.ReplacesInvoiceID, input.InvoiceDate, input.DueDate, input.Currency, total, actor)<br>L124: _, err = tx.Exec(ctx, `INSERT INTO supplier_invoice_lines(supplier_invoice_id,purchase_order_line_id,quantity,unit_price,line_total) VALUES($1,$2,$3,$4,$5)`, id, line.ReferenceID, line.Quantity, line.UnitPrice, line.Quantity*line.UnitPrice) |
| role_staff_l3_ap_invoice_effect1<br>Audit PUR-INV-001; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierInvoice | Audit dan commit | L129: if err = audit(ctx, tx, actor, "RECORD_SUPPLIER_INVOICE", "PUR-INV-001", "SUPPLIER_INVOICE", id, "", "MATCH_PENDING", "", request); err != nil {<br>L132: if err = tx.Commit(ctx); err != nil { |
| role_staff_l3_ap_invoice_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.createSupplierInvoice | Delegasi hasil respons | L124: result(w, document, err, 201) |
| role_staff_l3_ap_invoice_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | Respons sukses/gagal | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_staff_l3_ap_invoice_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.createSupplierInvoice` — API_SERVICE_BINDING: L123: document, err := h.Service.CreateSupplierInvoice(r.Context(), input, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.CreateSupplierInvoice` — DELEGATION: L81: return s.store.CreateSupplierInvoice(c, in, actor, request)
- `backend/internal/finance/adapter/postgres/store.go` / `supplierReplacementValid` — VALIDATION_HELPER: L58: return validateReplacement("", "", requestedID, false), nil
- `backend/internal/finance/adapter/postgres/store.go` / `validateReplacement` — REPLACEMENT_RULE: L47: func validateReplacement(latestID, latestStatus, requestedID string, hasPriorInvoice bool) bool {
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L159: invoice, e := finance.CreateSupplierInvoice(ctx, financeapp.SupplierInvoiceInput{

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Catat pembayaran supplier - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-ap-payment.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_ap_payment_input<br>UI: bayar satu invoice; isi nominal<br>userTask / Staff | frontend/features/finance/finance-workspace.tsx | FinancePanel | Input pengguna | L157: async function createPayment(event: FormEvent) {<br>L160: const created = await command(prefix, { partyId: party, paymentDate: today, method: "BANK_TRANSFER", currency: "IDR", amount: Number(amount), allocations: [{ invoiceId: invoice, amount: Number(amount) }] }); |
| role_staff_l3_ap_payment_check<br>Party aktif; alokasi unik sesuai amount; metadata terisi; constraint DB<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierPayment | Validasi aturan | L306: total, err := allocationTotal(input.Allocations)<br>L308: if err != nil \|\| amountErr != nil \|\| amount != total \|\| input.Method == "" \|\| input.Currency == "" \|\| input.PaymentDate == "" {<br>L317: err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE id=$1 AND active)`, input.PartyID).Scan(&amp;active) |
| role_staff_l3_ap_payment_effect0<br>Simpan payment DRAFT dan alokasi invoice<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierPayment | Penyimpanan data/state | L326: if _, err = tx.Exec(ctx, `INSERT INTO supplier_payments(id,document_no,supplier_id,status,payment_date,method,reference,currency,amount,created_by)<br>L327: VALUES($1,$2,$3,'DRAFT',$4,$5,NULLIF($6,''),$7,$8,$9)`, id, no, input.PartyID, input.PaymentDate, input.Method, input.Reference, input.Currency, input.Amount, actor); err != nil {<br>L331: tag, execErr := tx.Exec(ctx, `INSERT INTO supplier_payment_allocations(payment_id,invoice_id,amount) |
| role_staff_l3_ap_payment_effect1<br>Audit PUR-PAY-001; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.CreateSupplierPayment | Audit dan commit | L337: if err = audit(ctx, tx, actor, "CREATE_SUPPLIER_PAYMENT", "PUR-PAY-001", "SUPPLIER_PAYMENT", id, "", "DRAFT", "", request); err != nil {<br>L340: if err = tx.Commit(ctx); err != nil { |
| role_staff_l3_ap_payment_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.createSupplierPayment | Delegasi hasil respons | L174: result(w, document, err, 201) |
| role_staff_l3_ap_payment_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | Respons sukses/gagal | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_staff_l3_ap_payment_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.createSupplierPayment` — API_SERVICE_BINDING: L173: document, err := h.Service.CreateSupplierPayment(r.Context(), input, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.CreateSupplierPayment` — DELEGATION: L96: return s.store.CreateSupplierPayment(c, in, actor, request)
- `backend/internal/finance/adapter/postgres/store.go` / `allocationTotal` — VALIDATION_HELPER: L280: func allocationTotal(allocations []application.AllocationInput) (int64, error) {; L282: return 0, ErrConflict
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L173: payment, createErr := finance.CreateSupplierPayment(ctx, financeapp.PaymentInput{

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Perbaiki mismatch invoice supplier - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-ap-resolve.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_ap_resolve_input<br>Revisi lines invoice exception<br>userTask / Staff | frontend/features/finance/finance-workspace.tsx | FinancePanel | Input pengguna | L131: async function beginResolve(item: Document) {<br>L141: const resolved = await command(`/supplier-invoices/${resolvingInvoice.id}/resolve-match`, { lines }); |
| role_staff_l3_ap_resolve_check<br>Lock invoice MATCH_EXCEPTION; line qty/harga valid<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.ResolveSupplierInvoice | Validasi aturan | L214: if len(lines) == 0 {<br>L224: Scan(&amp;document.ID, &amp;document.DocumentNo, &amp;document.Status, &amp;document.PartyID, &amp;document.SourceID, &amp;document.Currency, &amp;document.Total, &amp;document.Outstanding, &amp;document.Version); err != nil \|\| document.Status != "MATCH_EXCEPTION" {<br>L232: if line.Quantity &lt;= 0 \|\| line.UnitPrice &lt; 0 { |
| role_staff_l3_ap_resolve_effect0<br>Ganti lines dan total; outstanding direset ke total<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.ResolveSupplierInvoice | Penyimpanan data/state | L227: if _, err = tx.Exec(ctx, `DELETE FROM supplier_invoice_lines WHERE supplier_invoice_id=$1`, id); err != nil {<br>L236: if _, err = tx.Exec(ctx, `INSERT INTO supplier_invoice_lines(supplier_invoice_id,purchase_order_line_id,quantity,unit_price,line_total) VALUES($1,$2,$3,$4,$5)`, id, line.ReferenceID, line.Quantity, line.UnitPrice, line.Quantity*line.UnitPrice); err != nil {<br>L240: if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET status='MATCH_PENDING',total=$2,outstanding=$2,mismatch_reason=NULL,version=version+1,updated_at=now() WHERE id=$1`, id, total); err != nil { |
| role_staff_l3_ap_resolve_effect1<br>Status MATCH_PENDING; audit PUR-INV-003; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.ResolveSupplierInvoice | Audit dan commit | L240: if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET status='MATCH_PENDING',total=$2,outstanding=$2,mismatch_reason=NULL,version=version+1,updated_at=now() WHERE id=$1`, id, total); err != nil {<br>L243: if err = audit(ctx, tx, actor, "RESOLVE_SUPPLIER_MATCH", "PUR-INV-003", "SUPPLIER_INVOICE", id, "MATCH_EXCEPTION", "MATCH_PENDING", "", request); err != nil {<br>L246: if err = tx.Commit(ctx); err != nil { |
| role_staff_l3_ap_resolve_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.resolveSupplierInvoice | Delegasi hasil respons | L144: result(w, document, err, 200) |
| role_staff_l3_ap_resolve_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | Respons sukses/gagal | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_staff_l3_ap_resolve_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.resolveSupplierInvoice` — API_SERVICE_BINDING: L143: document, err := h.Service.ResolveSupplierInvoice(r.Context(), chi.URLParam(r, "id"), input.Lines, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.ResolveSupplierInvoice` — DELEGATION: L87: return s.store.ResolveSupplierInvoice(c, id, lines, actor, request)
- `backend/test/integration/finance_test.go` / `TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid` — RELATED_TEST_NOT_EXECUTED: L77: invoice, err = finance.ResolveSupplierInvoice(ctx, invoice.ID, []financeapp.InvoiceLineInput{{ReferenceID: orderLineID, Quantity: 10, UnitPrice: 100}}, operatorID, "fin-p2p-10")

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Batalkan Goods Receipt draft - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-gr-cancel.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_gr_cancel_input<br>Isi alasan pembatalan GR<br>userTask / Staff | frontend/features/purchasing/goods-receipt-panel.tsx | GoodsReceiptPanel | Input pengguna | L21: &lt;form className="operation-card" onSubmit={(event) =&gt; { event.preventDefault(); if (canApprove(session?.role)) void run(`/goods-receipts/${receiptId}/post`, { version: Number(version) }, "Goods receipt berhasil diposting."); }}&gt;&lt;strong&gt;Kelola Goods Receipt&lt;/strong&gt;&lt;label&gt;Goods Receipt ID&lt;input value={receiptId} onChang |
| role_staff_l3_gr_cancel_check<br>GR DRAFT; alasan tidak kosong<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelReceipt | Validasi aturan | L546: if reason == "" {<br>L555: e = tx.QueryRow(ctx, `UPDATE goods_receipts SET status='CANCELLED',reason=$2,version=version+1 WHERE id=$1 AND status='DRAFT' RETURNING id,document_no,status,creator_id,version`, id, reason).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Version) |
| role_staff_l3_gr_cancel_effect0<br>Ubah GR CANCELLED; naikkan versi<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelReceipt | Penyimpanan data/state | L555: e = tx.QueryRow(ctx, `UPDATE goods_receipts SET status='CANCELLED',reason=$2,version=version+1 WHERE id=$1 AND status='DRAFT' RETURNING id,document_no,status,creator_id,version`, id, reason).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Version) |
| role_staff_l3_gr_cancel_effect1<br>Audit ACT-PUR-GR-CANCEL; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelReceipt | Audit dan commit | L559: if e = audit(ctx, tx, actor, "CANCEL_GR", "ACT-PUR-GR-CANCEL", "GOODS_RECEIPT", id, "DRAFT", "CANCELLED", reason, request); e != nil {<br>L562: e = tx.Commit(ctx) |
| role_staff_l3_gr_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.cancelReceipt | Respons sukses/gagal | L169: respond(w, 409, map[string]string{"message": e.Error()})<br>L172: respond(w, 200, v) |
| role_staff_l3_gr_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.cancelReceipt` — API_SERVICE_BINDING: L167: v, e := h.Service.CancelReceipt(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.CancelReceipt` — DELEGATION: L92: return s.store.CancelReceipt(c, id, reason, actor, r)

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Catat penerimaan barang - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-gr-create.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_gr_create_input<br>Pilih PO line dan jumlah diterima<br>userTask / Staff | frontend/features/purchasing/goods-receipt-panel.tsx | GoodsReceiptPanel | Input pengguna | L17: function create(event: FormEvent) { event.preventDefault(); return run(`/purchase-orders/${purchaseOrderId}/goods-receipts`, { lines: [{ purchaseOrderLineId, productId, quantity: Number(quantity) }] }, "Goods receipt draft berhasil dibuat."); }<br>L20: {canCreate(session?.role) &amp;&amp; &lt;form className="operation-card" onSubmit={create}&gt;&lt;strong&gt;Buat Goods Receipt&lt;/strong&gt;&lt;label&gt;Issued PO&lt;select value={purchaseOrderId} onChange={(event) =&gt; { void selectOrder(event.target.value); }} required&gt;&lt;option value=""&gt;Pilih purchase order&lt;/option&gt;{orders.map((order) =&gt; &lt;option key={or |
| role_staff_l3_gr_create_check<br>PO ISSUED/PARTIALLY_RECEIVED; line milik PO; qty positif<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateReceipt | Validasi aturan | L270: if e != nil \|\| (status != "ISSUED" &amp;&amp; status != "PARTIALLY_RECEIVED") {<br>L283: if l.Quantity &lt;= 0 {<br>L288: SELECT 1 FROM purchase_order_lines<br>L291: if e != nil \|\| !valid { |
| role_staff_l3_gr_create_effect0<br>Insert Goods Receipt DRAFT dan lines<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateReceipt | Penyimpanan data/state | L278: _, e = tx.Exec(ctx, `INSERT INTO goods_receipts(id,document_no,purchase_order_id,creator_id,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, po, actor)<br>L294: _, e = tx.Exec(ctx, `INSERT INTO goods_receipt_lines(goods_receipt_id,purchase_order_line_id,product_id,quantity)VALUES($1,$2,$3,$4)`, id, l.PurchaseOrderLineID, l.ProductID, l.Quantity) |
| role_staff_l3_gr_create_effect1<br>Audit PUR-GR-001; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateReceipt | Audit dan commit | L299: if e = audit(ctx, tx, actor, "CREATE_GOODS_RECEIPT", "PUR-GR-001", "GOODS_RECEIPT", id, "", "DRAFT", "", request); e != nil {<br>L302: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_gr_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.receipt | Respons sukses/gagal | L205: respond(w, 409, map[string]string{"message": e.Error()})<br>L208: respond(w, 201, v) |
| role_staff_l3_gr_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.receipt` — API_SERVICE_BINDING: L203: v, e := h.Service.CreateReceipt(r.Context(), chi.URLParam(r, "id"), in.Lines, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.CreateReceipt` — DELEGATION: L77: return s.store.CreateReceipt(ctx, po, l, actor, request)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L121: gr, e := s.CreateReceipt(ctx, po.ID, []purchaseapp.ReceiptLineInput{{PurchaseOrderLineID: lineID, ProductID: productID, Quantity: 4}}, operatorID, "r7")

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Konversi PR menjadi Purchase Order - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-po-convert.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_po_convert_input<br>Pilih PR, supplier dan warehouse<br>userTask / Staff | frontend/features/purchasing/purchase-order-panel.tsx | PurchaseOrderPanel | Input pengguna | L19: function convert(event: FormEvent) { event.preventDefault(); return run(`/purchase-requests/${requestId}/purchase-order`, { supplierId, warehouseId }, "Purchase order berhasil dibuat."); }<br>L21: {canCreate(session?.role) &amp;&amp; &lt;form className="operation-card" onSubmit={convert}&gt;&lt;strong&gt;Konversi PR menjadi PO&lt;/strong&gt;&lt;label&gt;Approved PR ID&lt;input value={requestId} onChange={(event) =&gt; setRequestId(event.target.value)} required /&gt;&lt;/label&gt;&lt;MasterDataSelect resource="suppliers" label="Supplier" value={supplierId} onCha |
| role_staff_l3_po_convert_check<br>Lock PR APPROVED; supplier dan warehouse aktif<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ConvertToOrder | Validasi aturan | L208: if e != nil \|\| status != "APPROVED" {<br>L212: e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM suppliers WHERE id=$1 AND active)<br>L213: AND EXISTS(SELECT 1 FROM warehouses WHERE id=$2 AND active)`, supplier, warehouse).Scan(&amp;active) |
| role_staff_l3_po_convert_effect0<br>Salin PR lines ke PO DRAFT baru<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ConvertToOrder | Penyimpanan data/state | L222: _, e = tx.Exec(ctx, `INSERT INTO purchase_orders(id,document_no,purchase_request_id,supplier_id,warehouse_id,creator_id,status)VALUES($1,$2,$3,$4,$5,$6,'DRAFT')`, po, no, id, supplier, warehouse, actor)<br>L226: _, e = tx.Exec(ctx, `INSERT INTO purchase_order_lines(purchase_order_id,product_id,quantity,unit_price)SELECT $1,product_id,quantity,estimated_price FROM purchase_request_lines WHERE purchase_request_id=$2`, po, id) |
| role_staff_l3_po_convert_effect1<br>PR menjadi CONVERTED_TO_PO; audit; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ConvertToOrder | Audit dan commit | L230: _, e = tx.Exec(ctx, `UPDATE purchase_requests SET status='CONVERTED_TO_PO',version=version+1,updated_at=now() WHERE id=$1`, id)<br>L234: if e = audit(ctx, tx, actor, "CONVERT_TO_PO", "PUR-PO-001", "PURCHASE_ORDER", po, "", "DRAFT", "", request); e != nil {<br>L237: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_po_convert_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.prAction | Respons sukses/gagal | L137: respond(w, 409, map[string]string{"message": e.Error()})<br>L140: respond(w, 200, v) |
| role_staff_l3_po_convert_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — API_SERVICE_BINDING: L122: v, e = h.Service.ConvertToOrder(r.Context(), chi.URLParam(r, "id"), in.SupplierID, in.WarehouseID, p.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.ConvertToOrder` — DELEGATION: L71: return s.store.ConvertToOrder(ctx, id, supplier, warehouse, actor, request)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L102: po, e := s.ConvertToOrder(ctx, pr.ID, supplierID, warehouseID, operatorID, "r5")

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Batalkan PR draft - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-cancel.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_pr_cancel_input<br>Cancel PR draft; UI kirim alasan demo<br>userTask / Staff | frontend/features/purchasing/purchase-request-panel.tsx | PurchaseRequestPanel | Input pengguna | L38: const reason = name === "reject" \|\| name === "cancel" ? "Diproses dari demo Mini ERP" : "";<br>L55: {document.status === "DRAFT" &amp;&amp; canCreate(session?.role) &amp;&amp; &lt;&gt;&lt;button onClick={() =&gt; action(document, "submit")}&gt;Submit&lt;/button&gt;&lt;button onClick={() =&gt; action(document, "cancel")}&gt;Cancel&lt;/button&gt;&lt;/&gt;} |
| role_staff_l3_pr_cancel_check<br>PR DRAFT; versi cocok; actor creator atau Super Admin; alasan ada<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Validasi aturan | L141: if d.Version != version {<br>L179: e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN')`, actor).Scan(&amp;admin)<br>L180: if e != nil \|\| from != "DRAFT" \|\| reason == "" \|\| (actor != d.CreatorID &amp;&amp; !admin) { |
| role_staff_l3_pr_cancel_effect0<br>Ubah PR CANCELLED; naikkan versi<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Penyimpanan data/state | L183: d.Status = "CANCELLED"<br>L187: tag, e := tx.Exec(ctx, `UPDATE purchase_requests SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject') THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version) |
| role_staff_l3_pr_cancel_effect1<br>Audit ACT-PUR-PR-CANCEL; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Audit dan commit | L177: trace = "ACT-PUR-PR-CANCEL"<br>L191: if e = audit(ctx, tx, actor, "PR_"+action, trace, "PURCHASE_REQUEST", id, from, d.Status, reason, request); e != nil {<br>L194: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_pr_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.prAction | Respons sukses/gagal | L137: respond(w, 409, map[string]string{"message": e.Error()})<br>L140: respond(w, 200, v) |
| role_staff_l3_pr_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — API_SERVICE_BINDING: L134: v, e = h.Service.TransitionRequest(r.Context(), chi.URLParam(r, "id"), action, p.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.TransitionRequest` — DELEGATION: L68: return s.store.TransitionRequest(ctx, id, action, actor, reason, version, request)

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Salin PR ditolak menjadi draft baru - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-copy.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_pr_copy_input<br>Pilih Copy to Draft<br>userTask / Staff | frontend/features/purchasing/purchase-request-panel.tsx | PurchaseRequestPanel | Input pengguna | L57: {document.status === "REJECTED" &amp;&amp; canCreate(session?.role) &amp;&amp; &lt;button onClick={() =&gt; action(document, "copy")}&gt;Copy to Draft&lt;/button&gt;} |
| role_staff_l3_pr_copy_check<br>PR sumber REJECTED; semua produk masih aktif<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CopyRequest | Validasi aturan | L484: e = tx.QueryRow(ctx, `SELECT pr.status,pr.purpose,NOT EXISTS(<br>L488: if e != nil \|\| status != "REJECTED" \|\| !mastersActive { |
| role_staff_l3_pr_copy_effect0<br>Salin purpose/lines ke PR baru; isi copied_from_id<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CopyRequest | Penyimpanan data/state | L496: _, e = tx.Exec(ctx, `INSERT INTO purchase_requests(id,document_no,creator_id,purpose,status,copied_from_id)VALUES($1,$2,$3,$4,'DRAFT',$5)`, newID, no, actor, purpose, id)<br>L500: _, e = tx.Exec(ctx, `INSERT INTO purchase_request_lines(purchase_request_id,product_id,quantity,estimated_price)SELECT $1,product_id,quantity,estimated_price FROM purchase_request_lines WHERE purchase_request_id=$2`, newID, id) |
| role_staff_l3_pr_copy_effect1<br>PR sumber tetap REJECTED; audit; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CopyRequest | Audit dan commit | L504: if e = audit(ctx, tx, actor, "COPY_REJECTED_PR", "ACT-PUR-PR-COPY", "PURCHASE_REQUEST", newID, "REJECTED", "DRAFT", "", request); e != nil {<br>L507: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_pr_copy_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.prAction | Respons sukses/gagal | L137: respond(w, 409, map[string]string{"message": e.Error()})<br>L140: respond(w, 200, v) |
| role_staff_l3_pr_copy_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — API_SERVICE_BINDING: L128: v, e = h.Service.CopyRequest(r.Context(), chi.URLParam(r, "id"), p.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.CopyRequest` — DELEGATION: L86: return s.store.CopyRequest(c, id, actor, r)
- `backend/test/integration/operations_test.go` / `TestCopyRejectedDocumentsRequiresActiveMasters` — RELATED_TEST_NOT_EXECUTED: L359: if _, e = purchasing.CopyRequest(ctx, request.ID, operatorID, "copy-master-pr-4"); e == nil {

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Buat Purchase Request - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-create.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_pr_create_input<br>Isi tujuan pembelian dan line produk<br>userTask / Staff | frontend/features/purchasing/purchase-request-panel.tsx | PurchaseRequestPanel | Input pengguna | L44: {canCreate(session?.role) &amp;&amp; &lt;form className="operation-form" onSubmit={create}&gt;<br>L45: &lt;label&gt;Tujuan pembelian&lt;input value={purpose} onChange={(event) =&gt; setPurpose(event.target.value)} placeholder="Contoh: kebutuhan operasional bulan ini" required /&gt;&lt;/label&gt; |
| role_staff_l3_pr_create_check<br>Tujuan terisi; line ada; qty positif; harga nonnegatif; produk aktif<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateRequest | Validasi aturan | L31: if len(lines) == 0 \|\| purpose == "" {<br>L49: if l.Quantity &lt;= 0 \|\| l.UnitPrice &lt; 0 {<br>L53: e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&amp;active) |
| role_staff_l3_pr_create_effect0<br>Insert PR DRAFT, creator dan lines<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateRequest | Penyimpanan data/state | L44: _, e = tx.Exec(ctx, `INSERT INTO purchase_requests(id,document_no,creator_id,purpose,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, actor, purpose)<br>L57: _, e = tx.Exec(ctx, `INSERT INTO purchase_request_lines(purchase_request_id,product_id,quantity,estimated_price)VALUES($1,$2,$3,$4)`, id, l.ProductID, l.Quantity, l.UnitPrice) |
| role_staff_l3_pr_create_effect1<br>Audit PUR-PR-001; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CreateRequest | Audit dan commit | L62: if e = audit(ctx, tx, actor, "CREATE_PURCHASE_REQUEST", "PUR-PR-001", "PURCHASE_REQUEST", id, "", "DRAFT", "", request); e != nil {<br>L65: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_pr_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.create | Respons sukses/gagal | L98: respond(w, 409, map[string]string{"message": e.Error()})<br>L101: respond(w, 201, v) |
| role_staff_l3_pr_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.create` — API_SERVICE_BINDING: L96: v, e := h.Service.CreateRequest(r.Context(), principal(r).UserID, in.Purpose, in.Lines, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.CreateRequest` — DELEGATION: L55: return s.store.CreateRequest(ctx, actor, purpose, l, req)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L89: pr, e := s.CreateRequest(ctx, operatorID, "Partial receipt test", []purchaseapp.LineInput{{ProductID: productID, Quantity: 10, UnitPrice: 100}}, "r1")

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Submit Purchase Request - Staff

BPMN: `docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-submit.bpmn`

| Task ID / nama | File | Function | Peran | Bukti baris kode |
|---|---|---|---|---|
| role_staff_l3_pr_submit_input<br>Kirim PR untuk review<br>userTask / Staff | frontend/features/purchasing/purchase-request-panel.tsx | PurchaseRequestPanel | Input pengguna | L55: {document.status === "DRAFT" &amp;&amp; canCreate(session?.role) &amp;&amp; &lt;&gt;&lt;button onClick={() =&gt; action(document, "submit")}&gt;Submit&lt;/button&gt;&lt;button onClick={() =&gt; action(document, "cancel")}&gt;Cancel&lt;/button&gt;&lt;/&gt;} |
| role_staff_l3_pr_submit_check<br>Lock PR; status DRAFT dan version cocok<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Validasi aturan | L137: e = tx.QueryRow(ctx, `SELECT id,document_no,status,creator_id,version FROM purchase_requests WHERE id=$1 FOR UPDATE`, id).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Version)<br>L141: if d.Version != version {<br>L147: case "submit":<br>L148: if from != "DRAFT" { |
| role_staff_l3_pr_submit_effect0<br>Ubah PR menjadi SUBMITTED; naikkan versi<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Penyimpanan data/state | L151: d.Status = "SUBMITTED"<br>L187: tag, e := tx.Exec(ctx, `UPDATE purchase_requests SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject') THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version) |
| role_staff_l3_pr_submit_effect1<br>Audit PUR-PR-002; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | Audit dan commit | L145: trace := "PUR-PR-002"<br>L191: if e = audit(ctx, tx, actor, "PR_"+action, trace, "PURCHASE_REQUEST", id, from, d.Status, reason, request); e != nil {<br>L194: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_pr_submit_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.prAction | Respons sukses/gagal | L137: respond(w, 409, map[string]string{"message": e.Error()})<br>L140: respond(w, 200, v) |
| role_staff_l3_pr_submit_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | Pengiriman JSON HTTP | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

Rantai operasi pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — API_SERVICE_BINDING: L134: v, e = h.Service.TransitionRequest(r.Context(), chi.URLParam(r, "id"), action, p.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.TransitionRequest` — DELEGATION: L68: return s.store.TransitionRequest(ctx, id, action, actor, reason, version, request)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L93: if _, e = s.TransitionRequest(ctx, pr.ID, "submit", operatorID, "", pr.Version, "r2"); e != nil {

4 event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.

## Batas penggunaan

- Existing implementation relationships do not imply required code changes.
- No test coverage completeness or function-level mapping accuracy measured.
- Other roles/modules, gateway branch semantics and shared domain/schema dependencies are not exhaustively mapped.
- Function bounds in this research script use declaration text, not a claimed TypeScript AST adapter.
