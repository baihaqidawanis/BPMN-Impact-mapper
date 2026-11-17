"""Route active experimental diagrams around shapes; preserve semantic XML."""
import heapq
import json
import shutil
import xml.etree.ElementTree as ET
from pathlib import Path
from build_structural_scenarios import q, tag, sha


def route(root):
    shapes={s.get('bpmnElement'):s for s in root.iter(q('bpmndi','BPMNShape'))}
    boxes={k:tuple(float(s.find(q('dc','Bounds')).get(a)) for a in ('x','y','width','height')) for k,s in shapes.items()}
    flows=list(root.iter(tag('sequenceFlow')))
    nodes={e.get('id'):e for e in root.iter() if e.get('id') in boxes and e.tag not in (tag('lane'),tag('participant')) and not e.tag.startswith('{http://www.omg.org/spec/BPMN/20100524/DI}')}
    obstacles=[boxes[k] for k in nodes]
    used=set()
    edges={e.get('bpmnElement'):e for e in root.iter(q('bpmndi','BPMNEdge'))}
    xs=set();ys=set()
    for x,y,w,h in obstacles:
        xs.update((x-24,x+w+24));ys.update((y-24,y+h+24))
    # Boundary points for branches differ from the through-flow.
    def port(id,side):
        x,y,w,h=boxes[id]
        return {'right':(x+w,y+h/2),'left':(x,y+h/2),'top':(x+w/2,y)}[side]
    def stub(p,side):return (p[0]+24,p[1]) if side=='right' else (p[0]-24,p[1]) if side=='left' else (p[0],p[1]-24)
    specs=[]
    for f in flows:
        a,b=f.get('sourceRef'),f.get('targetRef')
        outgoing=[v for v in flows if v.get('sourceRef')==a]
        incoming=[v for v in flows if v.get('targetRef')==b]
        sa='top' if nodes[a].tag.endswith('Gateway') and len(outgoing)>1 and outgoing.index(f)>0 else 'right'
        sb='top' if nodes[b].tag.endswith('Gateway') and len(incoming)>1 and incoming.index(f)>0 else 'left'
        start,end=port(a,sa),port(b,sb);ss,tt=stub(start,sa),stub(end,sb)
        xs.update((ss[0],tt[0]));ys.update((ss[1],tt[1]));specs.append((f,start,end,ss,tt))
    xx=sorted(xs);yy=sorted(ys)
    def clear(a,b):
        for x,y,w,h in obstacles:
            if a[0]==b[0] and x-6<a[0]<x+w+6 and max(min(a[1],b[1]),y-6)<min(max(a[1],b[1]),y+h+6):return False
            if a[1]==b[1] and y-6<a[1]<y+h+6 and max(min(a[0],b[0]),x-6)<min(max(a[0],b[0]),x+w+6):return False
        return True
    for f,start,end,ss,tt in specs:
        origin=(xx.index(ss[0]),yy.index(ss[1]),0)
        queue=[(0,origin)];dist={origin:0};previous={};finish=None
        while queue:
            cost,state=heapq.heappop(queue)
            if cost!=dist[state]:continue
            i,j,d=state;a=(xx[i],yy[j])
            if a==tt:finish=state;break
            for ni,nj,nd in ((i-1,j,1),(i+1,j,1),(i,j-1,2),(i,j+1,2)):
                if not(0<=ni<len(xx) and 0<=nj<len(yy)):continue
                b=(xx[ni],yy[nj])
                if not clear(a,b):continue
                segment=tuple(sorted((a,b)))
                value=cost+abs(a[0]-b[0])+abs(a[1]-b[1])+(35 if d and d!=nd else 0)+(120 if segment in used else 0)
                nxt=(ni,nj,nd)
                if value<dist.get(nxt,float('inf')):dist[nxt]=value;previous[nxt]=state;heapq.heappush(queue,(value,nxt))
        if finish is None:raise ValueError('No obstacle-free route for '+f.get('id'))
        points=[];state=finish
        while True:
            points.append((xx[state[0]],yy[state[1]]))
            if state==origin:break
            state=previous[state]
        points.reverse()
        for a,b in zip(points,points[1:]):used.add(tuple(sorted((a,b))))
        points=[start]+points+[end]
        simplified=[]
        for p in points:
            if simplified and simplified[-1]==p:continue
            while len(simplified)>1 and ((simplified[-2][0]==simplified[-1][0]==p[0]) or (simplified[-2][1]==simplified[-1][1]==p[1])):simplified.pop()
            simplified.append(p)
        edge=edges[f.get('id')]
        for child in list(edge):
            if child.tag==q('di','waypoint'):edge.remove(child)
        for i,(x,y) in enumerate(simplified):edge.insert(i,ET.Element(q('di','waypoint'),{'x':str(x),'y':str(y)}))
    # Place labels only after every edge is routed, so a later edge cannot
    # silently cross a label positioned earlier in the pass.
    placed=[]
    # bpmn.io creates external labels for named gateways/events even when
    # the XML has no explicit BPMNLabel bounds. Reserve that visible area.
    for id,n in nodes.items():
        if not n.get('name') or not (n.tag.endswith('Gateway') or n.tag.endswith('Event')):continue
        x,y,w,h=boxes[id]
        placed.append((x+w/2-55,y+h+3,110,48))
    for id,box in boxes.items():
        if id in nodes:continue
        x,y,w,h=box
        placed.extend([(x,y-7,w,14),(x,y+h-7,w,14)])
    segments={id:[(float(p.get('x')),float(p.get('y'))) for p in e if p.tag==q('di','waypoint')] for id,e in edges.items()}
    def overlap(a,b):
        x,y,w,h=a;xx,yy,ww,hh=b
        return x<xx+ww+5 and xx<x+w+5 and y<yy+hh+5 and yy<y+h+5
    def crosses(a,b,box):
        x,y,w,h=box
        return (a[0]==b[0] and x<a[0]<x+w and max(min(a[1],b[1]),y)<min(max(a[1],b[1]),y+h)) or (a[1]==b[1] and y<a[1]<y+h and max(min(a[0],b[0]),x)<min(max(a[0],b[0]),x+w))
    for id,edge in edges.items():
        label=edge.find(q('bpmndi','BPMNLabel'))
        if label is None:continue
        bounds=label.find(q('dc','Bounds'));w,h=120.,32.
        bounds.set('width',str(w));bounds.set('height',str(h))
        choices=[]
        for a,b in zip(segments[id],segments[id][1:]):
            for fraction in (.5,.25,.75):
                cx=a[0]+(b[0]-a[0])*fraction;cy=a[1]+(b[1]-a[1])*fraction
                for distance in (12,40,70,105):
                    for side in (-1,1):
                        box=(cx-w/2,cy-distance-h if side<0 else cy+distance,w,h) if a[1]==b[1] else (cx-distance-w if side<0 else cx+distance,cy-h/2,w,h)
                        blocked=sum(overlap(box,v) for v in obstacles+placed)
                        intersections=sum(crosses(c,d,box) for points in segments.values() for c,d in zip(points,points[1:]))
                        choices.append((blocked*100000+intersections*1000+distance,box))
        _,box=min(choices,key=lambda v:v[0]);placed.append(box)
        bounds.set('x',str(box[0]));bounds.set('y',str(box[1]))


