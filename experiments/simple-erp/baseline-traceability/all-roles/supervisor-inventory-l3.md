# Mapping before: supervisor / inventory

Snapshot paket: `876d5d30117e798a8733986217f19434`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; working tree dirty: True.

Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.
JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.

## Reverse adjustment selesai - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-adj-reverse.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_adj_reverse_input<br>Isi alasan reversal adjustment<br>userTask / Supervisor | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L19: ...ion?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reverse")}&gt;Reverse&lt;/button&gt;}&lt;/span&gt;&lt;/form&gt;}<br>L19: ...?.userId \|\| session?.role === "SUPER_ADMIN") &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reject")}&gt;Reject&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; (!selected \|\| selected.status === "DRAFT" \|\| selected.status === "APPROVED") &amp;&amp; &lt;button type="button" disabled={!reason \|\| Boolean(selected &amp;&amp; selected.c... |
| role_supervisor_l3_adj_reverse_check<br>Lock adjustment COMPLETED; saldo setelah inverse nonnegatif<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.ReverseAdjustment | VALIDATION | L270: if reason == "" {<br>L281: if e != nil \|\| d.Status != "COMPLETED" {<br>L309: if e != nil \|\| balance-l.qty &lt; 0 { |
| role_supervisor_l3_adj_reverse_effect0<br>Append movement inverse; ubah saldo<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.ReverseAdjustment | PERSISTENCE_OR_STATE | L312: _, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, l.qty, warehouse, l.product)<br>L316: ..._type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'STOCK_ADJUSTMENT_REVERSAL','INVENTORY_ADJUSTMENT',$4,$5,$6,$7)`, warehouse, l.product, -l.qty, id, l.line, l.movement, actor) |
| role_supervisor_l3_adj_reverse_effect1<br>Adjustment REVERSED; audit; commit<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.ReverseAdjustment | AUDIT_AND_COMMIT | L321: _, e = tx.Exec(ctx, `UPDATE inventory_adjustments SET status='REVERSED',reason=$1,version=version+1,updated_at=now() WHERE id=$2`, reason, id)<br>L325: if e = audit(ctx, tx, actor, "REVERSE_ADJUSTMENT", "ACT-INV-ADJ-REVERSE", id, "COMPLETED", "REVERSED", reason, request); e != nil {<br>L328: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_adj_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.action | SUCCESS_ERROR_RESPONSE | L107: respond(w, 409, map[string]string{"message": e.Error()})<br>L110: respond(w, 200, v) |
| role_supervisor_l3_adj_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_adj_reverse_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_supervisor_l3_adj_reverse_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_supervisor_l3_adj_reverse_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — API_SERVICE_BINDING: L102: v, e = h.Service.Reverse(r.Context(), chi.URLParam(r, "id"), in.Reason, actor.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.Reverse` — SERVICE_STORE_BINDING: L57: return s.store.ReverseAdjustment(c, id, reason, a, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — ROLE_ACCESS_BOUNDARY: L83: if (a == "approve" \|\| a == "reject" \|\| a == "reverse") &amp;&amp; !domain.CanApprove(actor.Role) \|\| (a != "approve" &amp;&amp; a != "reject" &amp;&amp; a != "reverse" &amp;&amp; !domain.CanCreate(actor.Role)) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin

Test: dedicated test link not reviewed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Review stock adjustment - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-adj-review.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_adj_review_input<br>Approve atau reject adjustment<br>userTask / Supervisor | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L19: ...sion?.role === "SUPER_ADMIN") &amp;&amp; &lt;button type="button" onClick={() =&gt; action("approve")}&gt;Approve&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; &lt;button type="button" onClick={() =&gt; action("post")}&gt;Post&lt;/button&gt;}{canApprove(session?.role) &amp;&amp; selected?.status === "SUBMITTED" &amp;&amp; (selected.creatorId !== session?.userId \|\| session?....<br>L19: ...ER_ADMIN") &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reject")}&gt;Reject&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; (!selected \|\| selected.status === "DRAFT" \|\| selected.status === "APPROVED") &amp;&amp; &lt;button type="button" disabled={!reason \|\| Boolean(selected &amp;&amp; selected.creatorId !== session?.userId &amp;&amp; se... |
| role_supervisor_l3_adj_review_check<br>SUBMITTED; versi cocok; Supervisor bukan creator; Super Admin boleh self-review; approve perlu snapshot masih sama<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | VALIDATION | L141: if from != "SUBMITTED" {<br>L145: allowed, roleErr := authorization.CanSelfReview(ctx, tx, actor)<br>L156: if e != nil \|\| !snapshotCurrent { |
| role_supervisor_l3_adj_review_positive<br>Set adjustment APPROVED<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | BRANCH_OUTCOME | L159: d.Status = "APPROVED" |
| role_supervisor_l3_adj_review_negative<br>Set adjustment REJECTED dan alasan<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | BRANCH_OUTCOME | L171: d.Status = "REJECTED"<br>L162: if from != "SUBMITTED" \|\| reason == "" { |
| role_supervisor_l3_adj_review_effect1<br>Simpan reviewer; audit INV-ADJ-003; commit<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | AUDIT_AND_COMMIT | L183: tag, e := tx.Exec(ctx, `UPDATE inventory_adjustments SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject')THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version)<br>L140: trace = "INV-ADJ-003"<br>L187: if e = audit(ctx, tx, actor, "ADJUSTMENT_"+action, trace, id, from, d.Status, reason, request); e != nil {<br>L190: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_adj_review_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.action | SUCCESS_ERROR_RESPONSE | L107: respond(w, 409, map[string]string{"message": e.Error()})<br>L110: respond(w, 200, v) |
| role_supervisor_l3_adj_review_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_adj_review_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_supervisor_l3_adj_review_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Jika saldo berubah, approve ditolak; belum ada refresh snapshot atau revisi adjustment.


**Catatan role_supervisor_l3_adj_review_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — API_SERVICE_BINDING: L104: v, e = h.Service.Transition(r.Context(), chi.URLParam(r, "id"), a, actor.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.Transition` — SERVICE_STORE_BINDING: L51: return s.store.TransitionAdjustment(c, id, action, actor, reason, v, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — ROLE_ACCESS_BOUNDARY: L83: if (a == "approve" \|\| a == "reject" \|\| a == "reverse") &amp;&amp; !domain.CanApprove(actor.Role) \|\| (a != "approve" &amp;&amp; a != "reject" &amp;&amp; a != "reverse" &amp;&amp; !domain.CanCreate(actor.Role)) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/platform/authorization/super_admin.go` / `CanSelfReview` — SELF_REVIEW_POLICY: L12: ...= tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='SUPER_ADMIN' AND status='ACTIVE')`, actor).Scan(&amp;allowed)
- `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` — RELATED_TEST_NOT_EXECUTED: L283: adjustment, err = s.Transition(ctx, adjustment.ID, "submit", operatorID, "", adjustment.Version, "inventory-recovery-submit")

Test: related source inspected, not executed.

4 event/gateway serta 11 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Reverse delivery posted - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-delivery-reverse.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_delivery_reverse_input<br>Isi alasan reversal delivery<br>userTask / Supervisor | frontend/features/sales/delivery-panel.tsx | DeliveryPanel | USER_INPUT_OR_VIEW | L19: ...tton" disabled={!reason} onClick={() =&gt; run(`/deliveries/${deliveryId}/reverse`, { reason }, "Delivery berhasil direverse.")}&gt;Reverse&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; run(`/deliveries/${deliveryId}/cancel`, { reason }, "Delivery dibatalkan.")}&gt;Cancel&lt;/button&gt;... |
| role_supervisor_l3_delivery_reverse_check<br>Lock delivery POSTED; tidak ada invoice penjualan non-VOID<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.ReverseDelivery | VALIDATION | L440: if reason == "" {<br>L451: if e != nil \|\| d.Status != "POSTED" {<br>L455: ...ELECT EXISTS(SELECT 1 FROM sales_invoices WHERE sales_order_id=$1 AND status&lt;&gt;'VOID')`, so).Scan(&amp;invoiced); e != nil \|\| invoiced { |
| role_supervisor_l3_delivery_reverse_effect0<br>Tambah kembali stok; append movement inverse; kurangi delivered qty<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.ReverseDelivery | PERSISTENCE_OR_STATE | L481: if _, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity+$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, item.qty, warehouse, item.product); e != nil {<br>L484: ..._type,source_id,source_line_id,reversal_of,created_by)VALUES($1,$2,$3,'SALES_DELIVERY_REVERSAL','DELIVERY',$4,$5,$6,$7)`, warehouse, item.product, item.qty, id, item.line, item.movement, actor); e != nil {<br>L487: if _, e = tx.Exec(ctx, `UPDATE sales_order_lines sol SET delivered_quantity=delivered_quantity-$1 FROM delivery_lines dl WHERE dl.id=$2 AND sol.id=dl.sales_order_line_id`, item.qty, item.line); e != nil { |
| role_supervisor_l3_delivery_reverse_effect1<br>Delivery REVERSED; SO APPROVED atau PARTIALLY_DELIVERED<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.ReverseDelivery | PERSISTENCE_OR_STATE | L491: if _, e = tx.Exec(ctx, `UPDATE deliveries SET status='REVERSED',reason=$1,version=version+1 WHERE id=$2`, reason, id); e != nil {<br>L494: ...M sales_order_lines WHERE sales_order_id=$1 AND delivered_quantity&gt;0) THEN 'PARTIALLY_DELIVERED' ELSE 'APPROVED' END,version=version+1 WHERE id=$1`, so); e != nil { |
| role_supervisor_l3_delivery_reverse_effect2<br>Audit reversal; commit<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.ReverseDelivery | AUDIT_AND_COMMIT | L497: if e = audit(ctx, tx, actor, "REVERSE_DELIVERY", "ACT-SAL-DLV-REVERSE", "DELIVERY", id, "POSTED", "REVERSED", reason, request); e != nil {<br>L500: if e = tx.Commit(ctx); e != nil { |
| role_supervisor_l3_delivery_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/sales/adapter/http/handler.go | Handler.reverse | SUCCESS_ERROR_RESPONSE | L201: respond(w, 409, map[string]string{"message": e.Error()})<br>L204: respond(w, 200, v) |
| role_supervisor_l3_delivery_reverse_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/sales/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L47: w.WriteHeader(s)<br>L48: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_delivery_reverse_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_supervisor_l3_delivery_reverse_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_supervisor_l3_delivery_reverse_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/sales/adapter/http/handler.go` / `Handler.reverse` — API_SERVICE_BINDING: L199: v, e := h.Service.ReverseDelivery(r.Context(), chi.URLParam(r, "id"), in.Reason, p(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/sales/application/service.go` / `Service.ReverseDelivery` — SERVICE_STORE_BINDING: L78: return s.store.ReverseDelivery(c, id, reason, actor, r)
- `backend/internal/sales/adapter/http/handler.go` / `Handler.reverse` — ROLE_ACCESS_BOUNDARY: L191: if !domain.CanApprove(p(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin

Test: dedicated test link not reviewed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat daftar stock adjustment - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-inv-adjustments.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_inv_adjustments_input<br>Buka daftar adjustment<br>userTask / Supervisor | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L13: useEffect(() =&gt; { api&lt;{ data: WorkflowDocument[] }&gt;("/inventory-adjustments").then((value) =&gt; setItems(value.data)).catch((error: Error) =&gt; onNotice(error.message)); }, [revision, onNotice]); |
| role_supervisor_l3_inv_adjustments_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.ListAdjustments | READ_QUERY | L61: ...:= s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM inventory_adjustments ORDER BY created_at DESC`) |
| role_supervisor_l3_inv_adjustments_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.listAdjustments | SUCCESS_ERROR_RESPONSE | L25: respond(w, 500, map[string]string{"message": "Gagal membaca stock adjustment."})<br>L28: respond(w, 200, map[string]any{"data": v}) |
| role_supervisor_l3_inv_adjustments_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_inv_adjustments_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.listAdjustments` — API_SERVICE_BINDING: L23: v, e := h.Service.ListAdjustments(r.Context())
- `backend/internal/inventory/application/service.go` / `Service.ListAdjustments` — SERVICE_STORE_BINDING: L45: return s.store.ListAdjustments(c)
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L86: r.Use(authMiddleware.RequireAuth); L94: inventoryHandler.Register(r)

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat saldo per gudang dan produk - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-inv-balances.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_inv_balances_input<br>Buka halaman saldo stok<br>userTask / Supervisor | frontend/features/inventory/inventory-workspace.tsx | InventoryWorkspace | USER_INPUT_OR_VIEW | L13: useEffect(() =&gt; { Promise.all([api&lt;{ data: Balance[] }&gt;("/inventory/balances"), api&lt;{ data: Movement[] }&gt;("/inventory/movements")]).then(([balanceData, movementData]) =&gt; { setBalances(balanceData.data); setMovements(movementData.data); }).catch((error: Error) =&gt; setNotice(error.message)); }, [revision]); |
| role_supervisor_l3_inv_balances_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.Balances | READ_QUERY | L29: rows, e := s.Pool.Query(ctx, `SELECT warehouse_id,product_id,quantity FROM stock_balances ORDER BY warehouse_id,product_id`) |
| role_supervisor_l3_inv_balances_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.balances | SUCCESS_ERROR_RESPONSE | L48: respond(w, 500, map[string]string{"message": "Gagal membaca saldo."})<br>L51: respond(w, 200, map[string]any{"data": v}) |
| role_supervisor_l3_inv_balances_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_inv_balances_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.balances` — API_SERVICE_BINDING: L46: v, e := h.Service.Balances(r.Context())
- `backend/internal/inventory/application/service.go` / `Service.Balances` — SERVICE_STORE_BINDING: L42: ...(s Service) Balances(c context.Context) ([]Balance, error)   { return s.store.Balances(c) }
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L86: r.Use(authMiddleware.RequireAuth); L94: inventoryHandler.Register(r)

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Telusuri movement stok - Supervisor

BPMN: `docs/bpm-haqi/roles/supervisor/inventory/level-3/role-supervisor-l3-inv-movements.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_supervisor_l3_inv_movements_input<br>Buka halaman movement stok<br>userTask / Supervisor | frontend/features/inventory/inventory-workspace.tsx | InventoryWorkspace | USER_INPUT_OR_VIEW | L13: ... data: Balance[] }&gt;("/inventory/balances"), api&lt;{ data: Movement[] }&gt;("/inventory/movements")]).then(([balanceData, movementData]) =&gt; { setBalances(balanceData.data); setMovements(movementData.data); }).catch((error: Error) =&gt; setNotice(error.message)); }, [revision]); |
| role_supervisor_l3_inv_movements_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.Movements | READ_QUERY | L45: ...,warehouse_id,product_id,quantity,movement_type,source_type,source_id FROM stock_movements ORDER BY created_at DESC,id DESC LIMIT 500`) |
| role_supervisor_l3_inv_movements_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.movements | SUCCESS_ERROR_RESPONSE | L56: respond(w, 500, map[string]string{"message": "Gagal membaca ledger."})<br>L59: respond(w, 200, map[string]any{"data": v}) |
| role_supervisor_l3_inv_movements_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_supervisor_l3_inv_movements_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.movements` — API_SERVICE_BINDING: L54: v, e := h.Service.Movements(r.Context())
- `backend/internal/inventory/application/service.go` / `Service.Movements` — SERVICE_STORE_BINDING: L43: ...(s Service) Movements(c context.Context) ([]Movement, error) { return s.store.Movements(c) }
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L86: r.Use(authMiddleware.RequireAuth); L94: inventoryHandler.Register(r)

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Batas penggunaan

- Single AI static source review; no independent review or runtime/role-session verification.
- Existing task-to-code relationships are not required-change labels or an exhaustive dependency graph.
- Gateway branches are preserved as BPMN control references; not all conditions have independent source mappings.
- Function ranges use declaration text, not AST; TypeScript component ranges may include nested functions.
- Related test source was inspected, not executed; absence of a link does not establish absence of tests.
- L0-L2 are navigation/decomposition references, not invented direct task-to-code mappings.
- BPMN labels and source evidence remain in their original language; no diagram/source is rewritten.
