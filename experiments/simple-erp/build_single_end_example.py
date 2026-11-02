"""Source-reviewed Staff example and combined-gateway fixture; ERP stays read-only.
Never rewrite existing frozen benchmark pairs. No inference or hidden labels read.
"""
import argparse
import copy
import json
from pathlib import Path
import xml.etree.ElementTree as ET
from build_structural_scenarios import tag, node, flow, remove, diagram, sha


def write(root, path):
    diagram(root)
    ET.indent(root)
    data = ET.tostring(root, encoding='utf-8', xml_declaration=True)
    path.write_bytes(data)
    return sha(data)


def build(repo, output):
    output.mkdir(parents=True, exist_ok=False)
    source = 'docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn'
    data = (repo / source).read_bytes()
    root = ET.fromstring(data)
    process = root.find(tag('process'))
    prefix = 'role_staff_l3_adj_submit_'
    system = process.findall('.//' + tag('lane'))[1]
    ends = process.findall(tag('endEvent'))
    if {e.get('id') for e in ends} != {prefix+'done', prefix+'failed'}:
        raise ValueError('Baseline endings changed; review before normalizing')
    if any(list(e) and any(c.tag.endswith('EventDefinition') for c in e) for e in ends):
        raise ValueError('Never normalize error/terminate ends as none ends')
    # Keep outcome descriptions on incoming edges; no successful-state update
    # is moved onto the rejected path. Common HTTP response exists on both paths.
    labels = {e.get('id'): e.get('name') for e in ends}
    for edge in process.findall(tag('sequenceFlow')):
        if edge.get('targetRef') in labels:
            outcome = labels[edge.get('targetRef')]
            edge.set('name', (edge.get('name', '')+'; '+outcome).strip('; '))
            edge.set('targetRef', 'ResponseMerge')
    for e in ends:
        remove(process, e.get('id'))
    node(process, 'exclusiveGateway', 'ResponseMerge', 'Gabungkan hasil alternatif', system, gatewayDirection='Converging')
    response = node(process, 'sendTask', 'ReturnResponse', 'Kirim respons hasil aksi (sukses atau gagal)', system)
    ET.SubElement(response, tag('documentation')).text = (
        'Abstraction of inventory HTTP Handler.action: respond 409 with error message '
        'or respond 200 with document. Preserve outcome and status. No shared commit is introduced.')
    node(process, 'endEvent', 'ResponseDelivered', 'Respons dikirim', system)
    flow(process, 'MergeToResponse', 'ResponseMerge', 'ReturnResponse')
    flow(process, 'ResponseToEnd', 'ReturnResponse', 'ResponseDelivered')
    (output/'original.bpmn').write_bytes(data)
    baseline_hash = write(root, output/'before.bpmn')

    changed = copy.deepcopy(root)
    process = changed.find(tag('process'))
    system = process.findall('.//' + tag('lane'))[1]
    bridge = next(e for e in process.findall(tag('sequenceFlow')) if e.get('sourceRef')==prefix+'input' and e.get('targetRef')==prefix+'check')
    bridge.set('targetRef', 'ReasonDecision')
    node(process, 'exclusiveGateway', 'ReasonDecision', 'Reason supplied?', system, gatewayDirection='Diverging', default='ReasonMissing')
    node(process, 'serviceTask', 'RejectMissingReason', 'Prepare rejection: reason is required', system)
    node(process, 'parallelGateway', 'MasterChecksSplit', 'Run both master-data checks', system, gatewayDirection='Diverging')
    node(process, 'serviceTask', 'CheckWarehouse', 'Validate warehouse is still active', system)
    node(process, 'serviceTask', 'CheckProducts', 'Validate all products are still active', system)
    node(process, 'parallelGateway', 'MasterChecksJoin', 'Collect both validation results', system, gatewayDirection='Converging')
    node(process, 'serviceTask', 'CombineValidation', 'Aggregate results without changing stock or status', system)
    node(process, 'exclusiveGateway', 'MasterDecision', 'Both checks passed?', system, gatewayDirection='Diverging', default='ChecksFailed')
    node(process, 'serviceTask', 'RejectInvalidMaster', 'Prepare rejection: inactive master data', system)
    flow(process, 'ReasonPresent', 'ReasonDecision', 'MasterChecksSplit', 'reason is not empty')
    flow(process, 'ReasonMissing', 'ReasonDecision', 'RejectMissingReason')
    flow(process, 'ReasonRejected', 'RejectMissingReason', 'ResponseMerge')
    flow(process, 'CheckWarehouseStart', 'MasterChecksSplit', 'CheckWarehouse')
    flow(process, 'CheckProductsStart', 'MasterChecksSplit', 'CheckProducts')
    flow(process, 'WarehouseChecked', 'CheckWarehouse', 'MasterChecksJoin')
    flow(process, 'ProductsChecked', 'CheckProducts', 'MasterChecksJoin')
    flow(process, 'CombineChecks', 'MasterChecksJoin', 'CombineValidation')
    flow(process, 'DecideChecks', 'CombineValidation', 'MasterDecision')
    flow(process, 'ChecksPassed', 'MasterDecision', prefix+'check', 'warehouse active and all products active')
    flow(process, 'ChecksFailed', 'MasterDecision', 'RejectInvalidMaster')
    flow(process, 'MasterRejected', 'RejectInvalidMaster', 'ResponseMerge')
    changed_hash = write(changed, output/'after.bpmn')
    paths = ['backend/internal/inventory/adapter/http/handler.go', 'backend/internal/inventory/adapter/postgres/store.go']
    record = {'baseline_source': source, 'original_sha256': sha(data), 'before_sha256': baseline_hash, 'after_sha256': changed_hash,
              'status': 'source-reviewed presentation abstraction; not runtime equivalence proof',
              'source_hashes': {p: sha((repo/p).read_bytes()) for p in paths},
              'requirement': 'Submit requires a reason and rechecks active warehouse and products. Both checks return results before deciding; failure cannot submit. Existing state/actor/version guards remain.',
              'reference_status': 'No scored ground truth for this combined scenario yet. Do not use source_hashes as expected affected files.'}
    (output/'provenance.json').write_text(json.dumps(record, indent=2)+'\n', encoding='utf-8')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    build(args.repo, args.output)
