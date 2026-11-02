"""Publish aggregate research metrics only; preserve historical four-method snapshot."""
import hashlib
import json
from pathlib import Path


def publish():
    rows = []
    bindings = []
    runs = 0
    baselines = {}
    for folder, variant in [('condition-focus-challenges', 'condition-focus'),
                            ('generic-window-challenges', 'generic-windows'),
                            ('code-window-condition', 'code-windows')]:
        root = Path('outputs') / folder
        summary_bytes = (root / 'study-summary.json').read_bytes()
        bindings.append({'file': (root / 'study-summary.json').as_posix(),
                         'sha256': hashlib.sha256(summary_bytes).hexdigest()})
        manifest = json.loads((root / 'manifest.json').read_text())
        cases = {s['id']: s for s in manifest['scenarios']}
        summary = json.loads(summary_bytes)
        assert len(summary) == 2 * len(cases) and manifest['repetitions'] == 2
        seen = set()
        for row in summary:
            key = (row['scenario'], row['variant'])
            assert key not in seen and row['variant'] in ('full', variant) and row['reproducible']
            seen.add(key)
            case = cases[row['scenario']]
            for file, key in [('old_bpmn_path', 'old_bpmn_sha256'), ('new_bpmn_path', 'new_bpmn_sha256'),
                              ('ground_truth_path', 'ground_truth_sha256')]:
                assert hashlib.sha256(Path(case[file]).read_bytes()).hexdigest() == case[key]
            first = root / row['scenario'] / row['variant'] / 'run1'
            second = first.parent / 'run2'
            for file in ('impact-report.json', 'evaluation.json'):
                assert (first / file).read_bytes() == (second / file).read_bytes()
            assert hashlib.sha256((first / 'impact-report.json').read_bytes()).hexdigest() == row['prediction_sha256']
            evaluation = json.loads((first / 'evaluation.json').read_text())
            assert evaluation['prediction_sha256'] == row['prediction_sha256']
            assert evaluation['metrics'] == row['metrics']
            runs += 2
            positive = row['metrics'].get('positive_ranking')
            if positive is None:
                assert not json.loads((first / 'impact-report.json').read_text())['changes']
                continue
            if row['variant'] == 'full':
                if row['scenario'] in baselines:
                    assert baselines[row['scenario']] == row['prediction_sha256']
                    continue
                baselines[row['scenario']] = row['prediction_sha256']
            rows.append({'scenario': row['scenario'], 'variant': row['variant'],
                         'precision_at_5': positive['precision_at_5'],
                         'recall_at_5': positive['recall_at_5'], 'recall_at_10': positive['recall_at_10'],
                         'map': positive['map'], 'mrr': positive['mrr'],
                         'r_precision': positive['r_precision']})
    result = {'version': 1, 'date': '2026-10-30', 'repository': 'Simple-ERP',
              'method': 'hybrid', 'reference_status': 'provisional-source-review',
              'analysis_runs': runs, 'top_k': 10, 'independent_validation': False,
              'adopted_variants': [], 'source_bindings': bindings, 'results': rows}
    output = Path('web/public/benchmarks/retrieval-studies-2026-10-30.json')
    output.write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    print(f'Published {len(rows)} per-case rows from {runs} runs; no variant adopted.')


if __name__ == '__main__':
    publish()
