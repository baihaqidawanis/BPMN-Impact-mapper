"""Research-only lock and label audit. Never reads predictions or runs ERP/models."""
import argparse
import fnmatch
import json
import subprocess
from pathlib import Path, PurePosixPath

from build_role_traceability import MAPPER, SourceReview, encoded, sha

PACKAGES = ('current-staff', 'p2p-transfer', 'selection-challenges')
EXTENSIONS = {'.go', '.mod', '.sum', '.ts', '.tsx', '.js', '.jsx', '.mjs', '.cjs', '.json', '.md', '.sql', '.yaml', '.yml', '.xml', '.bpmn', '.py', '.css', '.html', '.toml', '.txt'}
SKIP_DIRS = {'.git', 'node_modules', 'vendor', 'dist', 'build', 'cache', '.next', '__pycache__', '.data', 'outputs', 'coverage', 'test-results', 'playwright-report'}
SKIP_FILES = ('.env*', '*credential*', '*secret*', '*.pem', '*.key')


def read_relative(root, path):
    if not isinstance(path, str) or '\\' in path or ':' in path or PurePosixPath(path).is_absolute() or '..' in PurePosixPath(path).parts:
        raise ValueError('Unsafe relative path: ' + str(path))
    target = (root / path).resolve(strict=True)
    if not target.is_relative_to(root.resolve()):
        raise ValueError('Path escapes root: ' + path)
    return target.read_bytes()


def textual_snapshot(repo):
    """Git-listed current text, not the production ingestion corpus/archive."""
    listed = subprocess.check_output(['git', '-C', str(repo), 'ls-files', '--cached', '--others', '--exclude-standard', '-z']).decode('utf-8').split(chr(0))
    hashes, total = {}, 0
    for name in sorted(set(listed) - {''}):
        path = Path(name)
        if any(p.lower() in SKIP_DIRS for p in path.parts) or any(fnmatch.fnmatch(path.name.lower(), pat) for pat in SKIP_FILES):
            continue
        target = repo / path
        if path.suffix.lower() not in EXTENSIONS or not target.exists() or target.is_symlink() or not target.is_file():
            continue
        if any(p.is_symlink() for p in target.parents if p != repo and p.is_relative_to(repo)):
            continue
        if target.stat().st_size > 2 * 1024 * 1024:
            continue
        data = read_relative(repo, path.as_posix())
        if bytes([0]) in data:
            continue
        total += len(data)
        if total > 256 * 1024 * 1024 or len(hashes) >= 20000:
            raise ValueError('Snapshot safety bound exceeded')
        hashes[path.as_posix()] = sha(data)
    return hashes


