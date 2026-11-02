# Mapping before: super-admin / master-data

Snapshot paket: `876d5d30117e798a8733986217f19434`. Git HEAD: `c9344971a573d5c44d9d4e58f23657eda4d18a86`; working tree dirty: True.

Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.
JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.

## Buat master data - Super Admin

BPMN: `docs/bpm-haqi/roles/super-admin/master-data-konfigurasi/level-3/role-super-admin-l3-master-create.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_super_admin_l3_master_create_input<br>Pilih jenis; isi kode dan nama<br>userTask / Super Admin | frontend/features/master-data/master-data-panel.tsx | MasterDataPanel | USER_INPUT_OR_VIEW | L18: ...}&gt;{value}&lt;/button&gt;)}&lt;/div&gt;{canManage &amp;&amp; &lt;form className="inline-form" onSubmit={create}&gt;&lt;input aria-label="Kode" placeholder="Kode" value={code} onChange={(event) =&gt; setCode(event.target.value)} required /&gt;&lt;input aria-label="Nama" placeholder="Nama" value={name} onChange={(event) =&gt; setName(event.target.value)} require...<br>L16: ...; try { await api(`/master-data/${resource}`, { method: "POST", body: JSON.stringify({ code, name }) }); setCode(""); setName(""); setRevision((value) =&gt; value + 1); } catch (cause) { setError(cause instanceof Error ? cause.message : "Data gagal disimpan."); } } |
| role_super_admin_l3_master_create_check<br>Kind di whitelist; input valid; product perlu unit PCS aktif; constraint database<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.Create | VALIDATION | L52: t, e := table(r.Kind)<br>L62: ...INTO products(id,sku,name,unit_id)SELECT $1,$2,$3,id FROM units WHERE code='PCS' AND active`, r.ID, r.Code, r.Name)<br>L66: if tag.RowsAffected() != 1 { |
| role_super_admin_l3_master_create_check<br>Kind di whitelist; input valid; product perlu unit PCS aktif; constraint database<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/domain/entity.go | NewRecord | INPUT_VALIDATION | L29: return Record{}, errors.New("invalid kind")<br>L33: if code == "" \|\| name == "" { |
| role_super_admin_l3_master_create_effect0<br>Insert unit, product, supplier, customer atau warehouse<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.Create | PERSISTENCE_OR_STATE | L62: tag, insertErr := tx.Exec(ctx, `INSERT INTO products(id,sku,name,unit_id)SELECT $1,$2,$3,id FROM units WHERE code='PCS' AND active`, r.ID, r.Code, r.Name)<br>L70: _, e = tx.Exec(ctx, "INSERT INTO "+t+"(id,code,name) VALUES($1,$2,$3)", r.ID, r.Code, r.Name) |
| role_super_admin_l3_master_create_effect1<br>Audit CREATE_MASTER_DATA; commit<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.Create | AUDIT_AND_COMMIT | L75: ...,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'CREATE_MASTER_DATA','CAP-MDM-001',$2,$3,'ACTIVE',$4)`, actor, string(r.Kind), r.ID, request)<br>L79: return tx.Commit(ctx) |
| role_super_admin_l3_master_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | Handler.create | SUCCESS_ERROR_RESPONSE | L64: respond(w, 409, map[string]string{"message": e.Error()})<br>L67: respond(w, 201, item) |
| role_super_admin_l3_master_create_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_super_admin_l3_master_create_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_super_admin_l3_master_create_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Untuk product, INSERT SELECT unit code='PCS' aktif dan RowsAffected wajib 1. Tidak ada endpoint edit nama atau hapus.


**Catatan role_super_admin_l3_master_create_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.create` — API_SERVICE_BINDING: L62: item, e := h.Service.Create(r.Context(), kind(chi.URLParam(r, "kind")), in.Code, in.Name, p.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/masterdata/application/service.go` / `Service.Create` — SERVICE_STORE_BINDING: L28: err = s.store.Create(ctx, r, actor, request)
- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.create` — ROLE_ACCESS_BOUNDARY: L58: if !authdomain.CanApprove(p.Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/masterdata/adapter/postgres/store.go` / `table` — ALLOWED_MASTER_KINDS: L14: case domain.Product:; L16: case domain.Supplier:; L18: case domain.Customer:; L20: case domain.Unit:; L22: case domain.Warehouse:; L25: return "", fmt.Errorf("invalid kind")

Test: dedicated test link not reviewed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat master data - Super Admin

BPMN: `docs/bpm-haqi/roles/super-admin/master-data-konfigurasi/level-3/role-super-admin-l3-master-list.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_super_admin_l3_master_list_input<br>Pilih units, products, suppliers, customers, atau warehouses<br>userTask / Super Admin | frontend/features/master-data/master-data-panel.tsx | MasterDataPanel | USER_INPUT_OR_VIEW | L15: useEffect(() =&gt; { api&lt;{ data: RecordItem[] }&gt;(`/master-data/${resource}`).then((value) =&gt; setItems(value.data)).catch((cause: Error) =&gt; setError(cause.message)).finally(() =&gt; setBusy(false)); }, [resource, revision]); |
| role_super_admin_l3_master_list_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.List | READ_QUERY | L28: t, e := table(k)<br>L36: rows, e := s.Pool.Query(ctx, "SELECT id,"+code+",name,active,version FROM "+t+" ORDER BY "+code)<br>L36: ... s.Pool.Query(ctx, "SELECT id,"+code+",name,active,version FROM "+t+" ORDER BY "+code) |
| role_super_admin_l3_master_list_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | Handler.list | SUCCESS_ERROR_RESPONSE | L42: respond(w, 400, map[string]string{"message": e.Error()})<br>L45: respond(w, 200, map[string]any{"data": items}) |
| role_super_admin_l3_master_list_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_super_admin_l3_master_list_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.list` — API_SERVICE_BINDING: L40: items, e := h.Service.List(r.Context(), kind(chi.URLParam(r, "kind")))
- `backend/internal/masterdata/application/service.go` / `Service.List` — SERVICE_STORE_BINDING: L20: return s.store.List(ctx, k)
- `backend/cmd/api/main.go` / `main` — AUTHENTICATION_BOUNDARY: L83: api.Mount("/master-data", authMiddleware.RequireAuth(masterHandler.Routes()))
- `backend/internal/masterdata/adapter/postgres/store.go` / `table` — ALLOWED_MASTER_KINDS: L14: case domain.Product:; L16: case domain.Supplier:; L18: case domain.Customer:; L20: case domain.Unit:; L22: case domain.Warehouse:; L25: return "", fmt.Errorf("invalid kind")

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Aktif/nonaktifkan master data - Super Admin

BPMN: `docs/bpm-haqi/roles/super-admin/master-data-konfigurasi/level-3/role-super-admin-l3-master-status.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_super_admin_l3_master_status_input<br>Pilih master dan status aktif<br>userTask / Super Admin | frontend/features/master-data/master-data-panel.tsx | MasterDataPanel | USER_INPUT_OR_VIEW | L18: ...ll&gt;&lt;span className="row-actions"&gt;{canManage &amp;&amp; &lt;button onClick={() =&gt; toggle(item)}&gt;{item.active ? "Nonaktifkan" : "Aktifkan"}&lt;/button&gt;}&lt;/span&gt;&lt;/div&gt;)}&lt;/div&gt;&lt;/div&gt;;<br>L17: ...tus`, { method: "PATCH", body: JSON.stringify({ active: !item.active, version: item.version }) }); setRevision((value) =&gt; value + 1); } catch (cause) { setError(cause instanceof Error ? cause.message : "Status gagal diubah."); } } |
| role_super_admin_l3_master_status_check<br>Kind valid; id dan version cocok<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.SetActive | VALIDATION | L82: t, e := table(k)<br>L95: if tag.RowsAffected() != 1 { |
| role_super_admin_l3_master_status_effect0<br>Update active dan version<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.SetActive | PERSISTENCE_OR_STATE | L91: tag, e := tx.Exec(ctx, "UPDATE "+t+" SET active=$1,version=version+1,updated_at=now() WHERE id=$2 AND version=$3", active, id, version)<br>L91: ...c(ctx, "UPDATE "+t+" SET active=$1,version=version+1,updated_at=now() WHERE id=$2 AND version=$3", active, id, version) |
| role_super_admin_l3_master_status_effect1<br>Audit SET_MASTER_DATA_STATUS; commit<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/store.go | Store.SetActive | AUDIT_AND_COMMIT | L98: ...,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'SET_MASTER_DATA_STATUS','CAP-MDM-001',$2,$3,$4,$5)`, actor, string(k), id, map[bool]string{true: "ACTIVE", false: "INACTIVE"}[active], request)<br>L102: return tx.Commit(ctx) |
| role_super_admin_l3_master_status_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | Handler.status | SUCCESS_ERROR_RESPONSE | L89: respond(w, 409, map[string]string{"message": e.Error()})<br>L92: w.WriteHeader(204) |
| role_super_admin_l3_master_status_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_super_admin_l3_master_status_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_super_admin_l3_master_status_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request.


