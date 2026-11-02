# Mapping before: staff / master-data

Snapshot paket: `876d5d30117e798a8733986217f19434`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; working tree dirty: True.

Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.
JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.

## Lihat master data - Staff

BPMN: `docs/bpm-haqi/roles/staff/master-data/role-staff-l3-master-list.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_staff_l3_master_list_input<br>Pilih units, products, suppliers, customers, atau warehouses<br>userTask / Staff | frontend/features/master-data/master-data-panel.tsx | MasterDataPanel | USER_INPUT_OR_VIEW | L15: useEffect(() =&gt; { api&lt;{ data: RecordItem[] }&gt;(`/master-data/${resource}`).then((value) =&gt; setItems(value.data)).catch((cause: Error) =&gt; setError(cause.message)).finally(() =&gt; setBusy(false)); }, [resource, revision]); |
| role_staff_l3_master_list_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.List | READ_QUERY | L28: t, e := table(k)<br>L36: rows, e := s.Pool.Query(ctx, "SELECT id,"+code+",name,active,version FROM "+t+" ORDER BY "+code)<br>L36: ... s.Pool.Query(ctx, "SELECT id,"+code+",name,active,version FROM "+t+" ORDER BY "+code) |
| role_staff_l3_master_list_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | Handler.list | SUCCESS_ERROR_RESPONSE | L42: respond(w, 400, map[string]string{"message": e.Error()})<br>L45: respond(w, 200, map[string]any{"data": items}) |
| role_staff_l3_master_list_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_staff_l3_master_list_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.list` — API_SERVICE_BINDING: L40: items, e := h.Service.List(r.Context(), kind(chi.URLParam(r, "kind")))
- `backend/internal/masterdata/application/service.go` / `Service.List` — SERVICE_STORE_BINDING: L20: return s.store.List(ctx, k)
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L83: api.Mount("/master-data", authMiddleware.RequireAuth(masterHandler.Routes()))
- `backend/internal/masterdata/adapter/postgres/store.go` / `table` — ALLOWED_MASTER_KINDS: L14: case domain.Product:; L16: case domain.Supplier:; L18: case domain.Customer:; L20: case domain.Unit:; L22: case domain.Warehouse:; L25: return "", fmt.Errorf("invalid kind")

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