def required_changes(review, package, name):
    """Explicit review judgments, independent of candidate ranks/metrics."""
    if name == 'unchanged-control':
        return [], ['No requirement change.'], 'Identical BPMN control.', 'unchanged-control'
    inventory, copy_case = package == 'current-staff', package == 'selection-challenges'
    module = 'inventory' if inventory else 'purchasing'
    store = f'backend/internal/{module}/adapter/postgres/store.go'
    test = 'backend/test/integration/operations_test.go'
    panel = f'frontend/features/{module}/' + ('inventory-adjustment-panel.tsx' if inventory else 'purchase-request-panel.tsx')
    rel = review.relation
    if copy_case:
        expanded = name == 'change-condition'
        requirement = ('Allow REJECTED or CANCELLED PR copy; preserve active products and source state.' if expanded else 'Allow copying REJECTED PR with inactive products; retain status guard and source state.')
        links = [dict(file=store, reason='Extend status eligibility and audit origin.' if expanded else 'Remove active-product guard only; shared function remains.', evidence=[rel(store, 'Store.CopyRequest', 'CURRENT_GUARD', ['status != "REJECTED"', '!mastersActive', '"REJECTED"'])]),
                 dict(file=test, reason='Extend Copy PR tests; preserve unchanged sales assertions and retained guards.', evidence=[rel(test, 'TestCopyRejectedDocumentsRequiresActiveMasters', 'RELATED_TEST_NOT_EXECUTED', ['purchasing.CopyRequest('])])]
        if expanded:
            links += [dict(file=panel, reason='Expose existing Copy action for CANCELLED too.', evidence=[rel(panel, 'PurchaseRequestPanel', 'UI_GUARD', ['document.status === "REJECTED"', 'action(document, "copy")'])])]
        unchanged = ['No new route, request field, database column or role permission.', 'Existing application delegation/domain Copy do not implement the changed guard.']
        if not expanded:
            unchanged += ['Copy remains visible for REJECTED; no frontend change required.']
        return links, unchanged, requirement, 'pr-copy-eligibility'
    reason_required = inventory and name in ('add-task', 'add-xor', 'combined')
    masters_required = not inventory or name in ('add-and', 'combined')
    function = 'Store.TransitionAdjustment' if inventory else 'Store.TransitionRequest'
    evidence = [rel(store, function, 'CURRENT_SUBMIT', ['case "submit":', 'from != "DRAFT"', 'd.Status = "SUBMITTED"'])]
    if masters_required:
        evidence += [rel(store, 'Store.CreateAdjustment' if inventory else 'Store.CreateRequest', 'CREATE_TIME_CHECKS', ['FROM warehouses', 'FROM products'] if inventory else ['FROM products', 'l.UnitPrice < 0'])]
    requirement = ('Require nonempty adjustment submit reason. ' if reason_required else '') + (('At submit recheck active warehouse AND all products. ' if inventory else 'At PR submit recheck all products active AND estimated prices strictly positive. ') if masters_required else '') + 'Reject failures without changing state; preserve version/access guards.'
    links = [dict(file=store, reason='Add submit-time validation in the existing transaction before status mutation; create-time checks alone are insufficient.', evidence=evidence),
             dict(file=test, reason='Extend existing integration test for new failures, success and preserved invariants.', evidence=[rel(test, 'TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot' if inventory else 'TestP2PPartialReceiptPostsImmutableSourceLinkedLedger', 'RELATED_TEST_NOT_EXECUTED', ['.Transition(' if inventory else '.TransitionRequest('])])]
    if reason_required:
        links += [dict(file=panel, reason='Existing reason input sends empty reason on submit; require and send it for submit.', evidence=[rel(panel, 'InventoryAdjustmentPanel', 'CURRENT_REASON_PAYLOAD', ['const actionReason =', 'reason: actionReason'])])]
    unchanged = ['Existing endpoint, request/response and schema suffice; no migration.', 'AND requires both checks to finish before mutation; no goroutine requirement.']
    if not reason_required:
        unchanged += ['No new input; existing frontend/handler reuse version and error response.']
    return links, unchanged, requirement, 'inventory-submit-validation' if inventory else 'pr-submit-master-price-validation'


