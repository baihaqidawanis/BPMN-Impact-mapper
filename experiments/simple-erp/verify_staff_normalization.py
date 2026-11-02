"""Check preservation properties against backed-up Staff diagrams, not retrieval."""
import argparse
import json
from pathlib import Path
from xml.dom import minidom
from normalize_staff_bpmn import B, BD, attr, bounds, children, digest


def check(before, after):
    old,new=map(minidom.parseString,(before,after))
    op,np=[d.getElementsByTagNameNS(B,'process')[0] for d in (old,new)]
    def nodes(p):return {attr(e,'id'):e for e in children(p) if e.localName not in ('sequenceFlow','laneSet','documentation')}
    on,nn=nodes(op),nodes(np)
    ends={id for id,e in on.items() if e.localName=='endEvent'}
    assert sum(e.localName=='startEvent' for e in nn.values())==1
    assert sum(e.localName=='endEvent' for e in nn.values())==1
    for id,e in on.items():
        if id in ends:continue
        assert id in nn and e.localName==nn[id].localName,(id,'node lost/type changed')
        assert dict(e.attributes.items())==dict(nn[id].attributes.items()),(id,'business attributes changed')
    of={attr(e,'id'):e for e in children(op,'sequenceFlow')};nf={attr(e,'id'):e for e in children(np,'sequenceFlow')}
    for id,f in of.items():
        assert id in nf
        assert attr(f,'sourceRef')==attr(nf[id],'sourceRef')
        assert [c.toxml() for c in children(f,'conditionExpression')]==[c.toxml() for c in children(nf[id],'conditionExpression')],(id,'condition changed')
        if attr(f,'targetRef') not in ends:
            assert dict(f.attributes.items())==dict(nf[id].attributes.items()),(id,'nonterminal flow changed')
        else:
            assert attr(f,'name') in attr(nf[id],'name')
            # A rewritten exit must not re-enter any original business task.
            pending=[attr(nf[id],'targetRef')];seen=set();reached=False
            while pending:
                target=pending.pop()
                if target in seen:continue
                seen.add(target)
                assert target not in on or target in ends,(id,'exit now enters original business logic')
                if nn[target].localName=='endEvent':reached=True
                pending.extend(attr(e,'targetRef') for e in nf.values() if attr(e,'sourceRef')==target)
            assert reached
    def membership(p):
        return {''.join(c.data for c in ref.childNodes if c.nodeType==c.TEXT_NODE):attr(lane,'id')
                for lane in p.getElementsByTagNameNS(B,'lane') for ref in children(lane,'flowNodeRef')}
    om,nm=membership(op),membership(np)
    for id in on.keys()-ends:assert om[id]==nm[id],(id,'actor changed')
    for key,value in old.documentElement.attributes.items():
        if key.startswith('xmlns:'):assert new.documentElement.getAttribute(key)==value,('QName namespace lost',key)
    osh={attr(e,'bpmnElement'):e for e in old.getElementsByTagNameNS(BD,'BPMNShape')}
    nsh={attr(e,'bpmnElement'):e for e in new.getElementsByTagNameNS(BD,'BPMNShape')}
    for id in on.keys()-ends:assert bounds(osh[id])==bounds(nsh[id]),(id,'existing layout changed')
    for id in nn:assert id in nsh,(id,'missing shape')
    ned={attr(e,'bpmnElement'):e for e in new.getElementsByTagNameNS(BD,'BPMNEdge')}
    for id,f in nf.items():
        assert attr(f,'sourceRef') in nn and attr(f,'targetRef') in nn
        assert id in ned and len(children(ned[id],'waypoint'))>=2
    return True


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--review',type=Path,required=True);parser.add_argument('--applied-root',type=Path)
    args=parser.parse_args();records=json.loads((args.review/'manifest.json').read_text(encoding='utf-8'))
    for r in records:
        old=(args.review/'backup'/r['file']).read_bytes();new=((args.applied_root or args.review/'prepared')/r['file']).read_bytes()
        assert digest(old)==r['before_sha256'] and digest(new)==r['after_sha256']
        check(old,new)
    print(f'{len(records)} diagrams: preserved IDs/types/attributes, conditions, actor membership, QName bindings, original-node layout, exit isolation and complete DI.')
