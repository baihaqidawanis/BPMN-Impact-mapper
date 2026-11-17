import unittest
from evaluate_agent_selection import evaluate, selection_metrics, ranking_metrics, unique_files


class EvaluationTests(unittest.TestCase):
    def test_selection_duplicates_and_missing_files(self):
        result = selection_metrics(['a', 'a', 'x'], ['a', 'b'])
        self.assertEqual((result['precision'], result['recall'], result['f1']), (.5, .5, .5))
        self.assertEqual(result['missed'], 1)
        self.assertEqual(selection_metrics([], ['a'])['recall'], 0)

    def test_rank_positions_are_not_selection_precision(self):
        result = ranking_metrics(['x', 'a', 'b'], ['a', 'b'])
        self.assertEqual(result['average_precision'], .583333)
        self.assertEqual(result['reciprocal_rank'], .5)
        self.assertEqual(result['r_precision'], .5)

    def test_relative_paths_only(self):
        self.assertEqual(unique_files([{'file': 'a/b'}, {'file': 'a\\b'}]), ['a/b'])
        for name in ['C:/secret', '/tmp/secret', '../secret']:
            with self.assertRaises(ValueError):
                unique_files([{'file': name}])

    def test_scope_input_binding_and_unranked_draft(self):
        session = dict(id='s', status='NEEDS_REVIEW', identity=dict(model='test', prompt_version='v1'),
                       bundle=dict(analysis_id='a', input_fingerprint='f'),
                       draft=[dict(change_type='ADD_TASK', element_id='T', file='a'),
                              dict(change_type='ADD_TASK', element_id='Other', file='x')])
        report = dict(analysis_id='a', input_fingerprint='f', changes=[
            dict(change_type='ADD_TASK', new_element=dict(id='T'), impacts=[dict(file='a')])])
        reference = dict(cases=[dict(change_type='ADD_TASK', element_id='T', expected=[dict(file='a')])])
        result = evaluate(session, report, reference)
        agent = result['cases'][0]['agent']
        self.assertEqual(agent['files'], ['a'])
        self.assertEqual(agent['precision'], 1)
        self.assertIsNone(agent['map'])
        self.assertEqual(result['session_status'], 'NEEDS_REVIEW')
        report['input_fingerprint'] = 'different'
        with self.assertRaises(ValueError):
            evaluate(session, report, reference)


if __name__ == '__main__':
    unittest.main()