def build(repo):
    repo = repo.resolve(strict=True)
    review, bindings, scenarios = SourceReview(repo), {}, []
    baseline = 'experiments/simple-erp/baseline-traceability/all-roles/all-roles.json'
    data = read_relative(MAPPER, baseline)
    for path, digest in json.loads(data)['file_sha256'].items():
        if sha(review.read(path)) != digest:
            raise ValueError('Baseline source drift: ' + path)
    bindings[baseline] = sha(data)
    for package in PACKAGES:
        path = f'experiments/simple-erp/{package}/manifest.json'
        data = read_relative(MAPPER, path)
        bindings[path] = sha(data)
        manifest = json.loads(data)
        if manifest['top_k'] != 10 or manifest['repetitions'] != 2 or manifest['methods'] != ['bm25', 'semantic', 'fusion', 'hybrid']:
            raise ValueError('Experiment settings changed; review before freezing')
        for s in manifest['scenarios']:
            for key in ('old_bpmn', 'new_bpmn', 'ground_truth'):
                path = s[key + '_path']
                digest = sha(read_relative(MAPPER, path))
                if digest != s[key + '_sha256']:
                    raise ValueError('Manifest binding drift: ' + path)
                bindings[path] = digest
            folder = Path(s['new_bpmn_path']).parent
            name = folder.name
            rp = folder / 'source-review.json' if package == 'current-staff' else Path('experiments/simple-erp') / package / 'source-review.json'
            data = read_relative(MAPPER, rp.as_posix())
            bindings[rp.as_posix()] = sha(data)
            for path, digest in json.loads(data)['source_hashes'].items():
                if sha(review.read(path)) != digest:
                    raise ValueError('Scenario source drift: ' + path)
            labels = json.loads(read_relative(MAPPER, s['ground_truth_path']))
            links, unchanged, requirement, family = required_changes(review, package, name)
            expected = {e['file'] for case in labels['cases'] for e in case['expected']}
            if expected != {link['file'] for link in links}:
                raise ValueError('Review disagrees with labels: ' + s['id'])
            if any(e['required_action'] != 'MODIFY' for c in labels['cases'] for e in c['expected']):
                raise ValueError('Unexpected action requires review')
            if name == 'unchanged-control' and s['old_bpmn_sha256'] != s['new_bpmn_sha256']:
                raise ValueError('Control is not unchanged')
            scenarios.append(dict(id=s['id'], split='development-regression', prior_exposure=True, requirement_family=family,
                truth_scope=s.get('truth_scope', 'complete'), reference_status='provisional single AI source review; no implemented after',
                requirement=requirement, before=s['old_bpmn_path'], after=s['new_bpmn_path'], reference=s['ground_truth_path'],
                labelled_changes=[dict(change_type=c['change_type'], element_id=c['element_id']) for c in labels['cases']],
                required_change_files=links, unchanged_contracts=unchanged))
    for package in ('current-staff/combined', 'current-staff/add-and'):
        for name in ('node-reference.json', 'node-reference-review.json'):
            path = f'experiments/simple-erp/{package}/{name}'
            if (MAPPER / path).exists():
                bindings[path] = sha(read_relative(MAPPER, path))
    policy_files = sorted([p for top in ('internal', 'cmd') for p in (MAPPER / top).rglob('*.go')] + [MAPPER / 'go.mod', Path(__file__).resolve(), ROOT_SPECS,
        ROOT_SPECS.with_name('build_role_traceability.py'), ROOT_SPECS.with_name('build_baseline_traceability.py')])
    value = dict(version='1', purpose='research-only; excluded from inference/model assessment input',
        repository=dict(name=repo.name, head=subprocess.check_output(['git', '-C', str(repo), 'rev-parse', 'HEAD'], text=True).strip(), textual_worktree_sha256=textual_snapshot(repo)),
        snapshot_scope='Git-listed text allowlist, not production inventory/archive; excludes secrets/dependencies/generated/binary/symlink/oversized files.',
        mapper_policy_sha256={p.relative_to(MAPPER).as_posix(): sha(p.read_bytes()) for p in policy_files},
        research_input_sha256=dict(sorted(bindings.items())), reviewed_source_sha256=dict(sorted(review.hashes.items())),
        experiment=dict(methods=['bm25', 'semantic', 'fusion', 'hybrid'], top_k=10, repetitions=2, semantic_model='all-MiniLM-L6-v2',
            runtime_status='Not executed here. Future runs must record actual model hash, package versions and analysis fingerprint.',
            metrics=['Precision@5', 'Precision@10', 'Recall@5', 'Recall@10', 'MAP', 'MRR', 'R-precision'],
            evaluation_rule='Per labelled change, preserving subset scope; related files are not ranked hits. Future LLM selection P/R/F1 evaluated separately.'),
        splits=dict(development_regression=[s['id'] for s in scenarios], independent_test=[],
            reason='All existing cases exposed; no fabricated held-out set.',
            future_protocol=['Freeze method and requirements before first inference.', 'Use new requirements; group by baseline process and requirement family, including controls; never split near-duplicate mutations across development/test.',
                'Prepare source-based labels without candidates; record reviewer provenance.', 'Do not tune on independent test results; retire exposed/tuned cases to development.',
                'Report per-case/per-family results and controls separately; unlabeled nodes are not negatives.']), scenarios=scenarios)
    value['freeze_id'] = sha(encoded(value))[:32]
    return value


