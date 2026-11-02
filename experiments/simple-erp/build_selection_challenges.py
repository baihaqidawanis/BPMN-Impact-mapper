"""Freeze removal/condition transfer cases before inference; ERP remains read-only."""
import argparse
import copy
import json
from pathlib import Path
import xml.etree.ElementTree as ET
from build_structural_scenarios import tag, node, flow, diagram, sha


def save_json(path, value):
    path.write_bytes((json.dumps(value, indent=2) + '\n').encode())


def encode(root):
    diagram(root)
    ET.indent(root)
    return ET.tostring(root, encoding='utf-8', xml_declaration=True)


def build(repo, out):
    out.mkdir(parents=True, exist_ok=False)
    source = 'docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-copy.bpmn'
    original = (repo / source).read_bytes()
    root = ET.fromstring(original)
    process = root.find(tag('process'))
    prefix = 'role_staff_l3_pr_copy_'
    names = {
        'start': 'Purchase request copy requested', 'input': 'Select Copy purchase request to Draft',
        'check': 'Read source purchase request status under lock',
        'valid': 'Purchase request copy checks passed?',
        'effect0': 'Copy purchase request purpose and lines into a new DRAFT with copied_from_id',
        'effect1': 'Keep source purchase request unchanged; audit copy and commit',
        'done': 'Purchase request copy response delivered',
        'result_merge': 'Merge alternative purchase request copy outcomes',
        'result_activity': 'Send purchase request copy success or failure response',
    }
    for suffix, name in names.items():
        process.find(f"*[@id='{prefix + suffix}']").set('name', name)
    system = next(l for l in process.findall('.//' + tag('lane')) if l.get('name') == 'Sistem Simple ERP')
    node(process, 'serviceTask', 'CheckCopyProducts',
         'Check all source purchase request products are active', system)
    process.find(f"*[@id='{prefix}Flow_2']").set('sourceRef', 'CheckCopyProducts')
    flow(process, 'ReadThenCheckCopyProducts', prefix + 'check', 'CheckCopyProducts')

    def conditions(p, allowed, active):
        positive = f'source purchase request status in {allowed}' + (' and all linked products active' if active else '')
        for suffix, expression in [('Flow_3', positive), ('Flow_6', 'not (' + positive + ')')]:
            f = p.find(f"*[@id='{prefix + suffix}']")
            f.set('name', expression)
            f.find(tag('conditionExpression')).text = expression

    conditions(process, '[REJECTED]', True)
    before = encode(root)
    store = 'backend/internal/purchasing/adapter/postgres/store.go'
    tests = 'backend/test/integration/operations_test.go'
    ui = 'frontend/features/purchasing/purchase-request-panel.tsx'
    cases = []
    for case in ['remove-check', 'change-condition', 'unchanged-control']:
        folder = out / case
        folder.mkdir()
        (folder / 'before.bpmn').write_bytes(before)
        after_root = copy.deepcopy(root)
        p = after_root.find(tag('process'))
        labels = []
        if case == 'remove-check':
            p.remove(p.find("*[@id='CheckCopyProducts']"))
            p.remove(p.find("*[@id='ReadThenCheckCopyProducts']"))
            p.find(f"*[@id='{prefix}Flow_2']").set('sourceRef', prefix + 'check')
            for lane in p.findall('.//' + tag('lane')):
                for ref in list(lane):
                    if ref.tag == tag('flowNodeRef') and ref.text == 'CheckCopyProducts':
                        lane.remove(ref)
            conditions(p, '[REJECTED]', False)
            labels = [('REMOVE_TASK', 'CheckCopyProducts', [store, tests]),
                      ('MODIFY_GATEWAY_CONDITION', prefix + 'valid', [store, tests])]
        elif case == 'change-condition':
            conditions(p, '[REJECTED, CANCELLED]', True)
            labels = [('MODIFY_GATEWAY_CONDITION', prefix + 'valid', [store, tests, ui])]
        after = encode(after_root) if labels else before
        (folder / 'after.bpmn').write_bytes(after)
        save_json(folder / 'reference.json', {'version': '1', 'cases': [
            {'change_type': kind, 'element_id': id, 'expected': [
                {'file': f, 'required_action': 'MODIFY'} for f in files]}
            for kind, id, files in labels]})
        cases.append({
            'id': 'selection-challenge-' + case, 'repository_path': '../Simple-ERP',
            'repository_name': 'Simple-ERP', 'revision': 'selection-challenge-frozen',
            'old_bpmn_path': (folder / 'before.bpmn').as_posix(), 'old_bpmn_sha256': sha(before),
            'new_bpmn_path': (folder / 'after.bpmn').as_posix(), 'new_bpmn_sha256': sha(after),
            'ground_truth_path': (folder / 'reference.json').as_posix(),
            'ground_truth_sha256': sha((folder / 'reference.json').read_bytes()),
            'truth_scope': 'subset' if labels else 'complete',
        })
    save_json(out / 'source-review.json', {
        'status': 'provisional single-AI source review before inference; no independent truth',
        'baseline_source': source, 'baseline_source_sha256': sha(original),
        'baseline_refinement': 'English labels; split existing combined status/master check into two tasks. Same current source eligibility, not an untouched original BPMN.',
        'source_hashes': {f: sha((repo / f).read_bytes()) for f in [store, tests, ui,
            'backend/internal/purchasing/domain/purchase_request.go',
            'backend/internal/purchasing/application/service.go']},
        'requirements': {
            'remove-check': 'Allow copying REJECTED PR with inactive products. Remove active-master guard only; preserve status/locking/copy/audit/commit. No UI change: Copy already visible for REJECTED.',
            'change-condition': 'Allow REJECTED or CANCELLED source PR to be copied into new DRAFT, retain active-product check. Source status remains unchanged. UI exposes Copy for CANCELLED too; audit records actual source status.',
        },
        'evidence': [
            'Store.CopyRequest reads status/purpose/mastersActive in transaction; rejects status != REJECTED or !mastersActive, copies lines, writes hardcoded REJECTED audit origin.',
            'TestCopyRejectedDocumentsRequiresActiveMasters rejects copying inactive PR; operations tests should separate PR changes from unchanged sales assertions and cover retained guards.',
            'PurchaseRequestPanel displays Copy to Draft only for REJECTED; condition extension needs CANCELLED visibility. Guard removal does not change UI.',
            'Application.CopyRequest delegates unchanged signature. Domain PurchaseRequest.Copy already creates DRAFT with no REJECTED guard. Neither is labelled mandatory modification.',
            'No new fields, route, migration, role permission, or physical deletion of shared Store.CopyRequest required. REMOVE_TASK maps to MODIFY in shared function.',
        ],
        'policy': 'Use existing outputs/selection-calibration/policy.json unchanged. No tuning/labels altered after inference.',
    })
    save_json(out / 'manifest.json', {'version': '1', 'repetitions': 2, 'top_k': 10,
        'label_provenance': 'single-source-review', 'methods': ['bm25', 'semantic', 'fusion', 'hybrid'],
        'scenarios': cases})
    print('Frozen removal, condition, and unchanged cases; source-derived labels precede inference.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    build(args.repo, args.output)
