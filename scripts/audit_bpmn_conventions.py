"""Read-only audit of local modelling conventions (not BPMN execution validity)."""
import argparse
import hashlib
import json
from pathlib import Path
import xml.etree.ElementTree as ET

NS = '{http://www.omg.org/spec/BPMN/20100524/MODEL}'


def audit(root):
    results = []
    for path in sorted(root.rglob('*.bpmn')):
        data = path.read_bytes()
        issues = []
        try:
            doc = ET.fromstring(data)
            for scope in doc.iter():
                if scope.tag not in (NS+'process', NS+'subProcess'):
                    continue
                nodes = {n.get('id'): n for n in scope if n.get('id')}
                # Empty/black-box pools have no executable contents.
                if not any(n.tag in (NS+'startEvent', NS+'endEvent', NS+'sequenceFlow') for n in scope):
                    continue
                sid = scope.get('id', '')
                for kind in ('startEvent', 'endEvent'):
                    count = sum(n.tag == NS+kind for n in scope)
                    if count != 1:
                        issues.append({'scope': sid, 'rule': 'SINGLE_'+kind.upper(), 'count': count})
                for flow in scope.findall(NS+'sequenceFlow'):
                    source, target = nodes.get(flow.get('sourceRef')), nodes.get(flow.get('targetRef'))
                    if source is None or target is None:
                        issues.append({'scope': sid, 'rule': 'MISSING_ENDPOINT', 'flow': flow.get('id')})
                    elif source.tag.endswith('Gateway') and (target.tag.endswith('Gateway') or target.tag == NS+'endEvent'):
                        issues.append({'scope': sid, 'rule': 'ACTIVITY_AFTER_GATEWAY', 'flow': flow.get('id'), 'source': source.get('id'), 'target': target.get('id'), 'source_type': source.tag.split('}')[-1], 'target_type': target.tag.split('}')[-1]})
        except ET.ParseError as error:
            issues.append({'rule': 'INVALID_XML', 'message': str(error)})
        results.append({'file': path.relative_to(root).as_posix(), 'sha256': hashlib.sha256(data).hexdigest(), 'issues': issues})
    return {'files_checked': len(results), 'files_with_findings': sum(bool(r['issues']) for r in results), 'finding_count': sum(len(r['issues']) for r in results), 'files': results}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('root', type=Path)
    args = parser.parse_args()
    if not args.root.is_dir():
        parser.error('root must be an existing directory')
    print(json.dumps(audit(args.root), indent=2, ensure_ascii=True))