ROOT_SPECS = MAPPER / 'experiments/simple-erp/role_traceability_specs.py'


def render(value):
    out = ['# Freeze penelitian sebelum LLM', '', 'Freeze ID: ' + value['freeze_id'], '',
        'Sepuluh kasus lama dicatat sebagai development/regression. Belum ada kasus uji independen. Label tidak diubah mengikuti ranking mapper.', '',
        'Hash mengikat BPMN, label, source review, source ERP dan kode kebijakan saat ini. Ini pemeriksaan drift, bukan backup repository atau bukti runtime.', '',
        '| Kasus | Keluarga kebutuhan | Scope | File acuan |', '|---|---|---|---|']
    for s in value['scenarios']:
        out += [f'| {s["id"]} | {s["requirement_family"]} | {s["truth_scope"]} | {len(s["required_change_files"])} |']
    for s in value['scenarios']:
        out += ['', '## ' + s['id'], '', s['requirement'], '', 'Before: ' + s['before'], '', 'After: ' + s['after'], '', 'Label: ' + s['reference'], '', 'Acuan perubahan sementara:', '']
        for link in s['required_change_files']:
            out += ['- ' + link['file'] + ': ' + link['reason']]
            for relation in link['evidence']:
                for evidence in relation['evidence']:
                    out += [f'  - {relation["function"]} L{evidence["line"]}: {evidence["match"]}.']
        if not s['required_change_files']:
            out += ['Tidak ada kebutuhan berubah; kontrol identik.']
        out += ['', 'Kontrak yang tetap dapat digunakan:', ''] + ['- ' + x for x in s['unchanged_contracts']]
    out += ['', '## Aturan pengujian baru', ''] + ['- ' + x for x in value['splits']['future_protocol']]
    out += ['', '## Batas interpretasi', '',
        '- Label dari satu review AI terhadap source; bukan review independen atau implementation diff after.',
        '- Mapping before bukan daftar file wajib berubah. Requirement yang menentukan perubahan.',
        '- Node/flow di luar subset bukan otomatis negatif. Label node tambahan dievaluasi terpisah.',
        '- Source test terkait belum dijalankan; pilihan file test tetap provisional.',
        '- Tidak ada hasil ranking/metrik baru atau model yang dijalankan dalam audit ini.',
        '- Verifikasi konfigurasi runtime dan model hash pada hasil inference; freeze ini tidak membuktikan model siap.', '',
        'Verifikasi dari root Mapper:', '',
        '    python experiments/simple-erp/research_freeze.py --repo C:\\Simple-ERP --output experiments/simple-erp/research-freeze --verify', '']
    return ('\n'.join(out)).encode('utf-8')


def compare(directory, value):
    for name, data in [('freeze.json', encoded(value)), ('README.md', render(value))]:
        if (directory / name).read_bytes() != data:
            raise ValueError('Freeze drift: ' + name + '; inspect before creating a new version')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--verify', action='store_true')
    args = parser.parse_args()
    repo, output = args.repo.resolve(strict=True), args.output.resolve()
    if not output.is_relative_to(MAPPER) or output.is_relative_to(repo):
        parser.error('Output must stay in Mapper, outside ERP')
    value = build(repo)
    if args.verify:
        compare(output, value)
        print('Verified research freeze:', value['freeze_id'])
    else:
        output.mkdir(parents=True, exist_ok=False)
        (output / 'freeze.json').write_bytes(encoded(value))
        (output / 'README.md').write_bytes(render(value))
        print('Created research freeze:', value['freeze_id'], len(value['scenarios']), 'development cases; 0 independent tests')


if __name__ == '__main__':
    main()
