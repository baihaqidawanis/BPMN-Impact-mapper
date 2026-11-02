# Mapping before: supervisor / p2p

Snapshot paket: `876d5d30117e798a8733986217f19434`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; working tree dirty: True.

Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.
JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.

## Void invoice supplier - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-ap-invoice-void.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_ap_invoice_void_input<br>Isi alasan void invoice<br>userTask / Supervisor | frontend/features/finance/finance-workspace.tsx | FinancePanel | USER_INPUT_OR_VIEW | L218: ... === "DRAFT" &amp;&amp; &lt;button disabled={!voidReason} onClick={() =&gt; command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}&gt;Void&lt;/button&gt;}<br>L218: ...wApprove &amp;&amp; kind === "customer" &amp;&amp; item.status === "DRAFT" &amp;&amp; &lt;button disabled={!voidReason} onClick={() =&gt; command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}&gt;Void&lt;/button&gt;} |
| role_supervisor_l3_ap_invoice_void_check<br>Invoice POSTED; tanpa alokasi dari payment POSTED; alasan ada<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierInvoice | VALIDATION | L255: if reason == "" {<br>L265: WHERE i.id=$1 AND i.status='POSTED' AND NOT EXISTS(SELECT 1 FROM supplier_payment_allocations a JOIN supplier_payments p ON p.id=a.payment_id WHERE a.invoice_id=i.id AND p.status='POSTED')<br>L265: ... supplier_payments p ON p.id=a.payment_id WHERE a.invoice_id=i.id AND p.status='POSTED') |
| role_supervisor_l3_ap_invoice_void_effect0<br>Tandai invoice VOID; simpan voided_by dan alasan<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierInvoice | PERSISTENCE_OR_STATE | L264: err = tx.QueryRow(ctx, `UPDATE supplier_invoices i SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1,updated_at=now()<br>L264: ...Row(ctx, `UPDATE supplier_invoices i SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1,updated_at=now() |
| role_supervisor_l3_ap_invoice_void_effect1<br>Audit void; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierInvoice | AUDIT_AND_COMMIT | L271: if err = audit(ctx, tx, actor, "VOID_SUPPLIER_INVOICE", "ACT-FIN-SINV-VOID", "SUPPLIER_INVOICE", id, "POSTED", "VOID", reason, request); err != nil {<br>L274: if err = tx.Commit(ctx); err != nil { |
| role_supervisor_l3_ap_invoice_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.voidSupplierInvoice | RESPONSE_DISPATCH | L155: result(w, document, err, 200) |
| role_supervisor_l3_ap_invoice_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | SUCCESS_ERROR_RESPONSE | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_supervisor_l3_ap_invoice_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

**Catatan role_supervisor_l3_ap_invoice_void_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_ap_invoice_void_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Tidak membuat jurnal akuntansi baru; aplikasi tidak memiliki general ledger.


**Catatan role_supervisor_l3_ap_invoice_void_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.voidSupplierInvoice` — API_SERVICE_BINDING: L154: document, err := h.Service.VoidSupplierInvoice(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.VoidSupplierInvoice` — SERVICE_STORE_BINDING: L90: return s.store.VoidSupplierInvoice(c, id, reason, actor, request)
- `backend/internal/finance/adapter/http/handler.go` / `Handler.voidSupplierInvoice` — ROLE_ACCESS_BOUNDARY: L147: if !requireApprove(w, r) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/finance/adapter/http/handler.go` / `requireApprove` — ROLE_CHECK_HELPER: L60: if authdomain.CanApprove(role(r)) {
- `backend/test/integration/finance_test.go` / `TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid` — RELATED_TEST_NOT_EXECUTED: L116: if _, err = finance.VoidSupplierInvoice(ctx, invoice.ID, "blocked by payment", adminID, "fin-p2p-17"); err == nil {

Test: related source inspected, not executed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Match invoice supplier - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-ap-match.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_ap_match_input<br>Pilih Match<br>userTask / Supervisor | frontend/features/finance/finance-workspace.tsx | FinancePanel | USER_INPUT_OR_VIEW | L215: ...Id !== actorId \|\| allowSelfApprove) &amp;&amp; &lt;button onClick={() =&gt; command(`${invoicePrefix}/${item.id}/match`)}&gt;Match&lt;/button&gt;} |
| role_supervisor_l3_ap_match_check<br>Lock invoice MATCH_PENDING; Supervisor bukan creator; Super Admin boleh self-match; baca line PO dan invoice<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.MatchSupplierInvoice | VALIDATION | L147: ...ing, &amp;document.Version, &amp;creatorID); err != nil \|\| document.Status != "MATCH_PENDING" {<br>L151: allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)<br>L194: if matchErr := domain.ExactMatch(expected, actual); matchErr != nil { |
| role_supervisor_l3_ap_match_positive<br>Set invoice POSTED<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.MatchSupplierInvoice | BRANCH_OUTCOME | L192: to := "POSTED" |
| role_supervisor_l3_ap_match_negative<br>Set invoice MATCH_EXCEPTION<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.MatchSupplierInvoice | BRANCH_OUTCOME | L195: to = "MATCH_EXCEPTION"<br>L194: if matchErr := domain.ExactMatch(expected, actual); matchErr != nil { |
| role_supervisor_l3_ap_match_effect1<br>Simpan mismatch_reason dan versi; audit PUR-INV-002; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.MatchSupplierInvoice | AUDIT_AND_COMMIT | L198: _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET status=$1,mismatch_reason=NULLIF($2,''),version=version+1,updated_at=now() WHERE id=$3`, to, reason, id)<br>L202: if err = audit(ctx, tx, actor, "MATCH_SUPPLIER_INVOICE", "PUR-INV-002", "SUPPLIER_INVOICE", id, "MATCH_PENDING", to, reason, request); err != nil {<br>L205: if err = tx.Commit(ctx); err != nil { |
| role_supervisor_l3_ap_match_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.matchSupplierInvoice | RESPONSE_DISPATCH | L131: result(w, document, err, 200) |
| role_supervisor_l3_ap_match_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | SUCCESS_ERROR_RESPONSE | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_supervisor_l3_ap_match_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

**Catatan role_supervisor_l3_ap_match_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_ap_match_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Kode membandingkan line PO dengan invoice. Bukan query three-way match GR terpisah pada saat match; PO RECEIVED sudah disyaratkan saat create.


**Catatan role_supervisor_l3_ap_match_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.matchSupplierInvoice` — API_SERVICE_BINDING: L130: document, err := h.Service.MatchSupplierInvoice(r.Context(), chi.URLParam(r, "id"), actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.MatchSupplierInvoice` — SERVICE_STORE_BINDING: L84: return s.store.MatchSupplierInvoice(c, id, actor, request)
- `backend/internal/finance/adapter/http/handler.go` / `Handler.matchSupplierInvoice` — ROLE_ACCESS_BOUNDARY: L127: if !requireApprove(w, r) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/finance/adapter/http/handler.go` / `requireApprove` — ROLE_CHECK_HELPER: L60: if authdomain.CanApprove(role(r)) {
- `backend/internal/platform/authorization/super_admin.go` / `CanSelfReview` — SELF_REVIEW_POLICY: L12: ...= tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&amp;allowed)
- `backend/test/integration/finance_test.go` / `TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid` — RELATED_TEST_NOT_EXECUTED: L69: invoice, err = finance.MatchSupplierInvoice(ctx, invoice.ID, supervisorID, "fin-p2p-9")

Test: related source inspected, not executed.

4 event/gateway serta 11 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Posting pembayaran supplier - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-ap-payment-post.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_ap_payment_post_input<br>Pilih Post Payment<br>userTask / Supervisor | frontend/features/finance/finance-workspace.tsx | FinancePanel | USER_INPUT_OR_VIEW | L229: ...Id !== actorId \|\| allowSelfApprove) &amp;&amp; &lt;button onClick={() =&gt; command(`${paymentPrefix}/${item.id}/post`, { version: item.version })}&gt;Post&lt;/button&gt;} |
| role_supervisor_l3_ap_payment_post_check<br>Lock payment/invoice; DRAFT dan versi cocok; party/currency/tanggal/outstanding valid; Supervisor bukan creator, Super Admin aktif boleh self-post<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.PostSupplierPayment | VALIDATION | L355: ....Version, &amp;paymentDate, &amp;creatorID); err != nil \|\| document.Status != "DRAFT" \|\| document.Version != version {<br>L355: ...paymentDate, &amp;creatorID); err != nil \|\| document.Status != "DRAFT" \|\| document.Version != version {<br>L359: allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)<br>L388: if err = domain.ValidatePayment(item.amount, item.amount, document.Currency, item.currency, paymentDate, item.invoiceDate); err != nil \|\| item.amount &gt; item.outstanding {<br>L388: ...urrency, item.currency, paymentDate, item.invoiceDate); err != nil \|\| item.amount &gt; item.outstanding { |
| role_supervisor_l3_ap_payment_post_effect0<br>Kurangi outstanding; invoice PARTIALLY_PAID atau PAID<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.PostSupplierPayment | PERSISTENCE_OR_STATE | L408: if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, remaining, status); err != nil {<br>L406: status = "PAID" |
| role_supervisor_l3_ap_payment_post_effect1<br>Jika semua invoice non-VOID lunas: PO RECEIVED ke CLOSED; PR ke COMPLETED<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.PostSupplierPayment | PERSISTENCE_OR_STATE | L420: if _, err = tx.Exec(ctx, `UPDATE purchase_orders SET status='CLOSED',version=version+1,updated_at=now() WHERE id=$1 AND status='RECEIVED'`, item.source); err != nil {<br>L427: err = tx.QueryRow(ctx, `UPDATE purchase_requests r SET status='COMPLETED',version=r.version+1,updated_at=now() FROM purchase_orders p WHERE p.id=$1 AND r.id=p.purchase_request_id AND r.status='CONVERTED_TO_PO' RETURNING r.id`, item.source).Scan(&amp;requestID) |
| role_supervisor_l3_ap_payment_post_effect2<br>Payment POSTED; audit alokasi/penutupan; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.PostSupplierPayment | AUDIT_AND_COMMIT | L437: if _, err = tx.Exec(ctx, `UPDATE supplier_payments SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id); err != nil {<br>L411: if err = audit(ctx, tx, actor, "APPLY_SUPPLIER_PAYMENT", "PUR-PAY-002", "SUPPLIER_INVOICE", item.invoice, item.status, status, "", request); err != nil {<br>L443: if err = tx.Commit(ctx); err != nil { |
| role_supervisor_l3_ap_payment_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.postSupplierPayment | RESPONSE_DISPATCH | L185: result(w, document, err, 200) |
| role_supervisor_l3_ap_payment_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | SUCCESS_ERROR_RESPONSE | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_supervisor_l3_ap_payment_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

**Catatan role_supervisor_l3_ap_payment_post_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_ap_payment_post_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Created_by dibandingkan dengan actor, kecuali Super Admin aktif sebagai bypass.


**Catatan role_supervisor_l3_ap_payment_post_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.postSupplierPayment` — API_SERVICE_BINDING: L184: document, err := h.Service.PostSupplierPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.PostSupplierPayment` — SERVICE_STORE_BINDING: L99: return s.store.PostSupplierPayment(c, id, version, actor, request)
- `backend/internal/finance/adapter/http/handler.go` / `Handler.postSupplierPayment` — ROLE_ACCESS_BOUNDARY: L177: if !requireApprove(w, r) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/finance/adapter/http/handler.go` / `requireApprove` — ROLE_CHECK_HELPER: L60: if authdomain.CanApprove(role(r)) {
- `backend/internal/platform/authorization/super_admin.go` / `CanSelfReview` — SELF_REVIEW_POLICY: L12: ...= tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&amp;allowed)
- `backend/test/integration/finance_test.go` / `TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid` — RELATED_TEST_NOT_EXECUTED: L97: if _, err = finance.PostSupplierPayment(ctx, payment1.ID, payment1.Version, supervisorID, "fin-p2p-13"); err != nil {

Test: related source inspected, not executed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Void pembayaran supplier - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-ap-payment-void.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_ap_payment_void_input<br>Isi alasan void payment<br>userTask / Supervisor | frontend/features/finance/finance-workspace.tsx | FinancePanel | USER_INPUT_OR_VIEW | L230: ...=== "POSTED" &amp;&amp; &lt;button disabled={!voidReason} onClick={() =&gt; command(`${paymentPrefix}/${item.id}/void`, { reason: voidReason })}&gt;Void&lt;/button&gt;}<br>L218: ...wApprove &amp;&amp; kind === "customer" &amp;&amp; item.status === "DRAFT" &amp;&amp; &lt;button disabled={!voidReason} onClick={() =&gt; command(`${invoicePrefix}/${item.id}/void`, { reason: voidReason })}&gt;Void&lt;/button&gt;} |
| role_supervisor_l3_ap_payment_void_check<br>Lock payment POSTED dan invoice terkait; alasan ada<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierPayment | VALIDATION | L452: if reason == "" {<br>L462: ...ument.Total, &amp;document.Outstanding, &amp;document.Version); err != nil \|\| document.Status != "POSTED" { |
| role_supervisor_l3_ap_payment_void_effect0<br>Kembalikan outstanding; hitung ulang status invoice<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierPayment | PERSISTENCE_OR_STATE | L493: if _, err = tx.Exec(ctx, `UPDATE supplier_invoices SET outstanding=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1`, item.invoice, outstanding, status); err != nil {<br>L491: status = "POSTED" |
| role_supervisor_l3_ap_payment_void_effect1<br>Jika tidak lunas lagi: PO CLOSED ke RECEIVED; PR COMPLETED ke CONVERTED_TO_PO<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierPayment | PERSISTENCE_OR_STATE | L499: poTag, updateErr := tx.Exec(ctx, `UPDATE purchase_orders SET status='RECEIVED',version=version+1,updated_at=now() WHERE id=$1 AND status='CLOSED'`, item.source)<br>L508: err = tx.QueryRow(ctx, `UPDATE purchase_requests r SET status='CONVERTED_TO_PO',version=r.version+1,updated_at=now() FROM purchase_orders p WHERE p.id=$1 AND r.id=p.purchase_request_id AND r.status='COMPLETED' RETURNING r.id`, item.source).Scan(&amp;requestID) |
| role_supervisor_l3_ap_payment_void_effect2<br>Payment VOID; audit reversal alokasi; commit<br>serviceTask / Sistem Simple ERP | backend/internal/finance/adapter/postgres/store.go | Store.VoidSupplierPayment | AUDIT_AND_COMMIT | L517: if _, err = tx.Exec(ctx, `UPDATE supplier_payments SET status='VOID',void_reason=$2,voided_by=$3,voided_at=now(),version=version+1 WHERE id=$1`, id, reason, actor); err != nil {<br>L496: if err = audit(ctx, tx, actor, "REVERSE_SUPPLIER_PAYMENT_ALLOCATION", "ACT-FIN-SPAY-VOID", "SUPPLIER_INVOICE", item.invoice, item.status, status, reason, request); err != nil {<br>L523: if err = tx.Commit(ctx); err != nil { |
| role_supervisor_l3_ap_payment_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | Handler.voidSupplierPayment | RESPONSE_DISPATCH | L196: result(w, document, err, 200) |
| role_supervisor_l3_ap_payment_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | result | SUCCESS_ERROR_RESPONSE | L82: if err != nil {<br>L86: respond(w, status, document) |
| role_supervisor_l3_ap_payment_void_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/finance/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L78: w.WriteHeader(status)<br>L79: _ = json.NewEncoder(w).Encode(value) |

**Catatan role_supervisor_l3_ap_payment_void_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_ap_payment_void_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Alokasi historis dipertahankan; efek payment VOID dikecualikan dari hitungan pembayaran aktif.


**Catatan role_supervisor_l3_ap_payment_void_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/finance/adapter/http/handler.go` / `Handler.voidSupplierPayment` — API_SERVICE_BINDING: L195: document, err := h.Service.VoidSupplierPayment(r.Context(), chi.URLParam(r, "id"), value, actor(r), requestID(r))
- `backend/internal/finance/application/service.go` / `Service.VoidSupplierPayment` — SERVICE_STORE_BINDING: L102: return s.store.VoidSupplierPayment(c, id, reason, actor, request)
- `backend/internal/finance/adapter/http/handler.go` / `Handler.voidSupplierPayment` — ROLE_ACCESS_BOUNDARY: L188: if !requireApprove(w, r) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/finance/adapter/http/handler.go` / `requireApprove` — ROLE_CHECK_HELPER: L60: if authdomain.CanApprove(role(r)) {
- `backend/test/integration/finance_test.go` / `TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid` — RELATED_TEST_NOT_EXECUTED: L110: if _, err = finance.VoidSupplierPayment(ctx, payment2.ID, "incorrect transfer", adminID, "fin-p2p-16"); err != nil {

Test: related source inspected, not executed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Posting penerimaan dan stok masuk - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-gr-post.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_gr_post_input<br>Review dan post Goods Receipt<br>userTask / Supervisor | frontend/features/purchasing/goods-receipt-panel.tsx | GoodsReceiptPanel | USER_INPUT_OR_VIEW | L21: ... if (canApprove(session?.role)) void run(`/goods-receipts/${receiptId}/post`, { version: Number(version) }, "Goods receipt berhasil diposting."); }}&gt;&lt;strong&gt;Kelola Goods Receipt&lt;/strong&gt;&lt;label&gt;Goods Receipt ID&lt;input value={receiptId} onChange={(event) =&gt; setReceiptId(event.target.value)} required /&gt;&lt;/label&gt;&lt;label&gt;Versi... |
| role_supervisor_l3_gr_post_check<br>Lock GR/PO/lines; draft dan versi cocok; Supervisor bukan creator; Super Admin boleh self-post; qty tidak melebihi sisa<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.PostReceipt | VALIDATION | L316: if e != nil \|\| d.Status != "DRAFT" \|\| d.Version != version {<br>L316: if e != nil \|\| d.Status != "DRAFT" \|\| d.Version != version {<br>L320: allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)<br>L343: if l.qty &gt; l.remaining { |
| role_supervisor_l3_gr_post_effect0<br>Tambahkan saldo dan immutable stock movements<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.PostReceipt | PERSISTENCE_OR_STATE | L359: _, e = tx.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity)VALUES($1,$2,$3)ON CONFLICT(warehouse_id,product_id)DO UPDATE SET quantity=stock_balances.quantity+EXCLUDED.quantity,updated_at=now()`, warehouse, l.product, l.qty)<br>L363: _, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'GOODS_RECEIPT','GOODS_RECEIPT',$4,$5,$6)`, warehouse, l.product, l.qty, id, l.id, actor) |
| role_supervisor_l3_gr_post_effect1<br>Update GR POSTED; PO PARTIALLY_RECEIVED atau RECEIVED<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.PostReceipt | PERSISTENCE_OR_STATE | L384: _, e = tx.Exec(ctx, `UPDATE goods_receipts SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id)<br>L377: tag, e := tx.Exec(ctx, `UPDATE purchase_orders SET status=$1,version=version+1,updated_at=now() WHERE id=$2 AND status IN('ISSUED','PARTIALLY_RECEIVED')`, nextPOStatus, po) |
| role_supervisor_l3_gr_post_effect2<br>Audit PUR-GR-002; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.PostReceipt | AUDIT_AND_COMMIT | L388: if e = audit(ctx, tx, actor, "POST_GOODS_RECEIPT", "PUR-GR-002", "GOODS_RECEIPT", id, "DRAFT", "POSTED", "", request); e != nil {<br>L391: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_gr_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.postReceipt | SUCCESS_ERROR_RESPONSE | L223: respond(w, 409, map[string]string{"message": e.Error()})<br>L226: respond(w, 200, v) |
| role_supervisor_l3_gr_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_gr_post_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_gr_post_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_supervisor_l3_gr_post_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.postReceipt` — API_SERVICE_BINDING: L221: v, e := h.Service.PostReceipt(r.Context(), chi.URLParam(r, "id"), in.Version, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.PostReceipt` — SERVICE_STORE_BINDING: L80: return s.store.PostReceipt(ctx, id, version, actor, request)
- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.postReceipt` — ROLE_ACCESS_BOUNDARY: L211: if !domain.CanApprove(principal(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/platform/authorization/super_admin.go` / `CanSelfReview` — SELF_REVIEW_POLICY: L12: ...= tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&amp;allowed)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L125: if _, e = s.PostReceipt(ctx, gr.ID, gr.Version, operatorID, "r8-self"); e == nil {

Test: related source inspected, not executed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Reverse Goods Receipt yang sudah posted - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-gr-reverse.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_gr_reverse_input<br>Isi alasan reversal<br>userTask / Supervisor | frontend/features/purchasing/goods-receipt-panel.tsx | GoodsReceiptPanel | USER_INPUT_OR_VIEW | L21: ...n" disabled={!reason} onClick={() =&gt; run(`/goods-receipts/${receiptId}/reverse`, { reason }, "Goods receipt berhasil direverse.")}&gt;Reverse&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; run(`/goods-receipts/${receiptId}/cancel`, { reason }, "Goods receipt dibatalkan.")}&gt;Ca...<br>L21: ...on"&gt;Post&lt;/button&gt;}{canApprove(session?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; run(`/goods-receipts/${receiptId}/reverse`, { reason }, "Goods receipt berhasil direverse.")}&gt;Reverse&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; run(`/goods-receipts... |
| role_supervisor_l3_gr_reverse_check<br>Lock GR POSTED; tidak ada invoice non-VOID; stok akhir nonnegatif<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ReverseReceipt | VALIDATION | L399: if reason == "" {<br>L410: if e != nil \|\| d.Status != "POSTED" {<br>L414: ...EXISTS(SELECT 1 FROM supplier_invoices WHERE purchase_order_id=$1 AND status&lt;&gt;'VOID')`, po).Scan(&amp;invoiced); e != nil \|\| invoiced {<br>L442: if e != nil \|\| balance &lt; l.qty { |
| role_supervisor_l3_gr_reverse_effect0<br>Kurangi saldo/received qty; append movement inverse<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ReverseReceipt | PERSISTENCE_OR_STATE | L445: _, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, l.qty, warehouse, l.product)<br>L449: ..._type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'GOODS_RECEIPT_REVERSAL','GOODS_RECEIPT',$4,$5,$6,$7)`, warehouse, l.product, -l.qty, id, l.line, l.movement, actor)<br>L453: _, e = tx.Exec(ctx, `UPDATE purchase_order_lines pol SET received_quantity=received_quantity-$1 FROM goods_receipt_lines gl WHERE gl.id=$2 AND pol.id=gl.purchase_order_line_id`, l.qty, l.line) |
| role_supervisor_l3_gr_reverse_effect1<br>GR REVERSED; PO ISSUED atau PARTIALLY_RECEIVED<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ReverseReceipt | PERSISTENCE_OR_STATE | L458: _, e = tx.Exec(ctx, `UPDATE goods_receipts SET status='REVERSED',reason=$1,version=version+1 WHERE id=$2`, reason, id)<br>L462: ...rchase_order_lines WHERE purchase_order_id=$1 AND received_quantity&gt;0)THEN'PARTIALLY_RECEIVED'ELSE'ISSUED'END,version=version+1 WHERE id=$1`, po) |
| role_supervisor_l3_gr_reverse_effect2<br>Audit reversal; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.ReverseReceipt | AUDIT_AND_COMMIT | L466: if e = audit(ctx, tx, actor, "REVERSE_GOODS_RECEIPT", "ACT-PUR-GR-REVERSE", "GOODS_RECEIPT", id, "POSTED", "REVERSED", reason, request); e != nil {<br>L469: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_gr_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.reverseReceipt | SUCCESS_ERROR_RESPONSE | L241: respond(w, 409, map[string]string{"message": e.Error()})<br>L244: respond(w, 200, v) |
| role_supervisor_l3_gr_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_gr_reverse_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_gr_reverse_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Movement lama tidak diedit/dihapus. Tidak ada syarat reviewer berbeda dari creator untuk reversal.


**Catatan role_supervisor_l3_gr_reverse_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.reverseReceipt` — API_SERVICE_BINDING: L239: v, e := h.Service.ReverseReceipt(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.ReverseReceipt` — SERVICE_STORE_BINDING: L83: return s.store.ReverseReceipt(ctx, id, reason, actor, request)
- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.reverseReceipt` — ROLE_ACCESS_BOUNDARY: L229: if !domain.CanApprove(principal(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin

Test: dedicated test link not reviewed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Batalkan Purchase Order - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-po-cancel.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_po_cancel_input<br>Isi alasan pembatalan PO<br>userTask / Supervisor | frontend/features/purchasing/purchase-order-panel.tsx | PurchaseOrderPanel | USER_INPUT_OR_VIEW | L22: ...on" disabled={!reason} onClick={() =&gt; run(`/purchase-orders/${orderId}/cancel`, { reason }, "Purchase order dibatalkan.")}&gt;Cancel&lt;/button&gt;&lt;/span&gt;&lt;/form&gt;}<br>L22: ...tions"&gt;&lt;button className="button"&gt;Issue&lt;/button&gt;&lt;button type="button" disabled={!reason} onClick={() =&gt; run(`/purchase-orders/${orderId}/cancel`, { reason }, "Purchase order dibatalkan.")}&gt;Cancel&lt;/button&gt;&lt;/span&gt;&lt;/form&gt;} |
| role_supervisor_l3_po_cancel_check<br>PO DRAFT/ISSUED; tidak ada GR POSTED; alasan ada<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelOrder | VALIDATION | L513: if reason == "" {<br>L525: if e != nil \|\| (previous != "DRAFT" &amp;&amp; previous != "ISSUED") {<br>L528: ...STS(SELECT 1 FROM goods_receipts g WHERE g.purchase_order_id=p.id AND g.status='POSTED') RETURNING id,document_no,status,creator_id,version,purchase_request_id`, id, reason, previous).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Version, &amp;requestID) |
| role_supervisor_l3_po_cancel_effect0<br>Ubah PO CANCELLED; PR asal kembali APPROVED<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelOrder | PERSISTENCE_OR_STATE | L528: e = tx.QueryRow(ctx, `UPDATE purchase_orders p SET status='CANCELLED',reason=$2,version=version+1,updated_at=now() WHERE p.id=$1 AND p.status=$3 AND NOT EXISTS(SELECT 1 FROM goods_receipts g WHERE g.purchase_order_id=p.id AND g.status='POSTED') RETURNING id,document_no,status,creator_id,version,purchase_request_id`, id...<br>L532: tag, e := tx.Exec(ctx, `UPDATE purchase_requests SET status='APPROVED',version=version+1,updated_at=now() WHERE id=$1 AND status='CONVERTED_TO_PO'`, requestID) |
| role_supervisor_l3_po_cancel_effect1<br>Audit PO dan PR; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.CancelOrder | AUDIT_AND_COMMIT | L536: if e = audit(ctx, tx, actor, "CANCEL_PO", "ACT-PUR-PO-CANCEL", "PURCHASE_ORDER", id, previous, "CANCELLED", reason, request); e != nil {<br>L539: if e = audit(ctx, tx, actor, "REOPEN_PR_AFTER_PO_CANCEL", "ACT-PUR-PO-CANCEL", "PURCHASE_REQUEST", requestID, "CONVERTED_TO_PO", "APPROVED", reason, request); e != nil {<br>L542: e = tx.Commit(ctx) |
| role_supervisor_l3_po_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.cancelOrder | SUCCESS_ERROR_RESPONSE | L153: respond(w, 409, map[string]string{"message": e.Error()})<br>L156: respond(w, 200, v) |
| role_supervisor_l3_po_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_po_cancel_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_po_cancel_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. PO batal tetap tercatat. PR dapat dikonversi ke satu PO pengganti aktif; tidak ada owner check atau version input.


**Catatan role_supervisor_l3_po_cancel_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.cancelOrder` — API_SERVICE_BINDING: L151: v, e := h.Service.CancelOrder(r.Context(), chi.URLParam(r, "id"), in.Reason, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.CancelOrder` — SERVICE_STORE_BINDING: L89: return s.store.CancelOrder(c, id, reason, actor, r)
- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.cancelOrder` — ROLE_ACCESS_BOUNDARY: L143: if !domain.CanApprove(principal(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin

Test: dedicated test link not reviewed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Terbitkan Purchase Order - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-po-issue.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_po_issue_input<br>Review dan issue PO draft<br>userTask / Supervisor | frontend/features/purchasing/purchase-order-panel.tsx | PurchaseOrderPanel | USER_INPUT_OR_VIEW | L22: ...nt) =&gt; { event.preventDefault(); void run(`/purchase-orders/${orderId}/issue`, { version: Number(version) }, "Purchase order berhasil diterbitkan."); }}&gt;&lt;strong&gt;Review Purchase Order&lt;/strong&gt;&lt;label&gt;Purchase Order ID&lt;input value={orderId} onChange={(event) =&gt; setOrderId(event.target.value)} required /&gt;&lt;/label&gt;&lt;label&gt;Ver... |
| role_supervisor_l3_po_issue_check<br>PO DRAFT; version cocok; Supervisor bukan creator; Super Admin boleh self-issue<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.IssueOrder | VALIDATION | L249: ...ET status='ISSUED',version=version+1,updated_at=now() WHERE id=$1 AND status='DRAFT' AND version=$2 AND (creator_id&lt;&gt;$3 OR EXISTS(SELECT 1 FROM users WHERE id=$3 AND role='SUPER_ADMIN' AND status='ACTIVE')) RETURNING id,document_no,status,creator_id,version`, id, version, actor).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d...<br>L249: ...=$2 AND (creator_id&lt;&gt;$3 OR EXISTS(SELECT 1 FROM users WHERE id=$3 AND role='SUPER_ADMIN' AND status='ACTIVE')) RETURNING id,document_no,status,creator_id,version`, id, version, actor).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Version)<br>L249: ...version=version+1,updated_at=now() WHERE id=$1 AND status='DRAFT' AND version=$2 AND (creator_id&lt;&gt;$3 OR EXISTS(SELECT 1 FROM users WHERE id=$3 AND role='SUPER_ADMIN' AND status='ACTIVE')) RETURNING id,document_no,status,creator_id,version`, id, version, actor).Scan(&amp;d.ID, &amp;d.DocumentNo, &amp;d.Status, &amp;d.CreatorID, &amp;d.Vers... |
| role_supervisor_l3_po_issue_effect0<br>Update PO menjadi ISSUED<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.IssueOrder | PERSISTENCE_OR_STATE | L249: e = tx.QueryRow(ctx, `UPDATE purchase_orders SET status='ISSUED',version=version+1,updated_at=now() WHERE id=$1 AND status='DRAFT' AND version=$2 AND (creator_id&lt;&gt;$3 OR EXISTS(SELECT 1 FROM users WHERE id=$3 AND role='SUPER_ADMIN' AND status='ACTIVE')) RETURNING id,document_no,status,creator_id,version`, id, version, a... |
| role_supervisor_l3_po_issue_effect1<br>Audit PUR-PO-002; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.IssueOrder | AUDIT_AND_COMMIT | L253: if e = audit(ctx, tx, actor, "ISSUE_PO", "PUR-PO-002", "PURCHASE_ORDER", id, "DRAFT", "ISSUED", "", request); e != nil {<br>L256: e = tx.Commit(ctx) |
| role_supervisor_l3_po_issue_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.issue | SUCCESS_ERROR_RESPONSE | L187: respond(w, 409, map[string]string{"message": e.Error()})<br>L190: respond(w, 200, v) |
| role_supervisor_l3_po_issue_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_po_issue_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku.


**Catatan role_supervisor_l3_po_issue_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Tidak ada email atau pengiriman otomatis PO ke supplier.


**Catatan role_supervisor_l3_po_issue_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.issue` — API_SERVICE_BINDING: L185: v, e := h.Service.IssueOrder(r.Context(), chi.URLParam(r, "id"), in.Version, principal(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.IssueOrder` — SERVICE_STORE_BINDING: L74: return s.store.IssueOrder(ctx, id, version, actor, request)
- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.issue` — ROLE_ACCESS_BOUNDARY: L175: if !domain.CanApprove(principal(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L106: if _, e = s.IssueOrder(ctx, po.ID, po.Version, operatorID, "r6-self"); e == nil {

Test: related source inspected, not executed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Review Purchase Request - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/p2p/level-3/role-supervisor-l3-pr-review.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_pr_review_input<br>Pilih approve/reject; UI kirim alasan demo saat reject<br>userTask / Supervisor | frontend/features/purchasing/purchase-request-panel.tsx | PurchaseRequestPanel | USER_INPUT_OR_VIEW | L56: ...serId \|\| session?.role === "SUPER_ADMIN") &amp;&amp; &lt;&gt;&lt;button onClick={() =&gt; action(document, "approve")}&gt;Approve&lt;/button&gt;&lt;button onClick={() =&gt; action(document, "reject")}&gt;Reject&lt;/button&gt;&lt;/&gt;}<br>L56: ...&gt; action(document, "approve")}&gt;Approve&lt;/button&gt;&lt;button onClick={() =&gt; action(document, "reject")}&gt;Reject&lt;/button&gt;&lt;/&gt;} |
| role_supervisor_l3_pr_review_check<br>Lock PR; SUBMITTED; versi cocok; Supervisor bukan creator; Super Admin boleh self-review<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | VALIDATION | L154: if from != "SUBMITTED" {<br>L158: allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor) |
| role_supervisor_l3_pr_review_positive<br>Set PR APPROVED<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | BRANCH_OUTCOME | L163: d.Status = "APPROVED" |
| role_supervisor_l3_pr_review_negative<br>Set PR REJECTED dan alasan<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | BRANCH_OUTCOME | L175: d.Status = "REJECTED"<br>L166: if from != "SUBMITTED" \|\| reason == "" { |
| role_supervisor_l3_pr_review_effect1<br>Simpan reviewer; audit PUR-PR-003; commit<br>serviceTask / Sistem Simple ERP | backend/internal/purchasing/adapter/postgres/store.go | Store.TransitionRequest | AUDIT_AND_COMMIT | L187: tag, e := tx.Exec(ctx, `UPDATE purchase_requests SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject') THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version)<br>L153: trace = "PUR-PR-003"<br>L191: if e = audit(ctx, tx, actor, "PR_"+action, trace, "PURCHASE_REQUEST", id, from, d.Status, reason, request); e != nil {<br>L194: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_pr_review_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | Handler.prAction | SUCCESS_ERROR_RESPONSE | L137: respond(w, 409, map[string]string{"message": e.Error()})<br>L140: respond(w, 200, v) |
| role_supervisor_l3_pr_review_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/purchasing/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L73: w.WriteHeader(s)<br>L74: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_pr_review_input:** Request ini dijalankan oleh Supervisor; guard backend tetap berlaku. Panel PR tidak menyediakan input alasan reject; API tetap menerima reason.


**Catatan role_supervisor_l3_pr_review_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Bypass self-review Super Admin memeriksa role aktif di database; guard status/versi tetap berlaku.


**Catatan role_supervisor_l3_pr_review_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — API_SERVICE_BINDING: L134: v, e = h.Service.TransitionRequest(r.Context(), chi.URLParam(r, "id"), action, p.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/purchasing/application/service.go` / `Service.TransitionRequest` — SERVICE_STORE_BINDING: L68: return s.store.TransitionRequest(ctx, id, action, actor, reason, version, request)
- `backend/internal/purchasing/adapter/http/handler.go` / `Handler.prAction` — ROLE_ACCESS_BOUNDARY: L130: if (action == "approve" \|\| action == "reject") &amp;&amp; !domain.CanApprove(p.Role) \|\| (action != "approve" &amp;&amp; action != "reject" &amp;&amp; !domain.CanCreate(p.Role)) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/platform/authorization/super_admin.go` / `CanSelfReview` — SELF_REVIEW_POLICY: L12: ...= tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&amp;allowed)
- `backend/test/integration/operations_test.go` / `TestP2PPartialReceiptPostsImmutableSourceLinkedLedger` — RELATED_TEST_NOT_EXECUTED: L93: if _, e = s.TransitionRequest(ctx, pr.ID, "submit", operatorID, "", pr.Version, "r2"); e != nil {

Test: related source inspected, not executed.

4 event/gateway serta 11 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Batas penggunaan

- Single AI static source review; no independent review or runtime/role-session verification.
- Existing task-to-code relationships are not required-change labels or an exhaustive dependency graph.
- Gateway branches are preserved as BPMN control references; not all conditions have independent source mappings.
- Function ranges use declaration text, not AST; TypeScript component ranges may include nested functions.
- Related test source was inspected, not executed; absence of a link does not establish absence of tests.
- L0-L2 are navigation/decomposition references, not invented direct task-to-code mappings.
- BPMN labels and source evidence remain in their original language; no diagram/source is rewritten.
