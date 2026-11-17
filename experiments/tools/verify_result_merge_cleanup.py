"""Verify a prepared cleanup preserves the old business graph modulo documented presentation nodes."""
import argparse
import json
from pathlib import Path
from xml.dom import minidom
import xml.etree.ElementTree as ET

from normalize_staff_bpmn import B, BD, children, attr
from remove_redundant_result_merges import digest, normalize


def verify(before, after, record):
    old, new = minidom.parseString(before), minidom.parseString(after)
    op, np = old.getElementsByTagNameNS(B,'process')[0], new.getElementsByTagNameNS(B,'process')[0]
    excluded = ('sequenceFlow','laneSet','documentation')
    old_nodes={attr(n,'id'):n for n in children(op) if n.localName not in excluded}
    new_nodes={attr(n,'id'):n for n in children(np) if n.localName not in excluded}
    removed,added=set(record['removed_merges']),set(record['added_activities'])
    assert set(old_nodes)-removed == set(new_nodes)-added
    attrs=lambda n: sorted((n.attributes.item(i).namespaceURI or "",n.attributes.item(i).name,n.attributes.item(i).value) for i in range(n.attributes.length))
    old_root=dict((name,value) for _,name,value in attrs(old.documentElement))
    new_root=dict((name,value) for _,name,value in attrs(new.documentElement))
    if added: old_root['xmlns:bioc']='http://bpmn.io/schema/bpmn/biocolor/1.0'
    assert old_root==new_root,'Namespace/root changed'
    for id,n in old_nodes.items():
        if id not in removed:
            assert n.localName==new_nodes[id].localName and attrs(n)==attrs(new_nodes[id]),('Business node changed',id)
    old_flows={attr(f,'id'):f for f in children(op,'sequenceFlow')}
    new_flows={attr(f,'id'):f for f in children(np,'sequenceFlow')}
    removed_flows={id for id,f in old_flows.items() if attr(f,'sourceRef') in removed}
    extra_flows={id+'_to_end' for id in added}
    assert set(old_flows)-removed_flows == set(new_flows)-extra_flows
    for id,f in old_flows.items():
        if id in removed_flows:continue
        current=new_flows[id]
        expected_target=attr(f,'targetRef')
        if expected_target in removed:
            tail=next(t for t in old_flows.values() if attr(t,'sourceRef')==expected_target)
            expected_target=attr(tail,'targetRef')
        if added and expected_target=='role_staff_l2_ap_end':expected_target=next(iter(added))
        expected=dict((name,value) for _,name,value in attrs(f));expected['targetRef']=expected_target
        actual=dict((name,value) for _,name,value in attrs(current))
        assert expected==actual,('Flow attribute changed',id)
        assert [c.toxml() for c in children(f)]==[c.toxml() for c in children(current)],('Condition changed',id)
    lane_members=lambda doc: {attr(l,'id'):{''.join(c.data for c in r.childNodes if c.nodeType==c.TEXT_NODE) for r in children(l,'flowNodeRef')} for l in doc.getElementsByTagNameNS(B,'lane')}
    old_lanes,new_lanes=lane_members(old),lane_members(new)
    for lane,members in old_lanes.items():assert members-removed==new_lanes[lane]-added,('Lane changed',lane)
    # Every node has exactly one shape and lane, every flow one nonempty edge.
    shapes=[attr(s,'bpmnElement') for s in new.getElementsByTagNameNS(BD,'BPMNShape')]
    edges={attr(e,'bpmnElement'):e for e in new.getElementsByTagNameNS(BD,'BPMNEdge')}
    for id in new_nodes:
        assert shapes.count(id)==1,('Missing/duplicate shape',id)
        assert sum(id in members for members in new_lanes.values())==1,('Lane membership',id)
    assert set(edges)==set(new_flows)
    for id,e in edges.items():
        points=children(e,'waypoint');assert len(points)>=2,('Missing edge geometry',id)
        assert any(attr(a,'x')!=attr(b,'x') or attr(a,'y')!=attr(b,'y') for a,b in zip(points,points[1:])),('Zero length edge',id)
    ids=[attr(e,'id') for e in new.getElementsByTagName('*') if attr(e,'id')]
    assert len(ids)==len(set(ids)),'Duplicate XML ID'
    assert normalize(after)[0]==after,'Cleanup is not idempotent'


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('output',type=Path);args=parser.parse_args()
    records=json.loads((args.output/'manifest.json').read_text(encoding='utf-8'))
    for r in records:
        before=(args.output/'backup'/r['file']).read_bytes();after=(args.output/'prepared'/r['file']).read_bytes()
        assert digest(before)==r['before_sha256'] and digest(after)==r['after_sha256']
        verify(before,after,r)
    print('Verified',len(records),'diagrams: business attributes/conditions/lanes, projected flows, DI references and idempotence.')
