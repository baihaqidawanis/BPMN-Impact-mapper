"""Verify the active case-study input bundle without inference or ERP execution."""
import argparse
import importlib.util
import json
from pathlib import Path
import xml.etree.ElementTree as ET

from build_structural_scenarios import tag, q, sha

BASELINE='docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn'


def verify(repo,pack):
    manifest=json.loads((pack/'manifest.json').read_text(encoding='utf-8'))
    baseline=(repo/BASELINE).read_bytes()
    assert len(manifest['scenarios'])==5
    records=[]
    for case in manifest['scenarios']:
        before=Path(case['old_bpmn_path']);after=Path(case['new_bpmn_path']);reference=Path(case['ground_truth_path'])
        for path,key in [(before,'old_bpmn_sha256'),(after,'new_bpmn_sha256'),(reference,'ground_truth_sha256')]:
            assert path.resolve().is_relative_to(pack.resolve()),'Input outside bundle'
            assert sha(path.read_bytes())==case[key],('Hash drift',path)
        assert before.read_bytes()==baseline,('Baseline drift',before)
        source=json.loads((before.parent/'source-review.json').read_text(encoding='utf-8'))
        for rel,expected in source['source_hashes'].items():assert sha((repo/rel).read_bytes())==expected,('Source drift',rel)
        expected=json.loads(reference.read_text(encoding='utf-8'))
        for item in expected['cases']:
            for target in item['expected']:assert (repo/target['file']).is_file(),('Missing reference file',target['file'])
        for path in [before,after]:
            root=ET.fromstring(path.read_bytes());process=root.find(tag('process'))
            nodes={e.get('id'):e for e in process if e.get('id') and e.tag not in [tag('laneSet'),tag('sequenceFlow')]}
            flows=process.findall(tag('sequenceFlow'))
            shape_ids=[s.get('bpmnElement') for s in root.iter(q('bpmndi','BPMNShape'))]
            edges={e.get('bpmnElement'):e for e in root.iter(q('bpmndi','BPMNEdge'))}
            assert set(edges)=={f.get('id') for f in flows}
            for id in nodes:assert shape_ids.count(id)==1,('Missing shape',path,id)
            for flow in flows:
                assert flow.get('sourceRef') in nodes and flow.get('targetRef') in nodes
                points=list(edges[flow.get('id')]);points=[p for p in points if p.tag==q('di','waypoint')]
                assert len(points)>=2
                assert any(a.attrib!=b.attrib for a,b in zip(points,points[1:]))
        # A split needs branches; a join legitimately has one outgoing flow.
        for n in nodes.values():
            if n.tag.endswith('Gateway'):
                ins=sum(f.get('targetRef')==n.get('id') for f in flows)
                outs=sum(f.get('sourceRef')==n.get('id') for f in flows)
                if n.get('gatewayDirection')=='Diverging':assert outs>=2,('Split missing branches',after,n.get('id'))
                if n.get('gatewayDirection')=='Converging':assert ins>=2,('Join missing inputs',after,n.get('id'))
        if not case['id'].endswith('unchanged-control'):
            bounds={v.get('bpmnElement'):v.find(q('dc','Bounds')) for v in root.iter(q('bpmndi','BPMNShape'))}
            for f in flows:
                points=[(float(p.get('x')),float(p.get('y'))) for p in edges[f.get('id')] if p.tag==q('di','waypoint')]
                for a,b in zip(points,points[1:]):
                    assert a[0]==b[0] or a[1]==b[1],('Non-orthogonal flow',f.get('id'))
                    for id in nodes:
                        if id in (f.get('sourceRef'),f.get('targetRef')):continue
                        box=bounds[id];x,y,w,h=(float(box.get(k)) for k in ('x','y','width','height'))
                        collision=(a[0]==b[0] and x<a[0]<x+w and max(min(a[1],b[1]),y)<min(max(a[1],b[1]),y+h)) or (a[1]==b[1] and y<a[1]<y+h and max(min(a[0],b[0]),x)<min(max(a[0],b[0]),x+w))
                        assert not collision,('Flow crosses unrelated node',after,f.get('id'),id)
        if case['id'].endswith('unchanged-control'):assert before.read_bytes()==after.read_bytes()
        else:assert before.read_bytes()!=after.read_bytes()
        records.append({'id':case['id'],'baseline_sha256':sha(before.read_bytes()),'changed_sha256':sha(after.read_bytes())})
    module_path=Path(__file__).resolve().parents[2]/'scripts/audit_bpmn_conventions.py'
    spec=importlib.util.spec_from_file_location('conventions',module_path);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    audit=module.audit(pack)
    assert audit['files_checked']==10 and audit['finding_count']==0,audit
    return {'baseline_path':BASELINE,'baseline_sha256':sha(baseline),'pairs':records,'bpmn_files_checked':10,'strict_findings':0,'reference_status':'provisional source-derived diagnostic; not independent ground truth'}


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--pack',type=Path,required=True)
    args=parser.parse_args();value=verify(args.repo,args.pack)
    print(json.dumps(value,indent=2))
