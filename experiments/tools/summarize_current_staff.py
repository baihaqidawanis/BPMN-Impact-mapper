"""Summarize selected-primary diagnostics without mixing full/partial metrics."""
import argparse
import hashlib
import json
from pathlib import Path


def summarize(folder):
    report=json.loads((folder/'benchmark-report.json').read_text(encoding='utf-8'))
    rows=report['results']
    assert len(rows)==20 and all(r['reproducible'] and r['repetitions']==2 for r in rows)
    groups={}
    for row in rows:
        run=folder/row['scenario_id']/row['method']
        prediction=(run/'impact-report.json').read_bytes()
        evaluation=json.loads((run/'evaluation-report.json').read_text())
        assert hashlib.sha256(prediction).hexdigest()==row['prediction_sha256']==evaluation['prediction_sha256']
        assert evaluation['evaluation_id']==row['evaluation_id']
        groups.setdefault(row['scenario_id'],set()).add(row['input_fingerprint'])
    assert all(len(values)==1 for values in groups.values())
    fields=['precision_at_5','recall_at_5','recall_at_10','map','mrr','r_precision']
    summary=[]
    lines=['# Staff normalized: selected-primary diagnostics','',
           'Four exposed development cases, one labelled anchor each. Single-source provisional references; other changes unlabelled. Not full-scenario accuracy or held-out validation.', '',
           '| Case | Method | P@5 | R@5 | R@10 | AP | RR | R-precision |',
           '|---|---|---:|---:|---:|---:|---:|---:|']
    for row in rows:
        metric=row['metrics'].get('positive_ranking')
        if row.get('truth_scope')!='subset':continue
        assert metric and metric['positive_changes']==1
        item={'case':row['scenario_id'],'method':row['method'],**{f:metric[f] for f in fields}}
        summary.append(item)
        lines.append('| '+item['case']+' | '+item['method']+' | '+' | '.join(f'{item[f]:.4f}' for f in fields)+' |')
    assert len(summary)==16
    lines+=['','Selected-anchor macro means (four related cases; these are diagnostic means, not complete-case benchmark aggregates):','',
            '| Method | P@5 | R@5 | R@10 | MAP | MRR | R-precision |','|---|---:|---:|---:|---:|---:|---:|']
    means=[]
    for method in ['bm25','semantic','fusion','hybrid']:
        values=[r for r in summary if r['method']==method];assert len(values)==4
        mean={'method':method,**{f:sum(r[f] for r in values)/4 for f in fields}};means.append(mean)
        lines.append('| '+method+' | '+' | '.join(f'{mean[f]:.4f}' for f in fields)+' |')
    controls=[r for r in rows if r.get('truth_scope')!='subset']
    for r in controls:
        p=json.loads((folder/r['scenario_id']/r['method']/'impact-report.json').read_text());assert p['changes']==[]
    lines+=['',f'All {len(controls)} unchanged method controls have zero changes. All 20 cells reproduced byte-identical predictions across two runs. No ranking settings, reference files or labels changed after inference.','']
    (folder/'selected-primary-summary.json').write_text(json.dumps({'rows':summary,'diagnostic_means':means},indent=2)+'\n')
    (folder/'selected-primary-summary.md').write_text('\n'.join(lines),encoding='utf-8')
    print('\n'.join(lines[-11:]))


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('output',type=Path);summarize(p.parse_args().output)
