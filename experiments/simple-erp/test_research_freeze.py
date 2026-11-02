"""Research lock tests; no model or analyzed source execution."""
import json
import unittest
import uuid
from unittest.mock import patch

from research_freeze import MAPPER, compare, encoded, read_relative, render, sha, textual_snapshot


class FreezeTests(unittest.TestCase):
    def setUp(self):
        self.root = MAPPER / 'outputs' / ('freeze-test-' + uuid.uuid4().hex)
        self.root.mkdir(parents=True)
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for p in self.root.iterdir():
            p.unlink()
        self.root.rmdir()

    def test_safe_paths(self):
        for path in ('../escape', 'C:/secret', '/secret', 'nested\\secret'):
            with self.assertRaises(ValueError):
                read_relative(self.root, path)

    def test_text_snapshot_excludes_sensitive_binary_and_dependencies(self):
        (self.root / 'source.go').write_bytes(b'package fixture\n')
        (self.root / 'credentials.json').write_text('{}', encoding='utf-8')
        (self.root / '.env.json').write_text('{}', encoding='utf-8')
        (self.root / 'binary.txt').write_bytes(b'abc\x00def')
        names = 'source.go\x00credentials.json\x00.env.json\x00binary.txt\x00node_modules/dep.ts\x00removed.go\x00'
        with patch('research_freeze.subprocess.check_output', return_value=names.encode()):
            value = textual_snapshot(self.root)
        self.assertEqual(value, {'source.go': sha(b'package fixture\n')})

    def test_text_snapshot_detects_content_changes(self):
        source = self.root / 'source.go'
        source.write_bytes(b'first')
        with patch('research_freeze.subprocess.check_output', return_value=b'source.go\x00'):
            first = textual_snapshot(self.root)
            source.write_bytes(b'second')
            self.assertNotEqual(first, textual_snapshot(self.root))

    def test_frozen_integrity_split_and_label_sets(self):
        directory = MAPPER / 'experiments/simple-erp/research-freeze'
        v = json.loads((directory / 'freeze.json').read_bytes())
        self.assertEqual(v['freeze_id'], sha(encoded({k: x for k, x in v.items() if k != 'freeze_id'}))[:32])
        self.assertEqual(len(v['scenarios']), 10)
        self.assertEqual(v['splits']['independent_test'], [])
        self.assertEqual(set(v['splits']['development_regression']), {s['id'] for s in v['scenarios']})
        for s in v['scenarios']:
            self.assertTrue(s['prior_exposure'])
            self.assertEqual(s['split'], 'development-regression')
            labels = json.loads(read_relative(MAPPER, s['reference']))
            expected = {e['file'] for c in labels['cases'] for e in c['expected']}
            self.assertEqual(expected, {f['file'] for f in s['required_change_files']})
            for f in s['required_change_files']:
                self.assertTrue(f['reason'])
                self.assertTrue(f['evidence'])
        self.assertNotIn(b'C:/', encoded(v))
        self.assertNotIn(b'C:\\', encoded(v))
        compare(directory, v)

    def test_tampered_render_rejected(self):
        v = json.loads((MAPPER / 'experiments/simple-erp/research-freeze/freeze.json').read_bytes())
        (self.root / 'freeze.json').write_bytes(encoded(v))
        (self.root / 'README.md').write_bytes(render(v))
        compare(self.root, v)
        (self.root / 'README.md').write_text('changed', encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'Freeze drift'):
            compare(self.root, v)


if __name__ == '__main__':
    unittest.main()
