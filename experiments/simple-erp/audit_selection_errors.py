"""Audit saved retrieval/selection outputs. Labels never feed back into retrieval."""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path


def assess(expected, raw, selected, deferred):
    expected, raw, selected, deferred = map(set, (expected, raw, selected, deferred))
    if selected & deferred or selected | deferred != raw:
        raise ValueError('selection/deferred must partition raw files')
    return {
        'retained_expected': sorted(expected & selected),
        'missing_from_raw': sorted(expected - raw),
        'lost_by_selection': sorted(expected & raw - selected),
        'selected_not_in_reference': sorted(selected - expected),
        'deferred_not_in_reference': sorted(deferred - expected),
    }


def audit(study, selection, output):
    bindings = []

    def read(path):
        data = path.read_bytes()
        bindings.append({'file': path.as_posix(), 'sha256': hashlib.sha256(data).hexdigest()})
        return json.loads(data)

    # Reject changed bound inputs before analysing the selection run.
    for binding in read(selection / 'validation-bindings.json'):
        if hashlib.sha256(Path(binding['file']).read_bytes()).hexdigest() != binding['sha256']:
            raise ValueError('selection input drift: ' + binding['file'])
    manifest = read(study / 'manifest.json')
    scenarios = {s['id']: s for s in manifest['scenarios']}
    policies = {v['variant']: v['policy'] for v in read(selection / 'policy.json')['variants']}
    results = []
    seen_variants = set()
    for row in read(selection / 'validation.json'):
        if row['variant'] in seen_variants or policies.get(row['variant']) != row['policy']:
            raise ValueError('duplicate variant or frozen policy mismatch')
        seen_variants.add(row['variant'])
        reports, truths = {}, {}
        for scenario, config in scenarios.items():
            report = read(study / scenario / row['variant'] / 'run1' / 'impact-report.json')
            nodes = {}
            for c in report['changes']:
                if c.get('part_of') or c['change_type'] == 'MODIFY_SEQUENCE_FLOW':
                    continue
                id = (c.get('new_element') or c['old_element'])['id']
                if id in nodes:
                    raise ValueError('ambiguous same-ID changes; audit needs typed selection records')
                nodes[id] = c
            reports[scenario] = nodes
            truth = read(Path(config['ground_truth_path']))
            truths[scenario] = {(c['change_type'], c['element_id']): {e['file'] for e in c['expected']}
                                for c in truth['cases']}
        seen = set()
        groups = Counter()
        nodes = []
        for record in row['nodes']:
            key = (record['scenario'], record['element_id'])
            if key in seen:
                raise ValueError('duplicate selection node')
            seen.add(key)
            change = reports[key[0]][key[1]]
            raw = {}
            for candidate in change['impacts']:
                raw.setdefault(candidate['file'], candidate)
            expected = truths[key[0]][(change['change_type'], key[1])]
            findings = assess(expected, raw, record['selected'], record['deferred'])
            samples = []
            for file in findings['selected_not_in_reference']:
                c = raw[file]
                group = c.get('layer') or 'unknown'
                groups[group] += 1
                samples.append({
                    'file': file, 'module': c.get('module'), 'layer': group,
                    'retrieval_rank': list(raw).index(file) + 1,
                    'evidence': [{'type': e['type'], 'location': e.get('location', ''),
                                  'excerpt': e.get('excerpt', '')[:240]}
                                 for e in c.get('evidence', [])[:8]],
                })
            samples.sort(key=lambda candidate: candidate['retrieval_rank'])
            nodes.append({'scenario': key[0], 'element_id': key[1], 'change_type': change['change_type'],
                          **findings, 'extra_candidate_evidence': samples})
        if seen != {(scenario, id) for scenario, nodes_by_id in reports.items() for id in nodes_by_id}:
            raise ValueError('selection does not cover all primary nodes')
        results.append({'variant': row['variant'], 'policy': row['policy'],
                        'extra_by_layer': dict(sorted(groups.items())), 'nodes': nodes})
    if seen_variants != set(policies):
        raise ValueError('validation does not cover all frozen policies')
    output.mkdir(parents=True, exist_ok=False)
    result = {'status': 'development diagnostic against provisional references, not independent errors',
              'input_bindings': bindings, 'variants': results}
    (output / 'audit.json').write_text(json.dumps(result, indent=2) + '\n', encoding='utf-8')
    lines = ['# Selection error audit', '',
             'Not-in-reference files are provisional false positives, not proven irrelevant files.', '',
             '| Variant | Scenario | Node | Retained hits | Missing in raw | Lost by filter | Selected extras |',
             '|---|---|---|---:|---:|---:|---:|']
    for row in results:
        for n in row['nodes']:
            lines.append('| ' + ' | '.join([row['variant'], n['scenario'], n['element_id'],
                *[str(len(n[k])) for k in ['retained_expected', 'missing_from_raw', 'lost_by_selection', 'selected_not_in_reference']]]) + ' |')
    for row in results:
        lines += ['', '## ' + row['variant'], '', 'Selected extra pairs by indexed layer: ' + json.dumps(row['extra_by_layer']), '']
        for n in row['nodes']:
            lines += ['### ' + n['scenario'] + ' / ' + n['element_id'], '']
            for c in n['extra_candidate_evidence']:
                rank = next((e['excerpt'] for e in c['evidence'] if e['type'] == 'RRF_FUSION'), 'branch ranks unavailable')
                lines.append(f"- rank {c['retrieval_rank']}: `{c['file']}`; {rank}")
            lines.append('')
    (output / 'audit.md').write_text('\n'.join(lines), encoding='utf-8')
    print('Hash-bound audit written; selected extras and selection-caused misses kept separate.')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('--study', type=Path, required=True)
    p.add_argument('--selection', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    a = p.parse_args()
    audit(a.study, a.selection, a.output)
