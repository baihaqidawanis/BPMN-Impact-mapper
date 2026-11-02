"""Validate frozen context study and summarize per-case changes, never tune labels."""
import argparse
import hashlib
import json
from pathlib import Path
import xml.etree.ElementTree as ET

NS={'b':'http://www.omg.org/spec/BPMN/20100524/MODEL'}


def summarize(output, candidate_variant='multi-hop4'):
    manifest=json.loads((output/'manifest.json').read_text())
    summary=json.loads((output/'study-summary.json').read_text())
    cases={s['id']:s for s in manifest['scenarios']}
    assert len(summary)==len(cases)*2 and {r['variant'] for r in summary}=={'full',candidate_variant}
    indexed={};anchors_seen={}
    for row in summary:
        case=cases[row['scenario']];assert row['reproducible'] and row['scope'].startswith('primary-nodes')
        for path,key in [('old_bpmn_path','old_bpmn_sha256'),('new_bpmn_path','new_bpmn_sha256'),('ground_truth_path','ground_truth_sha256')]:
            assert hashlib.sha256(Path(case[path]).read_bytes()).hexdigest()==case[key]
        old=ET.parse(case['old_bpmn_path']).getroot();new=ET.parse(case['new_bpmn_path']).getroot()
        old_nodes={n.get('id'):n for p in old.findall('b:process',NS) for n in p if n.get('id')}
        flows={e.get('id'):(e.get('sourceRef'),e.get('targetRef')) for e in new.findall('.//b:sequenceFlow',NS)}
        predictions=[];evaluations=[];anchor_count=0
        for repeat in [1,2]:
            folder=output/row['scenario']/row['variant']/f'run{repeat}'
            data=(folder/'impact-report.json').read_bytes();predictions.append(data)
            evaluated=(folder/'evaluation.json').read_bytes();evaluations.append(evaluated)
            assert hashlib.sha256(data).hexdigest()==row['prediction_sha256']
            assert json.loads(evaluated)['prediction_sha256']==row['prediction_sha256']
            report=json.loads(data)
            for c in report['changes']:
                counts={}
                for anchor in (c.get('context') or {}).get('baseline_anchors',[]):
                    anchor_count+=1;assert row['variant']==candidate_variant=='multi-hop4'
                    assert anchor['id'] in old_nodes and anchor['name']==old_nodes[anchor['id']].get('name')
                    assert 1<=anchor['hops']<=4 and anchor['hops']==len(anchor['flow_path'])
                    current=c['new_element']['id'];direction=anchor['direction'];counts[direction]=counts.get(direction,0)+1;assert counts[direction]<=2
                    for fid in anchor['flow_path']:
                        a,b=flows[fid]
                        if direction=='predecessor':assert current==b;current=a
                        else:assert current==a;current=b
                    assert current==anchor['id']
        assert predictions[0]==predictions[1] and evaluations[0]==evaluations[1]
        indexed[(row['scenario'],row['variant'])]=row;anchors_seen[(row['scenario'],row['variant'])]=anchor_count
    lines=['# Bounded baseline-context study','',
           'Exposed development cases; frozen provisional source-derived node references. Sequence-flow changes excluded. Two runs per case/variant; no production adoption inferred.','',
           '| Case | Variant | P@5 | R@5 | MAP | Union5 files | Union5 precision | Union5 recall |',
           '|---|---|---:|---:|---:|---:|---:|---:|']
    for case in cases:
        left,right=indexed[(case,'full')],indexed[(case,candidate_variant)]
        assert left['input_fingerprint']==right['input_fingerprint'] and left['analysis_id']!=right['analysis_id']
        before=json.loads((output/case/'full/run1/impact-report.json').read_text())
        after=json.loads((output/case/candidate_variant/'run1/impact-report.json').read_text())
        if candidate_variant in ('generic-windows','code-windows'):
            assert [(f['file'],f['language']) for f in before['source_analysis']]==[(f['file'],f['language']) for f in after['source_analysis']]
            for a,b in zip(before['source_analysis'],after['source_analysis']):
                if a['adapter']=='go-syntax':assert a==b
                elif candidate_variant=='code-windows' and a['language'] not in ('JavaScript','TypeScript','Java','Python','C#'):assert a==b
                elif not a.get('fallback'):assert b['adapter']==('generic-code-windows-v1-min1200-max8' if candidate_variant=='code-windows' else 'generic-text-windows-v1-min1200-max8')
        else:assert before['source_analysis']==after['source_analysis']
        assert [(c['change_type'],c['old_element'],c['new_element'],c.get('part_of')) for c in before['changes']]==[(c['change_type'],c['old_element'],c['new_element'],c.get('part_of')) for c in after['changes']]
        assert 'baseline_context_policy' not in before['method_metadata']
        if candidate_variant=='multi-hop4':
            assert after['method_metadata']['baseline_context_policy'].endswith('hops4-cap128-anchors2')
        elif candidate_variant=='condition-focus':
            assert candidate_variant=='condition-focus' and after['method_metadata']['condition_query_policy']=='per-flow-token-delta-gateway-label-v1-full-fallback'
        else:assert candidate_variant in ('generic-windows','code-windows') and after['method_metadata']['context_study']==candidate_variant
        for row in [left,right]:
            p=row['metrics'].get('positive_ranking');u=row.get('union_top5_per_node')
            if p is None:
                assert not json.loads((output/case/row['variant']/'run1/impact-report.json').read_text())['changes']
                lines.append(f'| {case} | {row["variant"]} | control | control | control | 0 | n/a | n/a |')
            else:lines.append(f'| {case} | {row["variant"]} | {p["precision_at_5"]:.4f} | {p["recall_at_5"]:.4f} | {p["map"]:.4f} | {len(u["predicted"])} | {u["precision"]:.4f} | {u["recall"]:.4f} |')
        # Baseline-anchor expansion applies only to added-node clusters. Removal
        # and condition-only transfers intentionally use unchanged query context.
        has_added_node=any(c['change_type'] in ('ADD_TASK','ADD_GATEWAY') for c in after['changes'])
        if not has_added_node or case.endswith('add-task'):
            assert anchors_seen[(case,candidate_variant)]==0
    lines+=['','## Expected-file positions per node','','File ranks use distinct files. Missing means absent from the returned list.','',
            f'| Case | Node | Expected file | Full rank | {candidate_variant} rank |','|---|---|---|---:|---:|']
    for case,config in cases.items():
        truth=json.loads(Path(config['ground_truth_path']).read_text())
        predictions={v:json.loads((output/case/v/'run1/impact-report.json').read_text()) for v in ['full',candidate_variant]}
        for c in truth['cases']:
            for expected in c['expected']:
                ranks=[]
                for variant in ['full',candidate_variant]:
                    change=next(d for d in predictions[variant]['changes'] if d['change_type']==c['change_type'] and (d.get('new_element') or d['old_element'])['id']==c['element_id'])
                    files=list(dict.fromkeys(i['file'] for i in change['impacts']))
                    ranks.append(str(files.index(expected['file'])+1) if expected['file'] in files else 'missing')
                lines.append(f'| {case} | {c["element_id"]} | {expected["file"]} | {ranks[0]} | {ranks[1]} |')
    lines+=['',f'All {len(summary)} case/variant cells / {len(summary)*2} analyses reproduce predictions and evaluations byte-for-byte. Fingerprints match across variants; variant analysis identities differ. Recorded anchor labels and flow paths match the BPMN files. No reference/ranking edits performed after inference.','']
    (output/'comparison.md').write_text('\n'.join(lines),encoding='utf-8')
    print('\n'.join(lines))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('output',type=Path);p.add_argument('--candidate-variant',default='multi-hop4',choices=['multi-hop4','condition-focus','generic-windows','code-windows']);a=p.parse_args();summarize(a.output,a.candidate_variant)