def semantic(data):
    root=ET.fromstring(data)
    for child in list(root):
        if child.tag==q('bpmndi','BPMNDiagram'):root.remove(child)
    return ET.tostring(root)

if __name__=='__main__':
    pack=Path('experiments/simple-erp/current-staff-v2');backup=Path('outputs/bpmn-routing-before-2026-11-03')
    backup.mkdir(parents=True,exist_ok=True)
    manifest=json.loads((pack/'manifest.json').read_text(encoding='utf-8'))
    for case in manifest['scenarios']:
        path=Path(case['new_bpmn_path'])
        if case['id'].endswith('unchanged-control'):continue
        data=path.read_bytes();dest=backup/path.parent.name;dest.mkdir(exist_ok=True)
        if not (dest/path.name).exists():shutil.copyfile(path,dest/path.name)
        root=ET.fromstring(data);route(root);ET.indent(root);changed=ET.tostring(root,encoding='utf-8',xml_declaration=True)
        assert semantic(data)==semantic(changed),'Semantic process was changed'
        path.write_bytes(changed);case['new_bpmn_sha256']=sha(changed)
    (pack/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n',encoding='utf-8',newline='\n')
    snapshot=json.loads((pack/'snapshot.json').read_text(encoding='utf-8'))
    for item in snapshot['pairs']:
        case=next(c for c in manifest['scenarios'] if c['id']==item['id']);item['changed_sha256']=case['new_bpmn_sha256']
    (pack/'snapshot.json').write_text(json.dumps(snapshot,indent=2)+'\n',encoding='utf-8',newline='\n')
    print('Routed four active after diagrams; BPMN semantics unchanged.')
