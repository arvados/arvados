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

    def perform_vocabulary_tests(self, voc):
        self.assertEqual(voc.strict_keys, False)
        self.assertEqual(voc.key_aliases.keys(), set(['IDTAGANIMALS', 'IDTAGIMPORTANCE']))

        self.assertEqual(voc.key_aliases['IDTAGANIMALS'].strict, False)
        self.assertEqual(set(voc.key_aliases['IDTAGANIMALS'].aliases), set(['animal', 'creature']))
        self.assertEqual(voc.key_aliases['IDTAGANIMALS'].values.keys(), set(['IDVALANIMAL1', 'IDVALANIMAL2']))
        self.assertEqual(voc.key_aliases['IDTAGANIMALS'].values['IDVALANIMAL1'].aliases, set(['human', 'homo sapiens']))

    def test_empty_vocabulary(self):
        voc = vocabulary.Vocabulary()
        self.assertEqual(voc.strict_keys, False)
        self.assertEqual(voc.key_aliases, {})

    def test_load_vocabulary(self):
        voc = vocabulary.Vocabulary(self.EXAMPLE_VOC)
        self.perform_vocabulary_tests(voc)

    @mock.patch('arvados.api')
    def test_load_vocabulary_with_api(self, api_mock):
        api_mock.return_value = mock.MagicMock()
        api_mock.return_value.vocabulary.return_value = self.EXAMPLE_VOC

        voc = vocabulary.load_vocabulary(arvados.api('v1'))
        self.perform_vocabulary_tests(voc)
