"""Evaluate saved agent file proposals offline; never reads source or calls a model.

Plan order is not a relevance ranking. AP/MRR/R-precision are deliberately absent
for agent selections. References are evaluation inputs, never inference inputs.
"""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath


def unique_files(items):
    files = []
    for item in items:
        name = item['file'].replace('\\', '/')
        path = PurePosixPath(name)
        if path.is_absolute() or '..' in path.parts or ':' in name:
            raise ValueError('Evaluation exports require relative repository files')
        if name not in files:
            files.append(name)
    return files


def selection_metrics(predictions, expected):
    predicted, truth = set(predictions), set(expected)
    tp = len(predicted & truth)
    precision = tp / len(predicted) if predicted else 0
    recall = tp / len(truth) if truth else 0
    return dict(selected=len(predicted), true_positive=tp,
                extra_vs_reference=len(predicted - truth), missed=len(truth - predicted),
                precision=round(precision, 6), recall=round(recall, 6),
                f1=round(2 * precision * recall / (precision + recall), 6)
                if precision + recall else 0)


def ranking_metrics(files, expected):
    truth = set(expected)
    hits = 0
    ap = 0
    rr = 0
    for rank, file in enumerate(files, 1):
        if file in truth:
            hits += 1
            ap += hits / rank
            if not rr:
                rr = 1 / rank
    return dict(average_precision=round(ap / len(truth), 6),
                reciprocal_rank=round(rr, 6),
                r_precision=round(len(set(files[:len(truth)]) & truth) / len(truth), 6))


def evaluate(session, report, reference):
    bundle = session['bundle']
    if (bundle['analysis_id'] != report['analysis_id'] or
            bundle['input_fingerprint'] != report['input_fingerprint']):
        raise ValueError('Agent session and deterministic report use different inputs')
    cases = []
    for case in reference['cases']:
        expected = unique_files(case['expected'])
        if not expected:
            raise ValueError('This positive-reference diagnostic requires expected files')
        def matches(item):
            return (item['change_type'] == case['change_type'] and
                    item['element_id'] == case['element_id'])
        changes = [c for c in report['changes'] if
                   c['change_type'] == case['change_type'] and
                   (c.get('new_element') or c.get('old_element') or {}).get('id') == case['element_id']]
        if len(changes) != 1:
            raise ValueError('Reference must match exactly one reported change')
        ranked = unique_files(changes[0].get('impacts') or [])
        proposed = unique_files([p for p in session.get('draft') or [] if matches(p)])
        rows = []
        for k in (5, 10):
            files = ranked[:k]
            rows.append(dict(label=f'Hybrid displayed Top {k}', cutoff=k,
                             files=files, **selection_metrics(files, expected)))
        cases.append(dict(change_type=case['change_type'], element_id=case['element_id'],
                          expected_files=expected, hybrid_display=rows,
                          hybrid_ranking=ranking_metrics(ranked, expected),
                          agent=dict(files=proposed, **selection_metrics(proposed, expected),
                                     map=None, mrr=None, r_precision=None)))
    return dict(version='1', analysis_id=report['analysis_id'],
                input_fingerprint=report['input_fingerprint'], session_id=session['id'],
                session_status=session['status'], model=session['identity']['model'],
                prompt_version=session['identity']['prompt_version'],
                reference_status='provisional-source-review',
                selection_policy='unique file proposals in draft for each reference change',
                ranking_unavailable_reason='Agent proposals have no declared relevance ordering.',
                cases=cases)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('session', 'report', 'reference', 'output', 'date'):
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    paths = {name: Path(getattr(args, name)) for name in ('session', 'report', 'reference')}
    inputs = {name: json.loads(path.read_text(encoding='utf-8')) for name, path in paths.items()}
    result = evaluate(inputs['session']['session'], inputs['report'], inputs['reference'])
    result['date'] = args.date
    result['input_sha256'] = {name: hashlib.sha256(path.read_bytes()).hexdigest()
                              for name, path in paths.items()}
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')


if __name__ == '__main__':
    main()