**Catatan role_super_admin_l3_master_status_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.status` — API_SERVICE_BINDING: L87: e = h.Service.SetActive(r.Context(), kind(chi.URLParam(r, "kind")), chi.URLParam(r, "id"), in.Active, in.Version, p.UserID, middleware.GetReqID(r.Context()))
- `backend/internal/masterdata/application/service.go` / `Service.SetActive` — SERVICE_STORE_BINDING: L32: return s.store.SetActive(ctx, k, id, active, version, actor, request)
- `backend/internal/masterdata/adapter/http/handler.go` / `Handler.status` — ROLE_ACCESS_BOUNDARY: L83: if !authdomain.CanApprove(p.Role) {
- `backend/internal/auth/domain/role.go` / `CanApprove` — ROLE_POLICY: L18: return role == RoleSupervisor \|\| role == RoleSuperAdmin
- `backend/internal/masterdata/adapter/postgres/store.go` / `table` — ALLOWED_MASTER_KINDS: L14: case domain.Product:; L16: case domain.Supplier:; L18: case domain.Customer:; L20: case domain.Unit:; L22: case domain.Warehouse:; L25: return "", fmt.Errorf("invalid kind")

Test: dedicated test link not reviewed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Lihat pengaturan organisasi - Super Admin

BPMN: `docs/bpm-haqi/roles/super-admin/master-data-konfigurasi/level-3/role-super-admin-l3-settings-get.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_super_admin_l3_settings_get_input<br>Buka halaman Settings<br>userTask / Super Admin | frontend/features/settings/settings-form.tsx | SettingsForm | USER_INPUT_OR_VIEW | L3: ...useState(empty);const[message,setMessage]=useState("");useEffect(()=&gt;{api&lt;Settings&gt;("/settings/organization").then(setValue).catch(e=&gt;setMessage(e.message))},[]);async function save(e:React.FormEvent){e.preventDefault();try{await api("/settings/organization",{method:"PATCH",body:JSON.stringify(value)});setMessage("Peng... |
| role_super_admin_l3_settings_get_read<br>Autentikasi dan baca data<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/settings.go | SettingsStore.Get | READ_QUERY | L12: e = s.Pool.QueryRow(ctx, `SELECT organization_name,currency,timezone,default_warehouse_id FROM organization_settings WHERE singleton`).Scan(&amp;v.OrganizationName, &amp;v.Currency, &amp;v.Timezone, &amp;v.DefaultWarehouseID)<br>L12: ...ame,currency,timezone,default_warehouse_id FROM organization_settings WHERE singleton`).Scan(&amp;v.OrganizationName, &amp;v.Currency, &amp;v.Timezone, &amp;v.DefaultWarehouseID) |
| role_super_admin_l3_settings_get_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/settings.go | SettingsHandler.ServeHTTP | SUCCESS_ERROR_RESPONSE | L19: respond(w, 500, map[string]string{"message": "Pengaturan tidak tersedia."})<br>L22: respond(w, 200, v) |
| role_super_admin_l3_settings_get_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_super_admin_l3_settings_get_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/settings.go` / `SettingsHandler.ServeHTTP` — API_SERVICE_BINDING: L17: v, e := h.Service.Get(r.Context())
- `backend/internal/masterdata/application/settings.go` / `SettingsService.Get` — SERVICE_STORE_BINDING: L16: return s.store.Get(ctx)
- `backend/cmd/api/main.go` / `main` — ROLE_ACCESS_BOUNDARY: L84: api.Handle("/settings/organization", authMiddleware.RequireRole(domain.RoleSuperAdmin, settingsHandler))

Test: dedicated test link not reviewed.

4 event/gateway serta 7 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Ubah pengaturan organisasi - Super Admin

BPMN: `docs/bpm-haqi/roles/super-admin/master-data-konfigurasi/level-3/role-super-admin-l3-settings-update.bpmn`

| Task / actor | File | Function | Hubungan | Bukti source |
|---|---|---|---|---|
| role_super_admin_l3_settings_update_input<br>Isi organisasi, currency, timezone, warehouse<br>userTask / Super Admin | frontend/features/settings/settings-form.tsx | SettingsForm | USER_INPUT_OR_VIEW | L3: ...FormEvent){e.preventDefault();try{await api("/settings/organization",{method:"PATCH",body:JSON.stringify(value)});setMessage("Pengaturan tersimpan dan tercatat di audit trail.")}catch(e){setMessage(e instanceof Error?e.message:"Gagal menyimpan.")}}return &lt;form className="settings-card"onSubmit={save}&gt;&lt;label&gt;Nama organi...<br>L3: ...?e.message:"Gagal menyimpan.")}}return &lt;form className="settings-card"onSubmit={save}&gt;&lt;label&gt;Nama organisasi&lt;input value={value.organizationName}onChange={e=&gt;setValue({...value,organizationName:e.target.value})}/&gt;&lt;/label&gt;&lt;label&gt;Mata uang&lt;input value={value.currency}maxLength={3}onChange={e=&gt;setValue({...value,currency:... |
| role_super_admin_l3_settings_update_check<br>Currency 3 karakter; timezone dan warehouse terisi<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/settings.go | SettingsStore.Update | VALIDATION | L25: if tag.RowsAffected() != 1 { |
| role_super_admin_l3_settings_update_check<br>Currency 3 karakter; timezone dan warehouse terisi<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/domain/settings.go | OrganizationSettings.Validate | INPUT_VALIDATION | L16: if len(s.Currency) != 3 \|\| strings.TrimSpace(s.Timezone) == "" \|\| strings.TrimSpace(s.DefaultWarehouseID) == "" {<br>L16: if len(s.Currency) != 3 \|\| strings.TrimSpace(s.Timezone) == "" \|\| strings.TrimSpace(s.DefaultWarehouseID) == "" {<br>L16: if len(s.Currency) != 3 \|\| strings.TrimSpace(s.Timezone) == "" \|\| strings.TrimSpace(s.DefaultWarehouseID) == "" { |
| role_super_admin_l3_settings_update_effect0<br>Update singleton organization_settings<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/settings.go | SettingsStore.Update | PERSISTENCE_OR_STATE | L21: tag, e := tx.Exec(ctx, `UPDATE organization_settings SET organization_name=$1,currency=$2,timezone=$3,default_warehouse_id=$4,updated_by=$5,updated_at=now() WHERE singleton`, v.OrganizationName, v.Currency, v.Timezone, v.DefaultWarehouseID, actor)<br>L21: ...$2,timezone=$3,default_warehouse_id=$4,updated_by=$5,updated_at=now() WHERE singleton`, v.OrganizationName, v.Currency, v.Timezone, v.DefaultWarehouseID, actor) |
| role_super_admin_l3_settings_update_effect1<br>Audit UPDATE_SETTINGS; commit<br>serviceTask / Sistem Simple ERP | backend/internal/masterdata/adapter/postgres/settings.go | SettingsStore.Update | AUDIT_AND_COMMIT | L28: ...,action,trace_id,entity_type,entity_id,new_state,request_id)VALUES($1,'UPDATE_SETTINGS','CAP-MDM-001','ORGANIZATION_SETTINGS','singleton','UPDATED',$2)`, actor, request)<br>L32: return tx.Commit(ctx) |
| role_super_admin_l3_settings_update_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/settings.go | SettingsHandler.ServeHTTP | SUCCESS_ERROR_RESPONSE | L31: respond(w, 409, map[string]string{"message": e.Error()})<br>L22: respond(w, 200, v) |
| role_super_admin_l3_settings_update_result_activity<br>Kirim respons hasil aksi (sukses atau gagal)<br>sendTask / Sistem Simple ERP | backend/internal/masterdata/adapter/http/handler.go | respond | HTTP_JSON_RESPONSE | L96: w.WriteHeader(s)<br>L97: _ = json.NewEncoder(w).Encode(v) |

**Catatan role_super_admin_l3_settings_update_input:** Aktor alternatif pada lane, bukan dua persetujuan.


**Catatan role_super_admin_l3_settings_update_check:** Setelah kontrol akses HTTP; lihat l3-kontrol-request. Warehouse mengikuti FK; validasi domain tidak memeriksa timezone IANA, kode ISO atau warehouse aktif. Baca melalui GET pada route yang sama.


**Catatan role_super_admin_l3_settings_update_result_activity:** Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.


Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):

- `backend/internal/masterdata/adapter/http/settings.go` / `SettingsHandler.ServeHTTP` — API_SERVICE_BINDING: L30: if e := h.Service.Update(r.Context(), v, p.UserID, middleware.GetReqID(r.Context())); e != nil {
- `backend/internal/masterdata/application/settings.go` / `SettingsService.Update` — SERVICE_STORE_BINDING: L22: return s.store.Update(ctx, v, actor, request)
- `backend/cmd/api/main.go` / `main` — ROLE_ACCESS_BOUNDARY: L84: api.Handle("/settings/organization", authMiddleware.RequireRole(domain.RoleSuperAdmin, settingsHandler))

Test: dedicated test link not reviewed.

4 event/gateway serta 9 flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.

## Batas penggunaan

- Single AI static source review; no independent review or runtime/role-session verification.
- Existing task-to-code relationships are not required-change labels or an exhaustive dependency graph.
- Gateway branches are preserved as BPMN control references; not all conditions have independent source mappings.
- Function ranges use declaration text, not AST; TypeScript component ranges may include nested functions.
- Related test source was inspected, not executed; absence of a link does not establish absence of tests.
- L0-L2 are navigation/decomposition references, not invented direct task-to-code mappings.
- BPMN labels and source evidence remain in their original language; no diagram/source is rewritten.
