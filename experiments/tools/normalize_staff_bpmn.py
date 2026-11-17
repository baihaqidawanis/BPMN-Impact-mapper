"""Prepare/apply a bounded Staff document normalization, never ERP code.

Use minidom to preserve namespace declarations referenced only by QName attribute
values such as calledElement. Frozen experiment pairs are deliberately excluded.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from xml.dom import minidom

B = 'http://www.omg.org/spec/BPMN/20100524/MODEL'
BD = 'http://www.omg.org/spec/BPMN/20100524/DI'
DC = 'http://www.omg.org/spec/DD/20100524/DC'
DI = 'http://www.omg.org/spec/DD/20100524/DI'
BIO = 'http://bpmn.io/schema/bpmn/biocolor/1.0'
ROLE_COUNTS = {'staff': 32, 'supervisor': 31, 'super-admin': 49}

def role_root(role):
    if role not in ROLE_COUNTS: raise ValueError('Unsupported role')
    return Path('docs/bpm-haqi/roles') / role


def digest(data): return hashlib.sha256(data).hexdigest()
def children(e, local=None):
    return [n for n in e.childNodes if n.nodeType == n.ELEMENT_NODE and (local is None or n.localName == local)]
def attr(e, name): return e.getAttribute(name)
def delete(e): e.parentNode.removeChild(e)
def create(parent, ns, name, attrs=None, text=None):
    e = parent.ownerDocument.createElementNS(ns, name)
    for k, v in (attrs or {}).items(): e.setAttribute(k, str(v))
    if text is not None: e.appendChild(parent.ownerDocument.createTextNode(text))
    parent.appendChild(e)
    return e
def bounds(shape):
    b = children(shape, 'Bounds')[0]
    return tuple(float(attr(b, k)) for k in ('x', 'y', 'width', 'height'))
def set_bounds(shape, value):
    b = children(shape, 'Bounds')[0]
    for k, v in zip(('x', 'y', 'width', 'height'), value): b.setAttribute(k, str(v))


def normalize(data):
    doc = minidom.parseString(data)
    processes = doc.getElementsByTagNameNS(B, 'process')
    if len(processes) != 1: raise ValueError('Expected one Staff process')
    p = processes[0]
    starts, ends = children(p, 'startEvent'), children(p, 'endEvent')
    if len(starts) != 1 or not ends: raise ValueError('Review start/end semantics manually')
    if any(children(e) and any(n.localName.endswith('EventDefinition') for n in children(e)) for e in ends):
        raise ValueError('Cannot normalize special end events')
    flows = children(p, 'sequenceFlow')
    nodes = {attr(n, 'id'): n for n in children(p) if n.localName not in ('sequenceFlow', 'laneSet', 'documentation')}
    incoming = {id: [f for f in flows if attr(f, 'targetRef') == id] for id in nodes}
    joins = [n for id, n in nodes.items() if n.localName.endswith('Gateway') and len(incoming[id]) > 1]
    direct_join = [f for f in flows if attr(f, 'sourceRef') in {attr(n, 'id') for n in joins} and attr(f, 'targetRef') in {attr(n, 'id') for n in ends}]
    if len(ends) == 1 and not direct_join: return data, {'changed': False}
    # These source models have exclusively alternative exits. Never apply this
    # merge rule to concurrency, event subprocesses or boundary error semantics.
    if any(n.localName in ('parallelGateway', 'inclusiveGateway', 'subProcess', 'boundaryEvent') for n in nodes.values()):
        raise ValueError('Unsupported concurrency or nested-scope normalization')
    plane = doc.getElementsByTagNameNS(BD, 'BPMNPlane')[0]
    shapes = {attr(n, 'bpmnElement'): n for n in children(plane, 'BPMNShape')}
    edges = {attr(n, 'bpmnElement'): n for n in children(plane, 'BPMNEdge')}
    lanes = list(p.getElementsByTagNameNS(B, 'lane'))
    is_l3 = '_l3_' in attr(p, 'id')
    actor = ('Super Admin' if 'role_super_admin_' in attr(p, 'id') else
             'Supervisor' if 'role_supervisor_' in attr(p, 'id') else 'Staff')
    lane = next(n for n in lanes if ('Sistem' in attr(n, 'name') if is_l3 else attr(n, 'name') == actor))
    lane_bounds = bounds(shapes[attr(lane, 'id')])
    end_ids = {attr(n, 'id') for n in ends}
    # Prefer existing canonical success/end ID; retain its node type.
    end = next((n for n in ends if attr(n, 'id').endswith(('_done', '_end'))), ends[0])
    end_id = attr(end, 'id')
    prefix = attr(starts[0], 'id').removesuffix('_start')
    merge_id, activity_id = prefix+'_result_merge', prefix+'_result_activity'
    if merge_id in nodes or activity_id in nodes: raise ValueError('Normalization ID already used')
    # Tail must also clear former end positions: existing routes terminate there.
    right = max(bounds(shapes[id])[0]+bounds(shapes[id])[2] for id in nodes)
    x = right + 130
    cy = lane_bounds[1] + lane_bounds[3]/2
    y = cy - 27
    outcomes = {attr(n, 'id'): attr(n, 'name') for n in ends}

    def flow(id, source, target): return create(p, B, 'bpmn:sequenceFlow', {'id':id,'sourceRef':source,'targetRef':target})
    def node(kind, id, name):
        e = create(p, B, 'bpmn:'+kind, {'id':id, 'name':name})
        create(lane, B, 'bpmn:flowNodeRef', text=id)
        return e
    def shape(id, box, color, fill, label=None):
        e = create(plane, BD, 'bpmndi:BPMNShape', {'id':id+'_di','bpmnElement':id})
        e.setAttributeNS(BIO,'bioc:stroke',color);e.setAttributeNS(BIO,'bioc:fill',fill)
        create(e, DC, 'dc:Bounds', dict(zip(('x','y','width','height'),box)))
        if label:
            lab=create(e, BD, 'bpmndi:BPMNLabel')
            create(lab, DC, 'dc:Bounds', {'x':box[0]-45,'y':box[1]+box[3]+10,'width':150,'height':45})
        return e
    def edge(id, points):
        e = create(plane, BD, 'bpmndi:BPMNEdge', {'id':id+'_di','bpmnElement':id})
        for a,b in points: create(e, DI, 'di:waypoint', {'x':a,'y':b})
        return e

    doc.documentElement.setAttribute('xmlns:bioc', BIO)
    multiple = len(ends) > 1
    if multiple:
        merge = node('exclusiveGateway', merge_id, 'Gabungkan hasil alternatif')
        merge.setAttribute('gatewayDirection','Converging')
        shape(merge_id, (x,y,54,54),'#ffc107','#fff8e1',True)
        terminal_flows = [f for f in flows if attr(f,'targetRef') in end_ids]
        for i,f in enumerate(terminal_flows):
            outcome = outcomes[attr(f,'targetRef')]
            original_name = attr(f,'name')
            f.setAttribute('name', original_name if outcome in original_name else (original_name+'; '+outcome).strip('; '))
            f.setAttribute('targetRef',merge_id)
            e = edges[attr(f,'id')]
            points=children(e,'waypoint')
            last_x,last_y=float(attr(points[-1],'x')),float(attr(points[-1],'y'))
            rail=x-35-12*i
            for a,b in [(rail,last_y),(rail,cy),(x,cy)]:create(e,DI,'di:waypoint',{'x':a,'y':b})
            for lab in children(e,'BPMNLabel'):delete(lab)
            lab=create(e,BD,'bpmndi:BPMNLabel')
            create(lab,DC,'dc:Bounds',{'x':last_x-160,'y':last_y+12,'width':195,'height':60})
        ax=x+160
    else:
        # Existing join and all its incoming routes stay exactly as drawn.
        ax=x
        for f in direct_join:
            f.setAttribute('targetRef',activity_id)
            e=edges[attr(f,'id')]
            for point in children(e,'waypoint'):delete(point)
            source= bounds(shapes[attr(f,'sourceRef')]);sx,sy=source[0]+source[2],source[1]+source[3]/2
            for a,b in [(sx,sy),(ax-40,sy),(ax-40,cy),(ax,cy)]:create(e,DI,'di:waypoint',{'x':a,'y':b})
    title = 'Kirim respons hasil aksi (sukses atau gagal)' if is_l3 else 'Periksa hasil dan status proses'
    activity=node('sendTask' if is_l3 else 'userTask',activity_id,title)
    create(activity,B,'bpmn:documentation',text=(
        'Explicit presentation of existing response handling. Preserve success/error status and payload; no extra commit or state change.' if is_l3 else
        actor+' reads the existing result/status/message in the UI. This makes the existing review-of-result visible; no new approval, API request or state change.'))
    shape(activity_id,(ax,cy-45,200,90),'#ff9800' if is_l3 else '#9c27b0','#fff3e0' if is_l3 else '#f3e5f5')
    for old in ends:
        delete(shapes[attr(old,'id')])
        if old is not end:delete(old)
    for ln in lanes:
        for ref in children(ln,'flowNodeRef'):
            if ''.join(c.data for c in ref.childNodes if c.nodeType==c.TEXT_NODE) in end_ids:delete(ref)
    create(lane,B,'bpmn:flowNodeRef',text=end_id)
    end.setAttribute('name','Respons dikirim' if is_l3 else 'Hasil proses diperiksa')
    ex=ax+290
    shape(end_id,(ex,cy-19,38,38),'#e53935','#ffcdd2',True)
    if multiple:
        id=prefix+'_result_to_activity';flow(id,merge_id,activity_id);edge(id,[(x+54,cy),(ax,cy)])
    id=prefix+'_result_to_end';flow(id,activity_id,end_id);edge(id,[(ax+200,cy),(ex,cy)])
    # Only extend existing pools/lanes horizontally; original activity layout stays.
    for s in children(plane,'BPMNShape'):
        if attr(s,'bpmnElement') not in nodes and attr(s,'bpmnElement') not in (merge_id,activity_id):
            box=bounds(s)
            if attr(s,'bpmnElement') in {attr(l,'id') for l in lanes} or attr(s,'isHorizontal')=='true':
                set_bounds(s,(box[0],box[1],max(box[2],ex+100-box[0]),box[3]))
    all_flows=children(p,'sequenceFlow')
    for n in children(p):
        if n.localName in ('sequenceFlow','laneSet','documentation'):continue
        for ref in children(n):
            if ref.localName in ('incoming','outgoing'):delete(ref)
        for f in all_flows:
            if attr(f,'targetRef')==attr(n,'id'):create(n,B,'bpmn:incoming',text=attr(f,'id'))
            if attr(f,'sourceRef')==attr(n,'id'):create(n,B,'bpmn:outgoing',text=attr(f,'id'))
    def strip(e):
        for n in list(e.childNodes):
            if n.nodeType==n.TEXT_NODE and not n.data.strip():delete(n)
            elif n.nodeType==n.ELEMENT_NODE:strip(n)
    strip(doc.documentElement)
    result=doc.toprettyxml(indent='  ',encoding='UTF-8')
    return result, {'changed':True,'original_outcomes':outcomes,'activity':title,'lane':attr(lane,'name')}


def prepare(repo, out, role='staff'):
    source=repo/role_root(role)
    paths=sorted(source.rglob('*.bpmn'))
    if len(paths)!=ROLE_COUNTS[role]:raise ValueError('Role inventory changed; review expected file count')
    out.mkdir(parents=True,exist_ok=False)
    records=[]
    for path in paths:
        relative=path.relative_to(source)
        original=path.read_bytes();updated,details=normalize(original)
        for folder,data in [('backup',original),('prepared',updated)]:
            target=out/folder/relative;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(data)
        records.append({'file':relative.as_posix(),'before_sha256':digest(original),'after_sha256':digest(updated),**details})
    (out/'manifest.json').write_text(json.dumps(records,indent=2,ensure_ascii=False)+'\n',encoding='utf-8')
    print(f'Prepared {len(records)} {role} files; changed {sum(r["changed"] for r in records)}.')


def apply(repo,out,role='staff'):
    source=(repo/role_root(role)).resolve()
    records=json.loads((out/'manifest.json').read_text(encoding='utf-8'))
    operations=[]
    for r in records:
        rel=Path(r['file'])
        if rel.is_absolute() or '..' in rel.parts or rel.suffix!='.bpmn':raise ValueError('Invalid manifest path')
        target=source/rel
        if not target.resolve().is_relative_to(source) or any(p.is_symlink() for p in [target,*target.parents]):raise ValueError('Unsafe target')
        data=(out/'prepared'/rel).read_bytes()
        if digest(target.read_bytes())!=r['before_sha256'] or digest(data)!=r['after_sha256']:raise ValueError('File changed since review: '+str(rel))
        if r['changed']:operations.append((target,data))
    for target,data in operations:
        temp=target.with_suffix('.bpmn.normalizing')
        with temp.open('xb') as f:f.write(data)
        os.replace(temp,target)
    for r in records:
        if digest((source/r['file']).read_bytes())!=r['after_sha256']:raise ValueError('Applied hash mismatch')
    print(f'Replaced {len(operations)} canonical {role} BPMNs. All {len(records)} hashes verified.')


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--output',type=Path,required=True);parser.add_argument('--apply',action='store_true');parser.add_argument('--role',choices=ROLE_COUNTS,default='staff')
    args=parser.parse_args()
    (apply if args.apply else prepare)(args.repo,args.output,args.role)
