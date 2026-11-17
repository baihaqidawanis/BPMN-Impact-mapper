import unittest
from audit_selection_errors import assess


class SelectionAuditTests(unittest.TestCase):
    def test_retrieval_miss_is_separate_from_filter_loss(self):
        result = assess(['a', 'b', 'c'], ['a', 'b', 'noise'], ['a', 'noise'], ['b'])
        self.assertEqual(result['retained_expected'], ['a'])
        self.assertEqual(result['missing_from_raw'], ['c'])
        self.assertEqual(result['lost_by_selection'], ['b'])
        self.assertEqual(result['selected_not_in_reference'], ['noise'])

    def test_partition_rejects_hidden_or_invented_candidates(self):
        for selected, deferred in [(['a'], []), (['a', 'b'], ['b']), (['a', 'made-up'], ['b'])]:
            with self.subTest(selected=selected, deferred=deferred):
                with self.assertRaises(ValueError):
                    assess(['a'], ['a', 'b'], selected, deferred)

    def test_zero_selection_preserves_all_deferred(self):
        result = assess(['a'], ['a', 'b'], [], ['a', 'b'])
        self.assertEqual(result['lost_by_selection'], ['a'])
        self.assertEqual(result['deferred_not_in_reference'], ['b'])


if __name__ == '__main__':
    unittest.main()
