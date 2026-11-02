"""Build research-only before mappings. Reads ERP; never imports/runs ERP code.

Production retrieval must not read these labels. Review decisions live in
role_traceability_specs; this module anchors evidence and renders reference views.
"""
import argparse
import hashlib
import html
import json
import re
import subprocess
from pathlib import Path, PurePosixPath
import xml.etree.ElementTree as ET

from role_traceability_specs import SPECS, TESTS

NS = {'b': 'http://www.omg.org/spec/BPMN/20100524/MODEL'}
MAPPER = Path(__file__).resolve().parents[2]
ROLE_ROOT = 'docs/bpm-haqi/roles'
ROLES = ('staff', 'supervisor', 'super-admin')
MODULES = ('p2p', 'inventory', 'master-data')
EXPECTED = {'staff': (10, 8, 1), 'supervisor': (9, 6, 3), 'super-admin': (19, 11, 5)}
ACTIVITIES = {'task', 'userTask', 'serviceTask', 'sendTask', 'receiveTask', 'callActivity', 'subProcess'}
LIMITATIONS = [
    'Single AI static source review; no independent review or runtime/role-session verification.',
    'Existing task-to-code relationships are not required-change labels or an exhaustive dependency graph.',
    'Gateway branches are preserved as BPMN control references; not all conditions have independent source mappings.',
    'Function ranges use declaration text, not AST; TypeScript component ranges may include nested functions.',
    'Related test source was inspected, not executed; absence of a link does not establish absence of tests.',
    'L0-L2 are navigation/decomposition references, not invented direct task-to-code mappings.',
    'BPMN labels and source evidence remain in their original language; no diagram/source is rewritten.',
]


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2) + '\n').encode('utf-8')


def sha(data):
    return hashlib.sha256(data).hexdigest()


class SourceReview:
    def __init__(self, repo):
        self.repo = repo.resolve(strict=True)
        self.hashes = {}
        self.cache = {}

    def read(self, file):
        path = PurePosixPath(file)
        if path.is_absolute() or '..' in path.parts or '\\' in file or ':' in file:
            raise ValueError('Expected repository-relative POSIX path: ' + file)
        target = (self.repo / file).resolve(strict=True)
        if not target.is_relative_to(self.repo):
            raise ValueError('Source escapes repository: ' + file)
        if file not in self.cache:
            data = target.read_bytes()
            self.hashes[file] = sha(data)
            self.cache[file] = data
        return self.cache[file]

    def relation(self, file, function, role, needles):
        lines = self.read(file).decode('utf-8-sig').splitlines()
        if file.endswith('.go'):
            receiver, sep, name = function.rpartition('.')
            if sep:
                pattern = r'^func\s+\(\w+\s+\*?' + re.escape(receiver) + r'\)\s+' + re.escape(name) + r'\('
            else:
                pattern = r'^func\s+' + re.escape(function) + r'\('
            starts = [i for i, line in enumerate(lines) if line.startswith('func ')]
            hits = [i for i in starts if re.search(pattern, lines[i])]
            if len(hits) != 1:
                raise ValueError(('Function ambiguous/missing', file, function, hits))
            start = hits[0]
            end = next((i for i in starts if i > start), len(lines))
        else:
            pattern = r'^(?:export )?(?:async )?function\s+' + re.escape(function) + r'\b'
            starts = [i for i, line in enumerate(lines) if re.search(r'^(?:export )?(?:async )?function\s+', line)]
            hits = [i for i in starts if re.search(pattern, lines[i])]
            if len(hits) != 1:
                raise ValueError(('Component ambiguous/missing', file, function, hits))
            start = hits[0]
            end = next((i for i in starts if i > start), len(lines))
        evidence = []
        for needle in needles:
            hits = [i for i in range(start, end) if needle in lines[i]]
            if not hits:
                raise ValueError(('Reviewed evidence changed', file, function, needle))
            i = hits[0]
            line = lines[i].strip()
            pos = line.index(needle)
            # Dense one-line JSX must show the actual matched control, not an
            # unrelated prefix that hides the needle after truncation.
            start_excerpt = max(0, pos - 70)
            excerpt = line[start_excerpt:start_excerpt + 320]
            evidence.append({'line': i + 1, 'match': needle,
                             'excerpt': ('...' if start_excerpt else '') + excerpt + ('...' if start_excerpt + 320 < len(line) else '')})
        if not evidence:
            raise ValueError(('Evidence required', file, function))
        return dict(file=file, function=function, role=role,
                    function_start_line=start + 1, function_end_line=end, evidence=evidence)


