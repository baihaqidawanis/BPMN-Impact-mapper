"""Reference integrity tests. No ERP code/model/runtime execution."""
import copy
import json
import unittest
import uuid
from pathlib import Path

from build_role_traceability import MAPPER, SourceReview, artifacts, encoded, sha, validate


class EvidenceTests(unittest.TestCase):
    def setUp(self):
        (MAPPER / 'outputs').mkdir(exist_ok=True)
        self.root = MAPPER / 'outputs' / ('traceability-test-' + uuid.uuid4().hex)
        self.root.mkdir()
        self.addCleanup(self.cleanup_directory, self.root)

    @staticmethod
    def cleanup_directory(root):
        # Only direct children created by these tests; never recurse through a
        # link or delete a computed external repository path.
        if not root.resolve().is_relative_to((MAPPER / 'outputs').resolve()):
            raise ValueError('Unexpected test directory')
        for path in root.iterdir():
            if path.is_symlink() and path.is_dir():
                path.rmdir()
            else:
                path.unlink()
        root.rmdir()

    def test_function_scoping_and_receiver(self):
        (self.root / 'source.go').write_text('func (s Store) Read() {\n first()\n}\nfunc (s Other) Read() {\n second()\n}\n', encoding='utf-8')
        review = SourceReview(self.root)
        link = review.relation('source.go', 'Store.Read', 'READ_QUERY', ['first()'])
        self.assertEqual(link['evidence'][0]['line'], 2)
        with self.assertRaisesRegex(ValueError, 'Reviewed evidence changed'):
            review.relation('source.go', 'Store.Read', 'READ_QUERY', ['second()'])

    def test_ambiguous_function_rejected(self):
        (self.root / 'source.go').write_text('func Run() {}\nfunc Run() {}\n', encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'Function ambiguous'):
            SourceReview(self.root).relation('source.go', 'Run', 'CONTEXT', ['Run'])

    def test_dense_jsx_excerpt_keeps_match(self):
        (self.root / 'panel.tsx').write_text('export function Panel() {\n' + 'x' * 500 + '<button onClick={save}>Save</button>\n}\nfunction Other() { different() }\n', encoding='utf-8')
        review = SourceReview(self.root)
        link = review.relation('panel.tsx', 'Panel', 'INPUT', ['onClick={save}'])
        self.assertIn('onClick={save}', link['evidence'][0]['excerpt'])
        self.assertLessEqual(len(link['evidence'][0]['excerpt']), 326)
        with self.assertRaisesRegex(ValueError, 'Reviewed evidence changed'):
            review.relation('panel.tsx', 'Panel', 'INPUT', ['different()'])

    def test_paths_and_empty_evidence_rejected(self):
        review = SourceReview(self.root)
        for path in ('../escape', 'C:/secret', '/secret', 'nested\\secret'):
            with self.assertRaises(ValueError):
                review.read(path)
        (self.root / 'source.go').write_text('func Run() {}\n', encoding='utf-8')
        with self.assertRaisesRegex(ValueError, 'Evidence required'):
            review.relation('source.go', 'Run', 'CONTEXT', [])

    def test_symlink_escape_rejected(self):
        outside = MAPPER / 'outputs' / ('traceability-test-outside-' + uuid.uuid4().hex)
        outside.mkdir()
        self.addCleanup(self.cleanup_directory, outside)
        (outside / 'source.go').write_text('func Run() {}\n', encoding='utf-8')
        try:
            (self.root / 'link').symlink_to(outside, target_is_directory=True)
        except OSError:
            self.skipTest('Windows session has no symlink privilege')
        with self.assertRaisesRegex(ValueError, 'escapes repository'):
            SourceReview(self.root).read('link/source.go')


class SnapshotTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = MAPPER / 'experiments/simple-erp/baseline-traceability/all-roles'
        cls.value = json.loads((cls.directory / 'all-roles.json').read_bytes())

    def test_all_nine_cells_and_activity_coverage(self):
        validate(self.value)
        expected = {'staff': [50, 36, 3], 'supervisor': [51, 26, 13], 'super-admin': [101, 53, 21]}
        self.assertEqual(len(self.value['processes']), 72)
        self.assertEqual(len(self.value['hierarchy']), 40)
        for role, counts in expected.items():
            for module, count in zip(('p2p', 'inventory', 'master-data'), counts):
                rows = [a for p in self.value['processes'] if p['role'] == role and p['module'] == module for a in p['activities']]
                self.assertEqual(len(rows), count)
        # Role differences are deliberate, not copied permissions.
        staff_master = [p['operation'] for p in self.value['processes'] if p['role'] == 'staff' and p['module'] == 'master-data']
        self.assertEqual(staff_master, ['master-list'])
        self.assertTrue(all(p['role'] == 'super-admin' for p in self.value['processes'] if p['operation'].startswith('settings-')))

    def test_rendered_files_and_determinism(self):
        first, second = artifacts(self.value), artifacts(json.loads(encoded(self.value)))
        self.assertEqual(first, second)
        self.assertEqual(len(first), 20)
        for name, data in first.items():
            self.assertEqual((self.directory / name).read_bytes(), data, name)
        self.assertNotIn(b'C:\\', encoded(self.value))
        self.assertNotIn(b'C:/', encoded(self.value))

    def test_corruption_and_missing_mapping_rejected(self):
        altered = copy.deepcopy(self.value)
        altered['processes'][0]['activities'][0]['mappings'] = []
        with self.assertRaisesRegex(ValueError, 'snapshot hash'):
            validate(altered)
        altered['snapshot_id'] = sha(encoded({k: v for k, v in altered.items() if k != 'snapshot_id'}))[:32]
        with self.assertRaisesRegex(ValueError, 'incomplete activity'):
            validate(altered)

    def test_decomposition_targets_exist(self):
        processes = {p['process_id']: p['bpmn_file'] for p in self.value['processes'] + self.value['hierarchy']}
        for p in self.value['hierarchy']:
            for node in p['activities']:
                if node['called_process']:
                    self.assertEqual(node['target_bpmn'], processes[node['called_process']])


if __name__ == '__main__':
    unittest.main()
