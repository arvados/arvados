# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import unittest
import mock

from arvados import api, vocabulary

class VocabularyTest(unittest.TestCase):
    EXAMPLE_VOC = {
        'tags': {
            'IDTAGANIMALS': {
                'strict': False,
                'labels': [
                    {'label': 'Animal'},
                    {'label': 'Creature'},
                ],
                'values': {
                    'IDVALANIMAL1': {
                        'labels': [
                            {'label': 'Human'},
                            {'label': 'Homo sapiens'},
                        ],
                    },
                    'IDVALANIMAL2': {
                        'labels': [
                            {'label': 'Elephant'},
                            {'label': 'Loxodonta'},
                        ],
                    },
                },
            },
            'IDTAGIMPORTANCES': {
                'strict': True,
                'labels': [
                    {'label': 'Importance'},
                    {'label': 'Priority'},
                ],
                'values': {
                    'IDVALIMPORTANCE1': {
                        'labels': [
                            {'label': 'High'},
                            {'label': 'High priority'},
                        ],
                    },
                    'IDVALIMPORTANCE2': {
                        'labels': [
                            {'label': 'Medium'},
                            {'label': 'Medium priority'},
                        ],
                    },
                    'IDVALIMPORTANCE3': {
                        'labels': [
                            {'label': 'Low'},
                            {'label': 'Low priority'},
                        ],
                    },
                },
            },
            'IDTAGCOMMENTS': {
                'strict': False,
                'labels': [
                    {'label': 'Comment'},
                    {'label': 'Notes'},
                ],
                'values': None,
            },
        },
    }

    def setUp(self):
        self.api = arvados.api('v1')
        self.voc = vocabulary.Vocabulary(self.EXAMPLE_VOC)
        self.api.vocabulary = mock.MagicMock(return_value=self.EXAMPLE_VOC)

    def test_vocabulary_keys(self):
        self.assertEqual(self.voc.strict_keys, False)
        self.assertEqual(
            self.voc.key_aliases.keys(),
            set(['idtaganimals', 'creature', 'animal',
                'idtagimportances', 'importance', 'priority',
                'idtagcomments', 'comment', 'notes'])
        )

        vk = self.voc.key_aliases['creature']
        self.assertEqual(vk.strict, False)
        self.assertEqual(vk.identifier, 'IDTAGANIMALS')
        self.assertEqual(vk.aliases, ['Animal', 'Creature'])
        self.assertEqual(vk.preferred_label, 'Animal')
        self.assertEqual(
            vk.value_aliases.keys(),
            set(['idvalanimal1', 'human', 'homo sapiens',
                'idvalanimal2', 'elephant', 'loxodonta'])
        )

    def test_vocabulary_values(self):
        vk = self.voc.key_aliases['creature']
        vv = vk.value_aliases['human']
        self.assertEqual(vv.identifier, 'IDVALANIMAL1')
        self.assertEqual(vv.aliases, ['Human', 'Homo sapiens'])
        self.assertEqual(vv.preferred_label, 'Human')

    def test_vocabulary_indexing(self):
        self.assertEqual(self.voc['creature']['human'].identifier, 'IDVALANIMAL1')
        self.assertEqual(self.voc['Creature']['Human'].identifier, 'IDVALANIMAL1')
        self.assertEqual(self.voc['CREATURE']['HUMAN'].identifier, 'IDVALANIMAL1')
        with self.assertRaises(KeyError):
            inexistant = self.voc['foo']

    def test_empty_vocabulary(self):
        voc = vocabulary.Vocabulary({})
        self.assertEqual(voc.strict_keys, False)
        self.assertEqual(voc.key_aliases, {})

    def test_load_vocabulary_with_api(self):
        voc = vocabulary.load_vocabulary(self.api)
        self.assertEqual(voc['creature']['human'].identifier, 'IDVALANIMAL1')
        self.assertEqual(voc['Creature']['Human'].identifier, 'IDVALANIMAL1')
        self.assertEqual(voc['CREATURE']['HUMAN'].identifier, 'IDVALANIMAL1')

    def test_convert_to_identifiers(self):
        cases = [
            {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1'},
            {'IDTAGIMPORTANCES': 'High'},
            {'importance': 'IDVALIMPORTANCE1'},
            {'priority': 'high priority'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_identifiers(case),
                {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1'},
                "failing test case: {}".format(case)
            )

    def test_convert_to_identifiers_multiple_pairs(self):
        cases = [
            {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1', 'IDTAGANIMALS': 'IDVALANIMAL1', 'IDTAGCOMMENTS': 'Very important person'},
            {'IDTAGIMPORTANCES': 'High', 'IDTAGANIMALS': 'IDVALANIMAL1', 'comment': 'Very important person'},
            {'importance': 'IDVALIMPORTANCE1', 'animal': 'IDVALANIMAL1', 'notes': 'Very important person'},
            {'priority': 'high priority', 'animal': 'IDVALANIMAL1', 'NOTES': 'Very important person'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_identifiers(case),
                {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1', 'IDTAGANIMALS': 'IDVALANIMAL1', 'IDTAGCOMMENTS': 'Very important person'},
                "failing test case: {}".format(case)
            )

    def test_convert_to_identifiers_value_lists(self):
        cases = [
            {'IDTAGIMPORTANCES': ['IDVALIMPORTANCE1', 'IDVALIMPORTANCE2']},
            {'IDTAGIMPORTANCES': ['High', 'Medium']},
            {'importance': ['IDVALIMPORTANCE1', 'IDVALIMPORTANCE2']},
            {'priority': ['high', 'medium']},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_identifiers(case),
                {'IDTAGIMPORTANCES': ['IDVALIMPORTANCE1', 'IDVALIMPORTANCE2']},
                "failing test case: {}".format(case)
            )

    def test_convert_to_identifiers_unknown_key(self):
        # Non-strict vocabulary
        self.assertEqual(self.voc.strict_keys, False)
        self.assertEqual(self.voc.convert_to_identifiers({'foo': 'bar'}), {'foo': 'bar'})
        # Strict vocabulary
        strict_voc = arvados.vocabulary.Vocabulary(self.EXAMPLE_VOC)
        strict_voc.strict_keys = True
        with self.assertRaises(vocabulary.VocabularyKeyError):
            strict_voc.convert_to_identifiers({'foo': 'bar'})

    def test_convert_to_identifiers_invalid_key(self):
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_identifiers({42: 'bar'})
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_identifiers({None: 'bar'})
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_identifiers({('f', 'o', 'o'): 'bar'})

    def test_convert_to_identifiers_unknown_value(self):
        # Non-strict key
        self.assertEqual(self.voc['animal'].strict, False)
        self.assertEqual(self.voc.convert_to_identifiers({'Animal': 'foo'}), {'IDTAGANIMALS': 'foo'})
        # Strict key
        self.assertEqual(self.voc['priority'].strict, True)
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Priority': 'foo'})

    def test_convert_to_identifiers_invalid_value(self):
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': 42})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': None})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': {'hello': 'world'}})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': [42]})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': [None]})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Animal': [{'hello': 'world'}]})

    def test_convert_to_identifiers_unknown_value_list(self):
        # Non-strict key
        self.assertEqual(self.voc['animal'].strict, False)
        self.assertEqual(
            self.voc.convert_to_identifiers({'Animal': ['foo', 'loxodonta']}),
            {'IDTAGANIMALS': ['foo', 'IDVALANIMAL2']}
        )
        # Strict key
        self.assertEqual(self.voc['priority'].strict, True)
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_identifiers({'Priority': ['foo', 'bar']})

    def test_convert_to_labels(self):
        cases = [
            {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1'},
            {'IDTAGIMPORTANCES': 'High'},
            {'importance': 'IDVALIMPORTANCE1'},
            {'priority': 'high priority'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_labels(case),
                {'Importance': 'High'},
                "failing test case: {}".format(case)
            )

    def test_convert_to_labels_multiple_pairs(self):
        cases = [
            {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1', 'IDTAGANIMALS': 'IDVALANIMAL1', 'IDTAGCOMMENTS': 'Very important person'},
            {'IDTAGIMPORTANCES': 'High', 'IDTAGANIMALS': 'IDVALANIMAL1', 'comment': 'Very important person'},
            {'importance': 'IDVALIMPORTANCE1', 'animal': 'IDVALANIMAL1', 'notes': 'Very important person'},
            {'priority': 'high priority', 'animal': 'IDVALANIMAL1', 'NOTES': 'Very important person'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_labels(case),
                {'Importance': 'High', 'Animal': 'Human', 'Comment': 'Very important person'},
                "failing test case: {}".format(case)
            )

    def test_convert_to_labels_value_lists(self):
        cases = [
            {'IDTAGIMPORTANCES': ['IDVALIMPORTANCE1', 'IDVALIMPORTANCE2']},
            {'IDTAGIMPORTANCES': ['High', 'Medium']},
            {'importance': ['IDVALIMPORTANCE1', 'IDVALIMPORTANCE2']},
            {'priority': ['high', 'medium']},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_labels(case),
                {'Importance': ['High', 'Medium']},
                "failing test case: {}".format(case)
            )

    def test_convert_to_labels_unknown_key(self):
        # Non-strict vocabulary
        self.assertEqual(self.voc.strict_keys, False)
        self.assertEqual(self.voc.convert_to_labels({'foo': 'bar'}), {'foo': 'bar'})
        # Strict vocabulary
        strict_voc = arvados.vocabulary.Vocabulary(self.EXAMPLE_VOC)
        strict_voc.strict_keys = True
        with self.assertRaises(vocabulary.VocabularyKeyError):
            strict_voc.convert_to_labels({'foo': 'bar'})

    def test_convert_to_labels_invalid_key(self):
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_labels({42: 'bar'})
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_labels({None: 'bar'})
        with self.assertRaises(vocabulary.VocabularyKeyError):
            self.voc.convert_to_labels({('f', 'o', 'o'): 'bar'})

    def test_convert_to_labels_unknown_value(self):
        # Non-strict key
        self.assertEqual(self.voc['animal'].strict, False)
        self.assertEqual(self.voc.convert_to_labels({'IDTAGANIMALS': 'foo'}), {'Animal': 'foo'})
        # Strict key
        self.assertEqual(self.voc['priority'].strict, True)
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': 'foo'})

    def test_convert_to_labels_invalid_value(self):
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': {'high': True}})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': None})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': 42})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': False})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': [42]})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': [None]})
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': [{'high': True}]})

    def test_convert_to_labels_unknown_value_list(self):
        # Non-strict key
        self.assertEqual(self.voc['animal'].strict, False)
        self.assertEqual(
            self.voc.convert_to_labels({'IDTAGANIMALS': ['foo', 'IDVALANIMAL1']}),
            {'Animal': ['foo', 'Human']}
        )
        # Strict key
        self.assertEqual(self.voc['priority'].strict, True)
        with self.assertRaises(vocabulary.VocabularyValueError):
            self.voc.convert_to_labels({'IDTAGIMPORTANCES': ['foo', 'bar']})

    def test_convert_roundtrip(self):
        initial = {'IDTAGIMPORTANCES': 'IDVALIMPORTANCE1', 'IDTAGANIMALS': 'IDVALANIMAL1', 'IDTAGCOMMENTS': 'Very important person'}
        converted = self.voc.convert_to_labels(initial)
        self.assertNotEqual(converted, initial)
        self.assertEqual(self.voc.convert_to_identifiers(converted), initial)
