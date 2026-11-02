"""Create research-only BPMN pairs from read-only Staff baseline copies.
No ERP source execution. Standard-library XML; emits complete diagram shapes/edges.
"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import xml.etree.ElementTree as ET

NS = {'bpmn': 'http://www.omg.org/spec/BPMN/20100524/MODEL',
      'bpmndi': 'http://www.omg.org/spec/BPMN/20100524/DI',
      'dc': 'http://www.omg.org/spec/DD/20100524/DC', 'di': 'http://www.omg.org/spec/DD/20100524/DI',
      'xsi': 'http://www.w3.org/2001/XMLSchema-instance'}
for prefix, uri in NS.items(): ET.register_namespace(prefix, uri)
def tag(local): return '{'+NS['bpmn']+'}'+local
def q(prefix, local): return '{'+NS[prefix]+'}'+local
def sha(data): return hashlib.sha256(data).hexdigest()
def node(process, kind, id, name, lane=None, **attrs):
    value = ET.SubElement(process, tag(kind), {'id':id, 'name':name, **attrs})
    if lane is not None: ET.SubElement(lane, tag('flowNodeRef')).text=id
    return value
def flow(process, id, source, target, condition=None):
    value=ET.SubElement(process,tag('sequenceFlow'),{'id':id,'sourceRef':source,'targetRef':target})
    if condition:
        value.set('name',condition)
        ET.SubElement(value,tag('conditionExpression'),{q('xsi','type'):'bpmn:tFormalExpression'}).text=condition
    return value
def remove(process, id):
    for child in list(process):
        if child.get('id')==id: process.remove(child)
    for lane in process.findall('.//'+tag('lane')):
        for ref in list(lane):
            if ref.tag==tag('flowNodeRef') and ref.text==id: lane.remove(ref)
def refresh(process):
    nodes={n.get('id'):n for n in process if n.tag not in [tag('sequenceFlow'),tag('laneSet'),tag('documentation')]}
    for value in nodes.values():
        for child in list(value):
            if child.tag in [tag('incoming'),tag('outgoing')]: value.remove(child)
    for edge in process.findall(tag('sequenceFlow')):
        for attr,kind in [('sourceRef','outgoing'),('targetRef','incoming')]:
            if edge.get(attr) not in nodes: raise ValueError('dangling flow '+edge.get('id'))
            ET.SubElement(nodes[edge.get(attr)],tag(kind)).text=edge.get('id')
    return nodes
def diagram(root):
    process=root.find(tag('process')); nodes=refresh(process)
    edges=process.findall(tag('sequenceFlow')); lanes=process.findall('.//'+tag('lane'))
    memberships={ref.text:i for i,lane in enumerate(lanes) for ref in lane.findall(tag('flowNodeRef'))}
    if set(nodes)-set(memberships): raise ValueError('node missing lane')
    rank={id:0 for id in nodes}
    for _ in nodes:
        updated=False
        for id,value in nodes.items():
            host=value.get('attachedToRef')
            if host and rank[id]<rank[host]:rank[id]=rank[host];updated=True
        for edge in edges:
            a,b=edge.get('sourceRef'),edge.get('targetRef')
            if rank[b]<rank[a]+1: rank[b]=rank[a]+1; updated=True
        if not updated: break
    else: raise ValueError('scenario layout requires acyclic flow')
    positions={}; counts={}
    width=(max(rank.values())+1)*250+180; laneheight=470
    for id,value in nodes.items():
        lane=memberships[id]; key=(lane,rank[id]); row=counts.get(key,0);counts[key]=row+1
        kind=value.tag.split('}')[-1]
        w,h=(42,42) if kind.endswith('Event') else (55,55) if kind.endswith('Gateway') else (175,85)
        positions[id]=(170+rank[id]*250,110+lane*laneheight+row*145,w,h)
    for id,value in nodes.items():
        host=value.get('attachedToRef')
        if host:
            x,y,w,h=positions[host]
            positions[id]=(x+w-52,y+h-21,42,42)
    for old in list(root):
        if old.tag==q('bpmndi','BPMNDiagram'):root.remove(old)
    collab=root.find(tag('collaboration')); participant=collab.find(tag('participant'))
    plane=ET.SubElement(ET.SubElement(root,q('bpmndi','BPMNDiagram'),{'id':'StructuralDiagram'}),q('bpmndi','BPMNPlane'),{'id':'StructuralPlane','bpmnElement':collab.get('id')})
    def shape(id,bounds,horizontal=False):
        attrs={'id':id+'_di','bpmnElement':id}
        if horizontal:attrs['isHorizontal']='true'
        value=ET.SubElement(plane,q('bpmndi','BPMNShape'),attrs)
        ET.SubElement(value,q('dc','Bounds'),dict(zip(['x','y','width','height'],map(str,bounds))))
        return value
    shape(participant.get('id'),(20,40,width,laneheight*len(lanes)),True)
    for i,lane in enumerate(lanes):shape(lane.get('id'),(50,40+i*laneheight,width-30,laneheight),True)
    for id,bounds in positions.items():
        value=shape(id,bounds); kind=nodes[id].tag.split('}')[-1]
        if kind.endswith('Event') or kind.endswith('Gateway'):
            x,y,w,h=bounds
            ET.SubElement(ET.SubElement(value,q('bpmndi','BPMNLabel')),q('dc','Bounds'),{'x':str(x-50),'y':str(y+h+10),'width':'155','height':'60'})
    for edge in edges:
        a,b=positions[edge.get('sourceRef')],positions[edge.get('targetRef')]
        sx,sy=a[0]+a[2],a[1]+a[3]/2;tx,ty=b[0],b[1]+b[3]/2
        value=ET.SubElement(plane,q('bpmndi','BPMNEdge'),{'id':edge.get('id')+'_di','bpmnElement':edge.get('id')})
        for x,y in [(sx,sy),((sx+tx)/2,sy),((sx+tx)/2,ty),(tx,ty)]:ET.SubElement(value,q('di','waypoint'),{'x':str(x),'y':str(y)})
        if edge.get('name'):
            ET.SubElement(ET.SubElement(value,q('bpmndi','BPMNLabel')),q('dc','Bounds'),{'x':str((sx+tx)/2),'y':str(min(sy,ty)-40),'width':'180','height':'35'})

def generate(repo, output, names=None):
    specs=[]
    for name in names or ['remove-task','add-xor','remove-gateway','remove-gateway-control','move-role','add-and','add-or']:
        folder=output/name;folder.mkdir(parents=True,exist_ok=False)
        cancel=name=='remove-task';stem='role-staff-l3-adj-'+('cancel' if cancel else 'submit')
        baseline=repo/'docs/bpm-haqi/roles/staff/inventory/level-3'/f'{stem}.bpmn'
        data=baseline.read_bytes();(folder/'before.bpmn').write_bytes(data)
        root=ET.fromstring(data);process=root.find(tag('process'));prefix=stem.replace('-','_')
        get=lambda suffix:next(n for n in process if n.get('id')==prefix+'_'+suffix)
        staff,system=process.findall('.//'+tag('lane'))[:2]
        description=''; expected=[]
        if name=='remove-task':
            description='Cancellation no longer requires collecting a reason; keep creator, state and version guards. The cancel command still exists.'
            remove(process,prefix+'_input');remove(process,prefix+'_Flow_1');get('Flow_0').set('targetRef',prefix+'_check')
            get('check').set('name','DRAFT or APPROVED; matching version; creator or active Super Admin')
            expected=['frontend/features/inventory/inventory-adjustment-panel.tsx','backend/internal/inventory/adapter/postgres/store.go','backend/test/integration/operations_test.go']
        elif name=='move-role':
            description='Only Supervisor (plus existing Super Admin override) may submit stock adjustments; Staff can still prepare drafts. Other actions retain their permissions.'
            lane=ET.SubElement(process.find(tag('laneSet')),tag('lane'),{'id':'Lane_Submit_Supervisor','name':'Supervisor'})
            for ref in list(staff):
                if ref.tag==tag('flowNodeRef') and ref.text==prefix+'_input':staff.remove(ref);lane.append(ref)
            expected=['frontend/features/inventory/inventory-adjustment-panel.tsx','backend/internal/inventory/adapter/http/handler.go','backend/test/integration/operations_test.go']
        elif name in ['remove-gateway','remove-gateway-control']:
            if name=='remove-gateway-control':
                description='Notation-only simplification: keep validation and failure behaviour in the existing service task, show failure through a boundary error event instead of an explicit yes/no gateway. No source requirement changes.'
            else:
                description='Replace the explicit DRAFT-only eligibility gateway with service validation that allows DRAFT or APPROVED adjustment resubmission. Preserve version/access guards and failure rollback, clear previous reviewer on resubmit, and require a fresh approval without posting inventory. Other states remain forbidden.'
                get('check').set('name','Lock DRAFT or APPROVED adjustment; matching version; reset previous approval on resubmit')
                expected=['backend/internal/inventory/adapter/postgres/store.go','backend/test/integration/operations_test.go']
            remove(process,prefix+'_valid');remove(process,prefix+'_Flow_3');remove(process,prefix+'_Flow_6')
            get('Flow_2').set('targetRef',prefix+'_effect0')
            event=node(process,'boundaryEvent','ValidationFailure','Validation failed',system,attachedToRef=prefix+'_check',cancelActivity='true')
            ET.SubElement(event,tag('errorEventDefinition'),{'id':'ValidationFailureDefinition'})
            flow(process,'ValidationFailureFlow','ValidationFailure',prefix+'_failed')
        else:
            kind={'add-xor':'exclusiveGateway','add-and':'parallelGateway','add-or':'inclusiveGateway'}[name]
            split='ResearchSplit';join='ResearchJoin'
            node(process,kind,split,{'add-xor':'Is stock adjustment reason provided?','add-and':'Run both submit checks','add-or':'Select checks for stock delta signs'}[name],system,gatewayDirection='Diverging')
            get('Flow_1').set('targetRef',split)
            if name=='add-xor':
                description='Require a non-empty reason on adjustment submit; reject without changing state if absent. Existing reason request/storage fields are reused.'
                flow(process,'ReasonProvided',split,prefix+'_check','reason is not empty')
                flow(process,'ReasonMissing',split,prefix+'_failed','reason is empty')
                get('effect0') # retain existing state transition
                expected=['frontend/features/inventory/inventory-adjustment-panel.tsx','backend/internal/inventory/adapter/postgres/store.go','backend/test/integration/operations_test.go']
            else:
                node(process,kind,join,'All selected checks completed',system,gatewayDirection='Converging')
                if name=='add-and':
                    description='Before submit, validate the warehouse and all products against current active master data in independent branches; both must complete successfully. Failure rolls back; no partial submission. No requirement to use goroutines.'
                    labels=['Validate active warehouse for stock adjustment','Validate active products for stock adjustment'];conditions=[None,None]
                else:
                    description='Before submit, run active-product validation for positive deltas and available-stock validation for negative deltas. Mixed-sign lines activate both branches; wait for all selected checks. Zero-only lines take the default bypass. Failures roll back.'
                    labels=['Validate active products for positive stock deltas','Validate available stock for negative stock deltas'];conditions=['any counted quantity exceeds snapshot quantity','any counted quantity is below snapshot quantity']
                for i,label in enumerate(labels):
                    id='ResearchCheck'+str(i);node(process,'serviceTask',id,label,system)
                    flow(process,'CheckBranch'+str(i),split,id,conditions[i]);flow(process,'CheckJoin'+str(i),id,join)
                if name=='add-or':
                    get_split=next(n for n in process if n.get('id')==split);get_split.set('default','ZeroDeltaBypass')
                    flow(process,'ZeroDeltaBypass',split,join)
                flow(process,'ChecksCompleted',join,prefix+'_check')
                expected=['backend/internal/inventory/adapter/postgres/store.go','backend/test/integration/operations_test.go']
        diagram(root);ET.indent(root)
        after=ET.tostring(root,encoding='utf-8',xml_declaration=True);(folder/'after.bpmn').write_bytes(after)
        record={'scenario':name,'requirement':description,'reference_status':'provisional-source-review; not implementation proof','baseline_sha256':sha(data),'changed_sha256':sha(after),'expected_existing_files':expected,'evaluation_scope':'file union across primary changes; atomic references pending parser validation','source_hashes':{p:sha((repo/p).read_bytes()) for p in expected}}
        (folder/'requirements-and-reference.json').write_text(json.dumps(record,indent=2)+'\n',encoding='utf-8')
        (folder/'README.md').write_text('# '+name+'\n\n'+description+'\n\nBefore: `before.bpmn`. After: `after.bpmn`. Repository: local Simple-ERP, read-only.\nReference: provisional source review; do not upload reference JSON as inference input.\nComplete DI is regenerated for the changed diagram; coordinates are not business changes.\n',encoding='utf-8')
        specs.append(record)
    index=output/'scenarios.json'
    if index.exists(): specs=json.loads(index.read_text(encoding='utf-8'))+specs
    index.write_text(json.dumps(specs,indent=2)+'\n',encoding='utf-8')
    return specs

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--scenario',action='append',choices=['remove-task','add-xor','remove-gateway','remove-gateway-control','move-role','add-and','add-or'])
    args=parser.parse_args();generate(args.repo,args.output,args.scenario)