def paths(spec):
    base = 'backend/internal/' + spec['module']
    tail = 'settings.go' if spec.get('settings') else 'store.go'
    handler = base + '/adapter/http/' + ('settings.go' if spec.get('settings') else 'handler.go')
    service = base + '/application/' + ('settings.go' if spec.get('settings') else 'service.go')
    ui_module = {'masterdata': 'settings' if spec.get('settings') else 'master-data'}.get(spec['module'], spec['module'])
    frontend = 'frontend/features/' + ui_module + '/' + spec['ui'] + '.tsx'
    return base + '/adapter/postgres/' + tail, handler, service, frontend


def mapping(review, op, node):
    spec = SPECS[op]
    store, handler, service, frontend = paths(spec)
    receiver = 'SettingsStore' if spec.get('settings') else 'Store'
    rel = review.relation
    task = node.get('id')
    if task.endswith('_input'):
        return [rel(frontend, spec['component'], 'USER_INPUT_OR_VIEW', spec['input'])]
    if task.endswith('_read'):
        return [rel(store, receiver + '.' + spec['store'], 'READ_QUERY', spec['read'])]
    if task.endswith('_check'):
        links = [rel(store, receiver + '.' + spec['store'], 'VALIDATION', spec['check'])]
        if op == 'master-create':
            links += [rel('backend/internal/masterdata/domain/entity.go', 'NewRecord', 'INPUT_VALIDATION', ['invalid kind', 'code == "" || name == ""'])]
        if op == 'settings-update':
            links += [rel('backend/internal/masterdata/domain/settings.go', 'OrganizationSettings.Validate', 'INPUT_VALIDATION', ['len(s.Currency) != 3', 'strings.TrimSpace(s.Timezone)', 'strings.TrimSpace(s.DefaultWarehouseID)'])]
        return links
    for branch in ('positive', 'negative'):
        if task.endswith('_' + branch):
            return [rel(store, receiver + '.' + spec['store'], 'BRANCH_OUTCOME', spec['branches'][branch])]
    if re.search(r'_effect\d+$', task):
        index = int(task.rsplit('_effect', 1)[1])
        needles = spec['effects'][index]
        role = 'AUDIT_AND_COMMIT' if any('Commit(' in n for n in needles) else 'PERSISTENCE_OR_STATE'
        return [rel(store, receiver + '.' + spec['store'], role, needles)]
    if task.endswith('_result_activity'):
        function = ('SettingsHandler.' if spec.get('settings') else 'Handler.') + spec['handler']
        if spec['module'] == 'finance':
            return [rel(handler, function, 'RESPONSE_DISPATCH', ['result(w, document, err,']),
                    rel(handler, 'result', 'SUCCESS_ERROR_RESPONSE', ['if err != nil', 'respond(w, status, document)']),
                    rel(handler, 'respond', 'HTTP_JSON_RESPONSE', ['w.WriteHeader(status)', 'json.NewEncoder(w)'])]
        common = 'backend/internal/masterdata/adapter/http/handler.go' if spec.get('settings') else handler
        response = ['respond(w, 409', 'w.WriteHeader(204)'] if op == 'master-status' else (['respond(w, 400' if op == 'master-list' else 'respond(w, 500', 'respond(w, 200'] if 'read' in spec else ['respond(w, 409', 'respond(w, 20'])
        return [rel(handler, function, 'SUCCESS_ERROR_RESPONSE', response),
                rel(common, 'respond', 'HTTP_JSON_RESPONSE', ['w.WriteHeader(s)', 'json.NewEncoder(w)'])]
    raise ValueError(('Unreviewed activity', op, task))


