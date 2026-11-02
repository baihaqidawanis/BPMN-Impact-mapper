"""Offline, file-level diagnostic. Never call this from inference."""
import argparse
import json
from pathlib import Path


def unique(files):
    return list(dict.fromkeys(files))


def metrics(files, expected):
    files = set(files)
    tp = len(files & expected)
    precision = tp / len(files) if files else 0
    recall = tp / len(expected) if expected else 0
    return dict(returned=len(files), true_positive=tp,
                false_positive=len(files - expected), false_negative=len(expected - files),
                precision=precision, recall=recall,
                f1=2 * precision * recall / (precision + recall) if precision + recall else 0)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ('report', 'session', 'reference', 'output', 'element-id'):
        parser.add_argument('--' + key, required=True)
    parser.add_argument('--change-type', default='ADD_TASK')
    args = parser.parse_args()
    read = lambda path: json.loads(Path(path).read_text(encoding='utf-8-sig'))
    report, envelope, reference = map(read, (args.report, args.session, args.reference))
    session = envelope['session']
    if session['bundle']['analysis_id'] != report['analysis_id']:
        raise ValueError('Session and report identities differ')
    matches = lambda c: c['change_type'] == args.change_type and c['element_id'] == args.element_id
    truth = next(c for c in reference['cases'] if matches(c))
    expected = {x['file'] for x in truth['expected']}
    change = next(c for c in report['changes'] if c['change_type'] == args.change_type
                  and (c.get('new_element') or c.get('old_element'))['id'] == args.element_id)
    ranked = unique(x['file'] for x in change['impacts'])
    context = next(c for c in session['bundle']['changes'] if matches(c))
    seeds = unique(c['file'] for c in context['candidates'] if c.get('origin') != 'agent-source-tool')
    accepted = unique(d['file'] for d in session['decisions'] if matches(d) and d['verdict'] == 'RELEVANT')
    result = dict(analysis_id=report['analysis_id'], session_id=session['id'],
                  session_status=session['status'], change_type=args.change_type, element_id=args.element_id,
                  reference_status='provisional source-derived diagnostic, not independent implementation truth',
                  hybrid_returned=metrics(ranked, expected), hybrid_seed_set=metrics(seeds, expected),
                  qwen_relevant_set=metrics(accepted, expected),
                  added_by_qwen=sorted(set(accepted) - set(seeds)),
                  removed_by_qwen=sorted(set(seeds) - set(accepted)),
                  repository_search_calls=sum(o['request']['name'] == 'search_source' for o in session['observations']),
                  interpretation='Accepted-set precision and Precision@k are different. Compare equal budgets; file matches do not prove plan correctness.')
    for k in (3, 5, 10):
        result[f'hybrid_at_{k}'] = dict(precision=sum(f in expected for f in ranked[:k]) / k,
                                         recall=sum(f in expected for f in ranked[:k]) / len(expected))
    target = Path(args.output)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(result, indent=2))


if __name__ == '__main__':
    main()
