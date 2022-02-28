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
            'IDTAGIMPORTANCE': {
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
                'idtagimportance', 'importance', 'priority'])
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
            {'IDTAGIMPORTANCE': 'IDVALIMPORTANCE1'},
            {'IDTAGIMPORTANCE': 'High'},
            {'importance': 'IDVALIMPORTANCE1'},
            {'priority': 'high priority'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_identifiers(case),
                {'IDTAGIMPORTANCE': 'IDVALIMPORTANCE1'},
                "failing test case: {}".format(case)
            )

    def test_convert_to_labels(self):
        cases = [
            {'IDTAGIMPORTANCE': 'IDVALIMPORTANCE1'},
            {'IDTAGIMPORTANCE': 'High'},
            {'importance': 'IDVALIMPORTANCE1'},
            {'priority': 'high priority'},
        ]
        for case in cases:
            self.assertEqual(
                self.voc.convert_to_labels(case),
                {'Importance': 'High'},
                "failing test case: {}".format(case)
            )