def context(review, op):
    spec = SPECS[op]
    store, handler, service, _ = paths(spec)
    fn = spec.get('service', spec['store'])
    receiver = 'SettingsHandler' if spec.get('settings') else 'Handler'
    service_receiver = 'SettingsService' if spec.get('settings') else 'Service'
    rel = review.relation
    links = [rel(handler, receiver + '.' + spec['handler'], 'API_SERVICE_BINDING', ['h.Service.' + fn + '(']),
             rel(service, service_receiver + '.' + fn, 'SERVICE_STORE_BINDING', ['s.store.' + spec['store'] + '('])]
    if spec.get('settings'):
        links += [rel('backend/cmd/api/main.go', 'main', 'ROLE_ACCESS_BOUNDARY', ['api.Handle("/settings/organization", authMiddleware.RequireRole(domain.RoleSuperAdmin, settingsHandler))'])]
    elif spec['module'] == 'masterdata' and op == 'master-list':
        links += [rel('backend/cmd/api/main.go', 'main', 'AUTHENTICATION_BOUNDARY', ['api.Mount("/master-data", authMiddleware.RequireAuth(masterHandler.Routes()))'])]
    elif 'read' in spec:
        links += [rel('backend/cmd/api/main.go', 'main', 'AUTHENTICATION_BOUNDARY', ['r.Use(authMiddleware.RequireAuth)', 'inventoryHandler.Register(r)'])]
    else:
        creator = op in {'pr-create', 'pr-submit', 'pr-cancel', 'pr-copy', 'po-convert', 'gr-create', 'gr-cancel', 'ap-invoice', 'ap-resolve', 'ap-payment', 'adj-create', 'adj-submit', 'adj-post', 'adj-cancel', 'delivery-post'}
        gate = 'CanCreate' if creator else 'CanApprove'
        needle = ('requireCreate(w, r)' if creator else 'requireApprove(w, r)') if spec['module'] == 'finance' else 'domain.' + gate + '('
        links += [rel(handler, receiver + '.' + spec['handler'], 'ROLE_ACCESS_BOUNDARY', [needle]),
                  rel('backend/internal/auth/domain/role.go', gate, 'ROLE_POLICY', ['return role == RoleStaff || role == RoleSuperAdmin' if creator else 'return role == RoleSupervisor || role == RoleSuperAdmin'])]
        if spec['module'] == 'finance':
            links += [rel(handler, 'requireCreate' if creator else 'requireApprove', 'ROLE_CHECK_HELPER', ['authdomain.' + gate + '('])]
    if spec['module'] == 'masterdata' and not spec.get('settings'):
        links += [rel(store, 'table', 'ALLOWED_MASTER_KINDS', ['case domain.Product:', 'case domain.Supplier:', 'case domain.Customer:', 'case domain.Unit:', 'case domain.Warehouse:', 'invalid kind'])]
    if op in {'pr-review', 'gr-post', 'ap-match', 'ap-payment-post', 'adj-review'}:
        links += [rel('backend/internal/platform/authorization/super_admin.go', 'CanSelfReview', 'SELF_REVIEW_POLICY', ["role='SUPER_ADMIN' AND status='ACTIVE'"])]
    if op == 'ap-invoice':
        links += [rel(store, 'supplierReplacementValid', 'VALIDATION_HELPER', ['return validateReplacement(']),
                  rel(store, 'validateReplacement', 'REPLACEMENT_RULE', ['latestStatus'])]
    if op == 'ap-payment':
        links += [rel(store, 'allocationTotal', 'VALIDATION_HELPER', ['allocations', 'return'])]
    if op in TESTS:
        file, function, needle = TESTS[op]
        links += [rel('backend/test/integration/' + file, function, 'RELATED_TEST_NOT_EXECUTED', [needle])]
    return links


