# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging

from . import api

_logger = logging.getLogger('arvados.vocabulary')

def load_vocabulary(api_client=api('v1')):
    """Load the Arvados vocabulary from the API.
    """
    return Vocabulary(api_client.vocabulary())

class Vocabulary(object):
    def __init__(self, voc_definition={}):
        self.strict_keys = voc_definition.get('strict_tags', False)
        self.key_aliases = {}

        for key_id, val in voc_definition.get('tags', {}).items():
            strict = val.get('strict', False)
            key_labels = [l['label'] for l in val.get('labels', [])]
            values = {}
            for v_id, v_val in val.get('values', {}).items():
                labels = [l['label'] for l in v_val.get('labels', [])]
                values[v_id] = VocabularyValue(v_id, labels)
            vk = VocabularyKey(key_id, key_labels, values, strict)
            self.key_aliases[key_id] = vk
            for alias in vk.aliases:
                self.key_aliases[alias.lower()] = vk

    def __getitem__(self, key):
        return self.key_aliases[key.lower()]

class VocabularyData(object):
    def __init__(self, identifier, aliases=[]):
        self.identifier = identifier
        self.aliases = aliases

    def __getattribute__(self, name):
        if name == 'preferred_label':
            return self.aliases[0]
        return super(VocabularyData, self).__getattribute__(name)

class VocabularyValue(VocabularyData):
    def __init__(self, identifier, aliases=[]):
        super(VocabularyValue, self).__init__(identifier, aliases)

class VocabularyKey(VocabularyData):
    def __init__(self, identifier, aliases=[], values={}, strict=False):
        super(VocabularyKey, self).__init__(identifier, aliases)
        self.strict = strict
        self.value_aliases = {}
        for v_id, v_val in values.items():
            self.value_aliases[v_id] = v_val
            for v_alias in v_val.aliases:
                self.value_aliases[v_alias.lower()] = v_val

    def __getitem__(self, key):
        return self.value_aliases[key.lower()]