"""Validate complete study artifacts and aggregate positive-case metrics."""
import argparse
import hashlib
import json
from pathlib import Path

def summarize(root):
    rows=json.loads((root/'study-summary.json').read_text(encoding='utf-8'))
    manifest=json.loads((root/'manifest.json').read_text(encoding='utf-8'))
    variants=['full','no-context','no-start-end','both-top10']
    expected={(case['id'],variant) for case in manifest['scenarios'] for variant in variants}
    if {(r['scenario'],r['variant']) for r in rows}!=expected or len(rows)!=len(expected):
        raise ValueError('incomplete/duplicate study rows; not a complete experiment')
    summaries=[]
    for variant in variants:
        group=[r for r in rows if r['variant']==variant]
        positives=[];selected_hits=selected_count=0;controls=correct=0
        for row in group:
            if not row['reproducible']:raise ValueError('non-reproducible row')
            first=None
            for repetition in range(1,manifest['repetitions']+1):
                prefix=root/row['scenario']/variant/f'run{repetition}'
                prediction=(prefix/'impact-report.json').read_bytes()
                if hashlib.sha256(prediction).hexdigest()!=row['prediction_sha256']:raise ValueError('prediction hash mismatch')
                report=json.loads(prediction)
                if report['input_fingerprint']!=row['input_fingerprint']:raise ValueError('fingerprint mismatch')
                evaluation=json.loads((prefix/'evaluation.json').read_text(encoding='utf-8'))
                if evaluation['prediction_sha256']!=row['prediction_sha256']:raise ValueError('evaluation binding mismatch')
                if first is None:first=evaluation
                elif evaluation['evaluation_id']!=first['evaluation_id']:raise ValueError('evaluation differs between repetitions')
            ranking=row['metrics'].get('positive_ranking')
            if ranking and ranking['positive_changes']:positives.append(ranking)
            for case in first['changes']:
                truth=set(case['expected_files']);predicted=case['predicted_files']
                if truth:
                    selected_hits+=len(truth.intersection(predicted));selected_count+=len(predicted)
                else:controls+=1;correct+=not predicted
        count=sum(r['positive_changes'] for r in positives)
        summary={'variant':variant,'positive_changes':count,'negative_cases':controls,'correct_abstentions':correct,
                 'selected_candidate_precision':selected_hits/selected_count if selected_count else None,
                 'selected_candidates_positive_cases':selected_count}
        for metric in ['map','mrr','r_precision','precision_at_5','recall_at_5','recall_at_10']:
            summary[metric]=sum(r[metric]*r['positive_changes'] for r in positives)/count if count else None
        summaries.append(summary)
    # Cross-variant input consistency is required independently of each model score.
    for case in manifest['scenarios']:
        if len({r['input_fingerprint'] for r in rows if r['scenario']==case['id']})!=1:raise ValueError('variant input mismatch')
    (root/'comparison.json').write_text(json.dumps(summaries,indent=2)+'\n',encoding='utf-8')
    lines=['# Context study results','', 'Four exposed positive cases; provisional source references. Fixed-k metrics use denominator 5/10 even if a filter returns fewer candidates. Selected precision uses actual returned candidates for positive cases only.','', '| Variant | P@5 | R@5 | R@10 | MAP | MRR | R-Precision | Selected precision | Candidates |', '|---|---:|---:|---:|---:|---:|---:|---:|---:|']
    for r in summaries:
        keys=['precision_at_5','recall_at_5','recall_at_10','map','mrr','r_precision','selected_candidate_precision']
        lines.append('| '+r['variant']+' | '+' | '.join(f'{r[k]:.4f}' if r[k] is not None else 'n/a' for k in keys)+f" | {r['selected_candidates_positive_cases']} |")
    (root/'comparison.md').write_text('\n'.join(lines)+'\n',encoding='utf-8')
    print('\n'.join(lines))

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('output',type=Path);args=parser.parse_args();summarize(args.output)