def build(repo):
    review = SourceReview(repo)
    base = review.repo / ROLE_ROOT
    processes, hierarchy, by_id = [], [], {}
    for file in sorted(base.rglob('*.bpmn')):
        path = file.relative_to(review.repo).as_posix()
        root = ET.fromstring(review.read(path))
        p = root.find('b:process', NS)
        if p is None:
            raise ValueError('Missing process: ' + path)
        if p.get('id') in by_id:
            raise ValueError('Duplicate process: ' + p.get('id'))
        by_id[p.get('id')] = path
        role = file.relative_to(base).parts[0]
        if role not in ROLES:
            raise ValueError('Unreviewed role: ' + role)
        module = 'p2p' if '/p2p/' in path else ('inventory' if '/inventory/' in path else ('all' if '-l0' in file.stem else 'master-data'))
        match = re.search(r'-l([0-3])(?:-|$)', file.stem)
        if not match:
            raise ValueError('Unknown diagram level: ' + path)
        level = int(match[1])
        actors = {ref.text: lane.get('name') for lane in p.findall('.//b:lane', NS) for ref in lane.findall('b:flowNodeRef', NS)}
        nodes = [n for n in p if n.tag.split('}')[-1] in ACTIVITIES]
        common = dict(bpmn_file=path, process_id=p.get('id'), name=p.get('name'), role=role, module=module, level=level)
        controls = [{'id': n.get('id'), 'type': n.tag.split('}')[-1], 'name': n.get('name', ''),
                     'status': 'BPMN control reference; no standalone source mapping asserted'}
                    for n in p if n.tag.split('}')[-1].endswith(('Gateway', 'Event'))]
        flows = [dict(id=n.get('id'), source=n.get('sourceRef'), target=n.get('targetRef'), name=n.get('name', ''),
                      condition=n.findtext('b:conditionExpression', default='', namespaces=NS)) for n in p.findall('b:sequenceFlow', NS)]
        if level == 3:
            op = file.stem.split('-l3-', 1)[1]
            if op not in SPECS:
                raise ValueError('Unreviewed operation: ' + op)
            activities = [dict(task_id=n.get('id'), name=n.get('name'), type=n.tag.split('}')[-1],
                               actor=actors[n.get('id')], documentation=n.findtext('b:documentation', default='', namespaces=NS),
                               review_status='provisional-source-reviewed', mappings=mapping(review, op, n)) for n in nodes]
            processes.append(dict(**common, operation=op, activities=activities,
                                  operation_context=context(review, op), test_link_status='related source inspected, not executed' if op in TESTS else 'dedicated test link not reviewed',
                                  control_elements=controls, sequence_flows=flows))
        else:
            # Namespaced calledElement references are explicit decomposition, not
            # a guessed association based on task names or mapper candidates.
            hierarchy.append(dict(**common, activities=[dict(id=n.get('id'), name=n.get('name'), type=n.tag.split('}')[-1],
                called_process=n.get('calledElement', '').split(':')[-1],
                status='decomposition reference' if n.get('calledElement') else 'navigation or result observation; no direct source mapping asserted') for n in nodes],
                control_elements=controls, sequence_flows=flows))
    for process in hierarchy:
        for node in process['activities']:
            if node['called_process']:
                if node['called_process'] not in by_id:
                    raise ValueError('Missing called process: ' + node['called_process'])
                node['target_bpmn'] = by_id[node['called_process']]
    for role in ROLES:
        for module, expected in zip(MODULES, EXPECTED[role]):
            actual = sum(p['role'] == role and p['module'] == module for p in processes)
            if actual != expected:
                raise ValueError(('Coverage drift', role, module, expected, actual))
    if len(hierarchy) != 40 or len(processes) != 72:
        raise ValueError('Expected current 112-diagram package')
    ids = [a['task_id'] for p in processes for a in p['activities']]
    if len(ids) != len(set(ids)):
        raise ValueError('Duplicate activity IDs')
    revision = subprocess.check_output(['git', '-C', str(review.repo), 'rev-parse', 'HEAD'], text=True).strip()
    dirty = bool(subprocess.check_output(['git', '-C', str(review.repo), 'status', '--porcelain'], text=True))
    value = dict(schema_version='2', purpose='research-only baseline traceability; excluded from inference and model assessment input',
                 review_status='provisional single AI static source review',
                 repository=dict(name=review.repo.name, git_head=revision, working_tree_dirty=dirty), limitations=LIMITATIONS,
                 scope=dict(roles=list(ROLES), modules=list(MODULES), level3_processes=len(processes), activities=len(ids), hierarchy_diagrams=len(hierarchy)),
                 file_sha256=dict(sorted(review.hashes.items())), processes=processes, hierarchy=hierarchy)
    value['snapshot_id'] = sha(encoded(value))[:32]
    validate(value)
    return value


