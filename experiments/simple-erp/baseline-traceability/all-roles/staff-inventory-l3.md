# Mapping before: staff / inventory

Snapshot paket: `876d5d30117e798a8733986217f19434`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; working tree dirty: True.

Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.
JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.

## Batalkan adjustment draft atau approved - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-cancel.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_adj_cancel_input<br>Isi alasan pembatalan adjustment<br>userTask / Staff | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L19: ...= session?.userId &amp;&amp; session?.role !== "SUPER_ADMIN")} onClick={() =&gt; action("cancel")}&gt;Cancel&lt;/button&gt;}{canApprove(session?.role) &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reverse")}&gt;Reverse&lt;/button&gt;}&lt;/span&gt;&lt;/form&gt;}<br>L19: ...?.userId \|\| session?.role === "SUPER_ADMIN") &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reject")}&gt;Reject&lt;/button&gt;}{canCreate(session?.role) &amp;&amp; (!selected \|\| selected.status === "DRAFT" \|\| selected.status === "APPROVED") &amp;&amp; &lt;button type="button" disabled={!reason \|\| Boolean(selected &amp;&amp; selected.c... |
| role_staff_l3_adj_cancel_check<br>DRAFT atau APPROVED; versi cocok; creator atau Super Admin aktif; alasan ada<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | VALIDATION | L176: if e != nil \|\| (from != "DRAFT" &amp;&amp; from != "APPROVED") \|\| reason == "" \|\| (actor != d.CreatorID &amp;&amp; !admin) {<br>L176: ...!= nil \|\| (from != "DRAFT" &amp;&amp; from != "APPROVED") \|\| reason == "" \|\| (actor != d.CreatorID &amp;&amp; !admin) {<br>L162: if from != "SUBMITTED" \|\| reason == "" { |
| role_staff_l3_adj_cancel_effect0<br>Ubah adjustment CANCELLED<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | PERSISTENCE_OR_STATE | L179: d.Status = "CANCELLED"<br>L183: tag, e := tx.Exec(ctx, `UPDATE inventory_adjustments SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject')THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version) |
| role_staff_l3_adj_cancel_effect1<br>Audit ACT-INV-ADJ-CANCEL; commit<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | AUDIT_AND_COMMIT | L173: trace = "ACT-INV-ADJ-CANCEL"<br>L187: if e = audit(ctx, tx, actor, "ADJUSTMENT_"+action, trace, id, from, d.Status, reason, request); e != nil {<br>L190: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_adj_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.action | SUCCESS_ERROR_RESPONSE | L107: respond(w, 409, map[string]string{"message": e.Error()})<br>L110: respond(w, 200, v) |
| role_staff_l3_adj_cancel_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_adj_cancel_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_staff_l3_adj_cancel_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Jika stok berubah setelah APPROVED, Staff membatalkan dengan alasan, membuat draft baru dari snapshot terkini, lalu meminta approval ulang.


**Catatan role_staff_l3_adj_cancel_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — API_SERVICE_BINDING: L104: v, e = h.Service.Transition(r.Context(), chi.URLParam(r, "id"), a, actor.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.Transition` — SERVICE_STORE_BINDING: L51: return s.store.TransitionAdjustment(c, id, action, actor, reason, v, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — ROLE_ACCESS_BOUNDARY: L83: ...actor.Role) \|\| (a != "approve" &amp;&amp; a != "reject" &amp;&amp; a != "reverse" &amp;&amp; !domain.CanCreate(actor.Role)) {
- `backend/internal/auth/domain/role.go` / `CanCreate` — ROLE_POLICY: L14: return role == RoleStaff \|\| role == RoleSuperAdmin
- `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` — RELATED_TEST_NOT_EXECUTED: L283: adjustment, err = s.Transition(ctx, adjustment.ID, "submit", operatorID, "", adjustment.Version, "inventory-recovery-submit")

Test: related source inspected, not executed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Catat stock opname / adjustment - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-create.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_adj_create_input<br>UI: satu produk dan counted quantity<br>userTask / Staff | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L18: ...n-grid"&gt;{canCreate(session?.role) &amp;&amp; &lt;form className="operation-card" onSubmit={create}&gt;&lt;strong&gt;Buat Stock Adjustment&lt;/strong&gt;&lt;MasterDataSelect resource="warehouses" label="Warehouse" value={warehouseId} onChange={setWarehouseId} /&gt;&lt;MasterDataSelect resource="products" label="Product" value={productId} onChange={setPro...<br>L15: ...urn run("/inventory-adjustments", { warehouseId, lines: [{ productId, countedQuantity: Number(countedQuantity) }] }, "Stock adjustment draft berhasil dibuat."); } |
| role_staff_l3_adj_create_check<br>Warehouse/produk aktif; line ada; counted quantity nonnegatif<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.CreateAdjustment | VALIDATION | L77: if len(lines) == 0 {<br>L86: e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM warehouses WHERE id=$1 AND active)`, warehouse).Scan(&amp;active)<br>L100: if l.CountedQuantity &lt; 0 {<br>L103: e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id=$1 AND active)`, l.ProductID).Scan(&amp;active) |
| role_staff_l3_adj_create_effect0<br>Snapshot saldo; simpan counted quantity pada adjustment DRAFT<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.CreateAdjustment | PERSISTENCE_OR_STATE | L95: _, e = tx.Exec(ctx, `INSERT INTO inventory_adjustments(id,document_no,warehouse_id,creator_id,status)VALUES($1,$2,$3,$4,'DRAFT')`, id, no, warehouse, actor)<br>L107: ...RT INTO inventory_adjustment_lines(inventory_adjustment_id,product_id,snapshot_quantity,counted_quantity)VALUES($1,$2,COALESCE((SELECT quantity FROM stock_balances WHERE warehouse_id=$3 AND product_id=$2),0),$4)`, id, l.ProductID, warehouse, l.CountedQuantity) |
| role_staff_l3_adj_create_effect1<br>Audit INV-ADJ-001; commit<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.CreateAdjustment | AUDIT_AND_COMMIT | L112: if e = audit(ctx, tx, actor, "CREATE_ADJUSTMENT", "INV-ADJ-001", id, "", "DRAFT", "", request); e != nil {<br>L115: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_adj_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.create | SUCCESS_ERROR_RESPONSE | L75: respond(w, 409, map[string]string{"message": e.Error()})<br>L78: respond(w, 201, v) |
| role_staff_l3_adj_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_adj_create_input:** API menerima beberapa line; form saat ini hanya mengirim satu line.


**Catatan role_staff_l3_adj_create_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Tidak ada endpoint edit atau copy adjustment saat ini.


**Catatan role_staff_l3_adj_create_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.create` — API_SERVICE_BINDING: L73: v, e := h.Service.CreateAdjustment(r.Context(), in.WarehouseID, p(r).UserID, in.Lines, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.CreateAdjustment` — SERVICE_STORE_BINDING: L48: return s.store.CreateAdjustment(c, w, a, l, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.create` — ROLE_ACCESS_BOUNDARY: L62: if !domain.CanCreate(p(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanCreate` — ROLE_POLICY: L14: return role == RoleStaff \|\| role == RoleSuperAdmin
- `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` — RELATED_TEST_NOT_EXECUTED: L279: adjustment, err := s.CreateAdjustment(ctx, warehouseID, operatorID, []inventoryapp.AdjustmentLine{{ProductID: productID, CountedQuantity: 2}}, "inventory-recovery-create")

Test: related source inspected, not executed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Posting selisih stock adjustment - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-post.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_adj_post_input<br>Post adjustment approved<br>userTask / Staff | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L19: ...on&gt;}{canCreate(session?.role) &amp;&amp; &lt;button type="button" onClick={() =&gt; action("post")}&gt;Post&lt;/button&gt;}{canApprove(session?.role) &amp;&amp; selected?.status === "SUBMITTED" &amp;&amp; (selected.creatorId !== session?.userId \|\| session?.role === "SUPER_ADMIN") &amp;&amp; &lt;button type="button" disabled={!reason} onClick={() =&gt; action("reject")}&gt;R... |
| role_staff_l3_adj_post_check<br>Lock adjustment APPROVED dan saldo; versi dan snapshot cocok<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.PostAdjustment | VALIDATION | L205: if e != nil \|\| d.Status != "APPROVED" \|\| d.Version != version {<br>L205: if e != nil \|\| d.Status != "APPROVED" \|\| d.Version != version {<br>L236: if e != nil \|\| current != l.snapshot { |
| role_staff_l3_adj_post_effect0<br>Hitung counted minus snapshot; update saldo dan movement nonzero<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.PostAdjustment | PERSISTENCE_OR_STATE | L239: delta := l.counted - l.snapshot<br>L240: if delta == 0 {<br>L232: if _, e = tx.Exec(ctx, `INSERT INTO stock_balances(warehouse_id,product_id,quantity) VALUES($1,$2,0) ON CONFLICT DO NOTHING`, warehouse, l.product); e != nil {<br>L247: _, e = tx.Exec(ctx, `INSERT INTO stock_movements(warehouse_id,product_id,quantity,movement_type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'STOCK_ADJUSTMENT','INVENTORY_ADJUSTMENT',$4,$5,$6)`, warehouse, l.product, delta, id, l.id, actor) |
| role_staff_l3_adj_post_effect1<br>Ubah adjustment COMPLETED<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.PostAdjustment | PERSISTENCE_OR_STATE | L252: _, e = tx.Exec(ctx, `UPDATE inventory_adjustments SET status='COMPLETED',version=version+1,updated_at=now() WHERE id=$1`, id) |
| role_staff_l3_adj_post_effect2<br>Audit INV-ADJ-004 dan INV-ADJ-005; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.PostAdjustment | AUDIT_AND_COMMIT | L256: if e = audit(ctx, tx, actor, `POST_ADJUSTMENT_MOVEMENT`, `INV-ADJ-004`, id, `APPROVED`, `APPROVED`, ``, request); e != nil {<br>L259: if e = audit(ctx, tx, actor, `COMPLETE_ADJUSTMENT`, `INV-ADJ-005`, id, `APPROVED`, `COMPLETED`, ``, request); e != nil {<br>L262: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_adj_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.action | SUCCESS_ERROR_RESPONSE | L107: respond(w, 409, map[string]string{"message": e.Error()})<br>L110: respond(w, 200, v) |
| role_staff_l3_adj_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_adj_post_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_staff_l3_adj_post_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_staff_l3_adj_post_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — API_SERVICE_BINDING: L100: v, e = h.Service.Post(r.Context(), chi.URLParam(r, "id"), in.Version, actor.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.Post` — SERVICE_STORE_BINDING: L54: return s.store.PostAdjustment(c, id, v, a, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — ROLE_ACCESS_BOUNDARY: L83: ...actor.Role) \|\| (a != "approve" &amp;&amp; a != "reject" &amp;&amp; a != "reverse" &amp;&amp; !domain.CanCreate(actor.Role)) {
- `backend/internal/auth/domain/role.go` / `CanCreate` — ROLE_POLICY: L14: return role == RoleStaff \|\| role == RoleSuperAdmin
- `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` — RELATED_TEST_NOT_EXECUTED: L294: if _, err = s.Post(ctx, adjustment.ID, adjustment.Version, operatorID, "inventory-recovery-stale-post"); err == nil {

Test: related source inspected, not executed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Submit stock adjustment - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_adj_submit_input<br>Kirim adjustment untuk review<br>userTask / Staff | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L19: ...sName="operation-card" onSubmit={(event) =&gt; { event.preventDefault(); void action("submit"); }}&gt;&lt;strong&gt;Kelola Stock Adjustment&lt;/strong&gt;&lt;label&gt;Adjustment ID&lt;input value={adjustmentId} onChange={(event) =&gt; setAdjustmentId(event.target.value)} required /&gt;&lt;/label&gt;&lt;label&gt;Version&lt;input type="number" min="1" value={version} ... |
| role_staff_l3_adj_submit_check<br>Lock adjustment DRAFT; versi cocok<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | VALIDATION | L128: if e != nil \|\| d.Version != version {<br>L134: case "submit":<br>L135: if from != "DRAFT" { |
| role_staff_l3_adj_submit_effect0<br>Ubah adjustment SUBMITTED<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | PERSISTENCE_OR_STATE | L138: d.Status = "SUBMITTED"<br>L183: tag, e := tx.Exec(ctx, `UPDATE inventory_adjustments SET status=$1,reviewer_id=CASE WHEN $2 IN('approve','reject')THEN $3::uuid ELSE reviewer_id END,reason=NULLIF($4,''),version=version+1,updated_at=now() WHERE id=$5 AND version=$6`, d.Status, action, actor, reason, id, version) |
| role_staff_l3_adj_submit_effect1<br>Audit INV-ADJ-002; commit<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.TransitionAdjustment | AUDIT_AND_COMMIT | L132: trace := "INV-ADJ-002"<br>L187: if e = audit(ctx, tx, actor, "ADJUSTMENT_"+action, trace, id, from, d.Status, reason, request); e != nil {<br>L190: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_adj_submit_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.action | SUCCESS_ERROR_RESPONSE | L107: respond(w, 409, map[string]string{"message": e.Error()})<br>L110: respond(w, 200, v) |
| role_staff_l3_adj_submit_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_adj_submit_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_staff_l3_adj_submit_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_staff_l3_adj_submit_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — API_SERVICE_BINDING: L104: v, e = h.Service.Transition(r.Context(), chi.URLParam(r, "id"), a, actor.UserID, in.Reason, in.Version, middleware.GetReqID(r.Context()))
- `backend/internal/inventory/application/service.go` / `Service.Transition` — SERVICE_STORE_BINDING: L51: return s.store.TransitionAdjustment(c, id, action, actor, reason, v, r)
- `backend/internal/inventory/adapter/http/handler.go` / `Handler.action` — ROLE_ACCESS_BOUNDARY: L83: ...actor.Role) \|\| (a != "approve" &amp;&amp; a != "reject" &amp;&amp; a != "reverse" &amp;&amp; !domain.CanCreate(actor.Role)) {
- `backend/internal/auth/domain/role.go` / `CanCreate` — ROLE_POLICY: L14: return role == RoleStaff \|\| role == RoleSuperAdmin
- `backend/test/integration/operations_test.go` / `TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot` — RELATED_TEST_NOT_EXECUTED: L283: adjustment, err = s.Transition(ctx, adjustment.ID, "submit", operatorID, "", adjustment.Version, "inventory-recovery-submit")

Test: related source inspected, not executed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Posting delivery dan stok keluar - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-delivery-post.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_delivery_post_input<br>Post Delivery<br>userTask / Staff | frontend/features/sales/delivery-panel.tsx | DeliveryPanel | USER_INPUT_OR_VIEW | L19: ...t(); if (canCreate(session?.role)) void run(`/deliveries/${deliveryId}/post`, { version: Number(version) }, "Delivery berhasil diposting."); }}&gt;&lt;strong&gt;Kelola Delivery&lt;/strong&gt;&lt;label&gt;Delivery ID&lt;input value={deliveryId} onChange={(event) =&gt; setDeliveryId(event.target.value)} required /&gt;&lt;/label&gt;&lt;label&gt;Version&lt;input type... |
| role_staff_l3_delivery_post_check<br>Delivery DRAFT; versi cocok; SO APPROVED/PARTIALLY_DELIVERED; sisa dan stok cukup<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.PostDelivery | VALIDATION | L358: if e != nil \|\| d.Status != "DRAFT" \|\| d.Version != version {<br>L358: if e != nil \|\| d.Status != "DRAFT" \|\| d.Version != version {<br>L379: if l.qty &gt; l.remaining {<br>L393: if e != nil \|\| balance &lt; l.qty { |
| role_staff_l3_delivery_post_effect0<br>Kurangi stok; append stock movement keluar<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.PostDelivery | PERSISTENCE_OR_STATE | L396: _, e = tx.Exec(ctx, `UPDATE stock_balances SET quantity=quantity-$1,updated_at=now() WHERE warehouse_id=$2 AND product_id=$3`, l.qty, warehouse, l.product)<br>L404: ..._type,source_type,source_id,source_line_id,created_by)VALUES($1,$2,$3,'SALES_DELIVERY','DELIVERY',$4,$5,$6)`, warehouse, l.product, -l.qty, id, l.id, actor) |
| role_staff_l3_delivery_post_effect1<br>Delivery POSTED; SO PARTIALLY_DELIVERED atau DELIVERED<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.PostDelivery | PERSISTENCE_OR_STATE | L425: _, e = tx.Exec(ctx, `UPDATE deliveries SET status='POSTED',version=version+1,posted_at=now() WHERE id=$1`, id)<br>L418: tag, e := tx.Exec(ctx, `UPDATE sales_orders SET status=$1,version=version+1,updated_at=now() WHERE id=$2 AND status IN('APPROVED','PARTIALLY_DELIVERED')`, nextSOStatus, so) |
| role_staff_l3_delivery_post_effect2<br>Audit SAL-DLV-002; commit atomik<br>serviceTask / Sistem Simple ERP | backend/internal/sales/adapter/postgres/store.go | Store.PostDelivery | AUDIT_AND_COMMIT | L429: if e = audit(ctx, tx, actor, "POST_DELIVERY", "SAL-DLV-002", "DELIVERY", id, "DRAFT", "POSTED", "", request); e != nil {<br>L432: if e = tx.Commit(ctx); e != nil { |
| role_staff_l3_delivery_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/sales/adapter/http/handler.go | Handler.post | SUCCESS_ERROR_RESPONSE | L185: respond(w, 409, map[string]string{"message": e.Error()})<br>L188: respond(w, 200, v) |
| role_staff_l3_delivery_post_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/sales/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L47: w.WriteHeader(s)<br>L48: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_delivery_post_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_staff_l3_delivery_post_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Lock delivery, SO, lines dan saldo dalam transaksi. Posting hanya untuk delivery draft yang sudah dibuat di modul Sales.


**Catatan role_staff_l3_delivery_post_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/sales/adapter/http/handler.go` / `Handler.post` — API_SERVICE_BINDING: L183: v, e := h.Service.PostDelivery(r.Context(), chi.URLParam(r, "id"), in.Version, p(r).UserID, middleware.GetReqID(r.Context()))
- `backend/internal/sales/application/service.go` / `Service.PostDelivery` — SERVICE_STORE_BINDING: L75: return s.store.PostDelivery(c, id, v, actor, r)
- `backend/internal/sales/adapter/http/handler.go` / `Handler.post` — ROLE_ACCESS_BOUNDARY: L175: if !domain.CanCreate(p(r).Role) {
- `backend/internal/auth/domain/role.go` / `CanCreate` — ROLE_POLICY: L14: return role == RoleStaff \|\| role == RoleSuperAdmin

Test: dedicated test link not reviewed.

4 event/gateway serta 10 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat daftar stock adjustment - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-inv-adjustments.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_inv_adjustments_input<br>Buka daftar adjustment<br>userTask / Staff | frontend/features/inventory/inventory-adjustment-panel.tsx | InventoryAdjustmentPanel | USER_INPUT_OR_VIEW | L13: useEffect(() =&gt; { api&lt;{ data: WorkflowDocument[] }&gt;("/inventory-adjustments").then((value) =&gt; setItems(value.data)).catch((error: Error) =&gt; onNotice(error.message)); }, [revision, onNotice]); |
| role_staff_l3_inv_adjustments_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.ListAdjustments | READ_QUERY | L61: ...:= s.Pool.Query(ctx, `SELECT id,document_no,status,creator_id,version FROM inventory_adjustments ORDER BY created_at DESC`) |
| role_staff_l3_inv_adjustments_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.listAdjustments | SUCCESS_ERROR_RESPONSE | L25: respond(w, 500, map[string]string{"message": "Gagal membaca stock adjustment."})<br>L28: respond(w, 200, map[string]any{"data": v}) |
| role_staff_l3_inv_adjustments_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_inv_adjustments_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.listAdjustments` — API_SERVICE_BINDING: L23: v, e := h.Service.ListAdjustments(r.Context())
- `backend/internal/inventory/application/service.go` / `Service.ListAdjustments` — SERVICE_STORE_BINDING: L45: return s.store.ListAdjustments(c)
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L86: r.Use(authMiddleware.RequireAuth); L94: inventoryHandler.Register(r)

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat saldo per gudang dan produk - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-inv-balances.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_inv_balances_input<br>Buka halaman saldo stok<br>userTask / Staff | frontend/features/inventory/inventory-workspace.tsx | InventoryWorkspace | USER_INPUT_OR_VIEW | L13: useEffect(() =&gt; { Promise.all([api&lt;{ data: Balance[] }&gt;("/inventory/balances"), api&lt;{ data: Movement[] }&gt;("/inventory/movements")]).then(([balanceData, movementData]) =&gt; { setBalances(balanceData.data); setMovements(movementData.data); }).catch((error: Error) =&gt; setNotice(error.message)); }, [revision]); |
| role_staff_l3_inv_balances_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.Balances | READ_QUERY | L29: rows, e := s.Pool.Query(ctx, `SELECT warehouse_id,product_id,quantity FROM stock_balances ORDER BY warehouse_id,product_id`) |
| role_staff_l3_inv_balances_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.balances | SUCCESS_ERROR_RESPONSE | L48: respond(w, 500, map[string]string{"message": "Gagal membaca saldo."})<br>L51: respond(w, 200, map[string]any{"data": v}) |
| role_staff_l3_inv_balances_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_inv_balances_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/inventory/adapter/http/handler.go` / `Handler.balances` — API_SERVICE_BINDING: L46: v, e := h.Service.Balances(r.Context())
- `backend/internal/inventory/application/service.go` / `Service.Balances` — SERVICE_STORE_BINDING: L42: ...(s Service) Balances(c context.Context) ([]Balance, error)   { return s.store.Balances(c) }
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L86: r.Use(authMiddleware.RequireAuth); L94: inventoryHandler.Register(r)

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Telusuri movement stok - Staff

BPMN: `docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-inv-movements.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_inv_movements_input<br>Buka halaman movement stok<br>userTask / Staff | frontend/features/inventory/inventory-workspace.tsx | InventoryWorkspace | USER_INPUT_OR_VIEW | L13: ... data: Balance[] }&gt;("/inventory/balances"), api&lt;{ data: Movement[] }&gt;("/inventory/movements")]).then(([balanceData, movementData]) =&gt; { setBalances(balanceData.data); setMovements(movementData.data); }).catch((error: Error) =&gt; setNotice(error.message)); }, [revision]); |
| role_staff_l3_inv_movements_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/inventory/adapter/postgres/store.go | Store.Movements | READ_QUERY | L45: ...,warehouse_id,product_id,quantity,movement_type,source_type,source_id FROM stock_movements ORDER BY created_at DESC,id DESC LIMIT 500`) |
| role_staff_l3_inv_movements_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | Handler.movements | SUCCESS_ERROR_RESPONSE | L56: respond(w, 500, map[string]string{"message": "Gagal membaca ledger."})<br>L59: respond(w, 200, map[string]any{"data": v}) |
| role_staff_l3_inv_movements_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/inventory/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L42: w.WriteHeader(s)<br>L43: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_inv_movements_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


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
