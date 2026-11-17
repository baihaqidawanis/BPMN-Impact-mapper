"""Prepare reviewed BPMN-only edits; collapse exclusive result merges into existing activities.

No business task is invented for the common case. Activity incoming flows form an
implicit merge only for these non-concurrent alternative outcomes. Apply is
explicit, manifest-bounded, and checks the current canonical hashes first.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from xml.dom import minidom

from normalize_staff_bpmn import B, BD, DC, DI, BIO, children, attr, delete, create, bounds, set_bounds


def digest(data):
    return hashlib.sha256(data).hexdigest()


def normalize(data):
    doc = minidom.parseString(data)
    process = doc.getElementsByTagNameNS(B, 'process')
    if len(process) != 1:
        raise ValueError('Expected exactly one process')
    p = process[0]
    nodes = {attr(n, 'id'): n for n in children(p) if n.localName not in ('sequenceFlow', 'laneSet', 'documentation')}
    flows = children(p, 'sequenceFlow')
    direct = [f for f in flows if nodes[attr(f, 'sourceRef')].localName.endswith('Gateway') and
              (nodes[attr(f, 'targetRef')].localName.endswith('Gateway') or nodes[attr(f, 'targetRef')].localName == 'endEvent')]
    if not direct:
        return data, {'changed': False, 'removed_merges': [], 'added_activities': []}
    if any(n.localName in ('parallelGateway', 'inclusiveGateway', 'eventBasedGateway', 'complexGateway', 'subProcess', 'boundaryEvent') for n in nodes.values()):
        raise ValueError('Concurrent/nested/event semantics require individual review')
    plane = doc.getElementsByTagNameNS(BD, 'BPMNPlane')[0]
    shapes = {attr(n, 'bpmnElement'): n for n in children(plane, 'BPMNShape')}
    edges = {attr(n, 'bpmnElement'): n for n in children(plane, 'BPMNEdge')}
    lanes = list(p.getElementsByTagNameNS(B, 'lane'))
    targets = sorted({attr(f, 'targetRef') for f in direct})
    removed, added = [], []
    for target_id in targets:
        target = nodes[target_id]
        if target_id.endswith('_result_merge'):
            if target.localName != 'exclusiveGateway' or children(target, 'extensionElements'):
                raise ValueError('Not a plain exclusive result merge')
            outgoing = [f for f in flows if attr(f, 'sourceRef') == target_id]
            incoming = [f for f in flows if attr(f, 'targetRef') == target_id]
            if len(outgoing) != 1 or len(incoming) < 2:
                raise ValueError('Unexpected merge topology')
            tail = outgoing[0]
            activity_id = attr(tail, 'targetRef')
            activity = nodes[activity_id]
            if activity_id != target_id.removesuffix('_merge')+'_activity' or activity.localName not in ('sendTask', 'userTask'):
                raise ValueError('Expected existing outcome activity')
            if attr(tail, 'name') or children(tail, 'conditionExpression') or attr(target, 'default'):
                raise ValueError('Result merge has decision semantics')
            box = bounds(shapes[activity_id]); tx, ty = box[0], box[1]+box[3]/2
            for flow in incoming:
                flow.setAttribute('targetRef', activity_id)
                # Keep original business routes; extend their last segment across
                # the removed merge to the unchanged activity bounds.
                edge = edges[attr(flow, 'id')]
                points = children(edge, 'waypoint')
                lx, ly = float(attr(points[-1], 'x')), float(attr(points[-1], 'y'))
                if ly != ty: create(edge, DI, 'di:waypoint', {'x': lx, 'y': ty})
                create(edge, DI, 'di:waypoint', {'x': tx, 'y': ty})
            for lane in lanes:
                for ref in children(lane, 'flowNodeRef'):
                    if ''.join(c.data for c in ref.childNodes if c.nodeType == c.TEXT_NODE) == target_id: delete(ref)
            delete(edges[attr(tail, 'id')]); delete(tail); delete(shapes[target_id]); delete(target)
            removed.append(target_id)
        elif target.localName == 'endEvent' and attr(p, 'id') == 'Process_role_staff_l2_ap':
            # The current Staff AP model ends immediately after PO CLOSED.
            # Make viewing the existing finance response/status explicit, without
            # adding an approval, request, state change or receipt acknowledgement.
            doc.documentElement.setAttribute('xmlns:bioc', BIO)
            incoming = [f for f in flows if attr(f, 'targetRef') == target_id]
            if len(incoming) != 1: raise ValueError('Review AP end topology')
            flow = incoming[0]; tid = target_id+'_review_result'
            task = create(p, B, 'bpmn:userTask', {'id': tid, 'name': 'Periksa hasil posting pembayaran supplier'})
            create(task, B, 'bpmn:documentation', text='Existing UI observation only: finance-workspace.tsx displays the transaction result and payment status. No new approval, API call, state mutation or guaranteed extra user click.')
            lane = next(l for l in lanes if any(''.join(c.data for c in r.childNodes if c.nodeType == c.TEXT_NODE) == target_id for r in children(l, 'flowNodeRef')))
            create(lane, B, 'bpmn:flowNodeRef', text=tid)
            box = bounds(shapes[target_id]); cy = box[1]+box[3]/2; x = box[0]
            shape = create(plane, BD, 'bpmndi:BPMNShape', {'id': tid+'_di', 'bpmnElement': tid})
            shape.setAttributeNS(BIO, 'bioc:stroke', '#9c27b0'); shape.setAttributeNS(BIO, 'bioc:fill', '#f3e5f5')
            create(shape, DC, 'dc:Bounds', {'x': x, 'y': cy-40, 'width': 200, 'height': 80})
            flow.setAttribute('targetRef', tid)
            fid = tid+'_to_end'
            create(p, B, 'bpmn:sequenceFlow', {'id': fid, 'sourceRef': tid, 'targetRef': target_id})
            edge = create(plane, BD, 'bpmndi:BPMNEdge', {'id': fid+'_di', 'bpmnElement': fid})
            for px, py in [(x+200, cy), (x+270, cy)]: create(edge, DI, 'di:waypoint', {'x': px, 'y': py})
            set_bounds(shapes[target_id], (x+270, box[1], box[2], box[3]))
            for label in children(shapes[target_id], 'BPMNLabel'):
                old = bounds(label); set_bounds(label, (old[0]+270, old[1], old[2], old[3]))
            for shape in shapes.values():
                if attr(shape, 'bpmnElement') in {attr(l, 'id') for l in lanes} or attr(shape, 'isHorizontal') == 'true':
                    old = bounds(shape); set_bounds(shape, (old[0], old[1], max(old[2],x+370-old[0]),old[3]))
            added.append(tid)
        else:
            raise ValueError('Unreviewed direct gateway target: '+target_id)
    # Refresh explicit references after removing merge nodes/flows.
    current_flows = children(p, 'sequenceFlow')
    for n in children(p):
        if n.localName in ('sequenceFlow', 'laneSet', 'documentation'): continue
        for ref in children(n):
            if ref.localName in ('incoming', 'outgoing'): delete(ref)
        for flow in current_flows:
            if attr(flow, 'targetRef') == attr(n, 'id'): create(n, B, 'bpmn:incoming', text=attr(flow, 'id'))
            if attr(flow, 'sourceRef') == attr(n, 'id'): create(n, B, 'bpmn:outgoing', text=attr(flow, 'id'))
    def strip(node):
        for child in list(node.childNodes):
            if child.nodeType == child.TEXT_NODE and not child.data.strip(): delete(child)
            elif child.nodeType == child.ELEMENT_NODE: strip(child)
    strip(doc.documentElement)
    return doc.toprettyxml(indent='  ', encoding='UTF-8'), {'changed': True, 'removed_merges': removed, 'added_activities': added}


def prepare(repo, out):
    source = repo/'docs/bpm-haqi/roles'
    files = sorted(source.rglob('*.bpmn'))
    if len(files) != 112: raise ValueError('Review changed role inventory')
    out.mkdir(parents=True, exist_ok=False)
    records = []
    for path in files:
        rel = path.relative_to(source); before = path.read_bytes(); after, detail = normalize(before)
        for folder, data in [('backup', before), ('prepared', after)]:
            dest = out/folder/rel; dest.parent.mkdir(parents=True, exist_ok=True); dest.write_bytes(data)
        records.append({'file': rel.as_posix(), 'before_sha256': digest(before), 'after_sha256': digest(after), **detail})
    (out/'manifest.json').write_text(json.dumps(records, indent=2)+'\n', encoding='utf-8')
    print('Prepared', len(records), 'files;', sum(r['changed'] for r in records), 'changed.')


def apply(repo, out):
    root = (repo/'docs/bpm-haqi/roles').resolve()
    operations = []
    records = json.loads((out/'manifest.json').read_text(encoding='utf-8'))
    if len(records) != 112 or len({r['file'] for r in records}) != 112: raise ValueError('Invalid inventory')
    for r in records:
        rel = Path(r['file']); target = root/rel
        if rel.is_absolute() or '..' in rel.parts or rel.suffix != '.bpmn' or not target.resolve().is_relative_to(root) or any(p.is_symlink() for p in [target,*target.parents]): raise ValueError('Unsafe target')
        data = (out/'prepared'/rel).read_bytes()
        if digest(target.read_bytes()) != r['before_sha256'] or digest(data) != r['after_sha256']: raise ValueError('File changed since review: '+str(rel))
        if r['changed']: operations.append((target,data))
    for target,data in operations:
        temporary=target.with_suffix('.bpmn.reviewed')
        with temporary.open('xb') as handle: handle.write(data)
        os.replace(temporary,target)
    for r in records:
        if digest((root/r['file']).read_bytes()) != r['after_sha256']: raise ValueError('Post-apply hash mismatch')
    print('Applied',len(operations),'BPMN-only changes; verified all',len(records),'hashes.')


if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--output',type=Path,required=True);parser.add_argument('--apply',action='store_true')
    args=parser.parse_args()
    (apply if args.apply else prepare)(args.repo,args.output)