def validate(value):
    """Check a stored bundle without interpreting predicted impact results."""
    ids = set()
    expected_hash = sha(encoded({k: v for k, v in value.items() if k != 'snapshot_id'}))[:32]
    if value['snapshot_id'] != expected_hash:
        raise ValueError('Bundle snapshot hash mismatch')
    for p in value['processes']:
        if p['bpmn_file'] not in value['file_sha256']:
            raise ValueError('Missing BPMN hash')
        for a in p['activities']:
            if a['task_id'] in ids or not a['actor'] or not a['mappings']:
                raise ValueError('Duplicate or incomplete activity')
            ids.add(a['task_id'])
        for link in [m for a in p['activities'] for m in a['mappings']] + p['operation_context']:
            if link['file'] not in value['file_sha256'] or not link['evidence']:
                raise ValueError('Missing source hash/evidence')
            for evidence in link['evidence']:
                if not link['function_start_line'] <= evidence['line'] <= link['function_end_line']:
                    raise ValueError('Evidence outside function')
                if evidence['match'] not in evidence['excerpt']:
                    raise ValueError('Excerpt hides matching evidence')
    if len(ids) != value['scope']['activities']:
        raise ValueError('Activity coverage mismatch')


def esc(value):
    return html.escape(str(value), quote=False).replace('|', '\\|').replace('\n', ' ').strip()


def render(value):
    out = ['# Mapping before: ' + esc(value['scope']['role']) + ' / ' + esc(value['scope']['module']), '',
           f'Snapshot paket: `{value["snapshot_id"]}`. Git HEAD: `{value["repository"]["git_head"]}`; working tree dirty: {value["repository"]["working_tree_dirty"]}.', '',
           'Acuan existing dari inspeksi source; bukan daftar file wajib berubah, bukti runtime, atau input inference.',
           'JSON adalah data utama; tabel ini dirender dari object yang sama. Path relatif terhadap Simple-ERP; baris berlaku pada hash snapshot.', '']
    for p in value['processes']:
        out += ['## ' + esc(p['name']), '', f'BPMN: `{p["bpmn_file"]}`', '',
                '| Task / actor | File | Function | Hubungan | Bukti source |', '|---|---|---|---|---|']
        for a in p['activities']:
            for link in a['mappings']:
                evidence = '<br>'.join(f'L{e["line"]}: {esc(e["excerpt"])}' for e in link['evidence'])
                out.append('| ' + ' | '.join([esc(a['task_id']) + '<br>' + esc(a['name']) + '<br>' + esc(a['type'] + ' / ' + a['actor']),
                           esc(link['file']), esc(link['function']), esc(link['role']), evidence]) + ' |')
        for a in p['activities']:
            if a['documentation']:
                out += ['', '**Catatan ' + esc(a['task_id']) + ':** ' + esc(a['documentation']), '']
        out += ['', 'Rantai API, role, helper dan test pendukung (bukan direct mapping semua task):', '']
        for link in p['operation_context']:
            evidence = '; '.join(f'L{e["line"]}: {esc(e["excerpt"])}' for e in link['evidence'])
            out.append(f'- `{link["file"]}` / `{link["function"]}` — {link["role"]}: {evidence}')
        out += ['', 'Test: ' + p['test_link_status'] + '.', '',
                f'{len(p["control_elements"])} event/gateway serta {len(p["sequence_flows"])} flow disimpan di JSON sebagai kontrol BPMN. Tidak dinyatakan sebagai function tersendiri.', '']
    out += ['## Batas penggunaan', ''] + ['- ' + s for s in value['limitations']] + ['']
    return ('\n'.join(out)).encode('utf-8')


def artifacts(value):
    files = {'all-roles.json': encoded(value)}
    index = ['# Mapping baseline tiga modul × tiga role', '',
             'Pemetaan task existing → file/function, terpisah dari ground truth perubahan dan input retrieval.', '',
             f'Snapshot `{value["snapshot_id"]}`: 72 proses L3, {value["scope"]["activities"]} aktivitas, 40 diagram L0–L2.', '',
             '| Role | Modul | Proses L3 | Aktivitas | Tabel | JSON |', '|---|---|---|---|---|---|']
    for role in ROLES:
        for module in MODULES:
            subset = [p for p in value['processes'] if p['role'] == role and p['module'] == module]
            view = {k: v for k, v in value.items() if k not in ('processes', 'hierarchy', 'scope')}
            view.update(scope=dict(role=role, module=module, level=3, processes=len(subset), activities=sum(len(p['activities']) for p in subset)), processes=subset)
            name = role + '-' + module + '-l3'
            data = encoded(view)
            files[name + '.json'] = data
            files[name + '.md'] = render(json.loads(data))
            index.append(f'| {role} | {module} | {len(subset)} | {view["scope"]["activities"]} | [Buka]({name}.md) | [Data]({name}.json) |')
    index += ['', '## Level 0–2', '',
              'Indeks berikut memakai calledElement yang benar-benar ada di XML. Task pemilihan/menu dan pemeriksaan hasil ditandai sebagai navigasi/observasi, bukan dibuatkan mapping function palsu.', '',
              '| Role / level | Diagram | Proses turunan |', '|---|---|---|']
    for p in value['hierarchy']:
        targets = '<br>'.join(esc(a['name']) + ' → `' + a['target_bpmn'] + '`' for a in p['activities'] if a.get('target_bpmn')) or 'Tidak ada calledElement; lihat XML sebagai peta akses.'
        index.append(f'| {p["role"]} / L{p["level"]} | `{p["bpmn_file"]}` | {targets} |')
    index += ['', '## Membaca dan memverifikasi', '',
              '- USER_INPUT_OR_VIEW: form, tombol atau halaman frontend terkait kebutuhan task.',
              '- VALIDATION / INPUT_VALIDATION: guard atau aturan input yang terlihat di function.',
              '- PERSISTENCE_OR_STATE / BRANCH_OUTCOME: query/state sukses atau alternatif.',
              '- AUDIT_AND_COMMIT: audit dan penyelesaian transaksi.',
              '- RESPONSE: handler/helper mengirim hasil atau error, bukan layanan pesan baru.',
              '- ROLE_POLICY: batas akses; role tetap memiliki proses yang berbeda sesuai implementasi.',
              '- RELATED_TEST_NOT_EXECUTED: kaitan source test, tidak membuktikan runtime lulus.', '',
              '```powershell', 'python experiments/simple-erp/build_role_traceability.py --repo C:\\Simple-ERP --output experiments/simple-erp/baseline-traceability/all-roles --verify', '```', '',
              'Verifikasi membangun ulang reference dari keputusan review eksplisit, memeriksa bukti function, hash, cakupan dan byte JSON/Markdown. Drift harus ditinjau; jangan menyesuaikan referensi berdasarkan kandidat mapper.', '',
              'Catalog/generator BPMN dibaca sebagai petunjuk struktur saja; tidak dieksekusi. Bukti mapping berasal dari source implementasi. Paket ini bukan metode mapping otomatis atau klaim generalisasi.', '',
              '## Batas penggunaan', ''] + ['- ' + s for s in value['limitations']] + ['']
    files['README.md'] = ('\n'.join(index)).encode('utf-8')
    return files


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--verify', action='store_true')
    args = parser.parse_args()
    repo, output = args.repo.resolve(strict=True), args.output.resolve()
    if not output.is_relative_to(MAPPER) or output.is_relative_to(repo):
        parser.error('Output must be inside Mapper and outside the analyzed repository')
    value = build(repo)
    files = artifacts(value)
    if args.verify:
        for name, data in files.items():
            if (output / name).read_bytes() != data:
                raise ValueError('Snapshot/render drift: ' + name + '; review changes before regenerating')
        print(f'Verified: {len(files)} artifacts, 72 L3 processes, {value["scope"]["activities"]} activities, 40 hierarchy diagrams; source evidence, hashes and rendered Markdown match.')
    else:
        output.mkdir(parents=True, exist_ok=False)
        for name, data in files.items():
            (output / name).write_bytes(data)
        print(f'Created: {len(files)} artifacts; {value["scope"]}; snapshot {value["snapshot_id"]}')


if __name__ == '__main__':
    main()
