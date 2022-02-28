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
        self._definition = voc_definition
        self.strict_keys = self._definition.get('strict_tags', False)
        self.key_aliases = {}

        for key_id, val in voc_definition.get('tags', {}).items():
            strict = val.get('strict', False)
            key_labels = [l['label'] for l in val.get('labels', [])]
            values = {}
            for v_id, v_val in val.get('values', {}).items():
                labels = [l['label'] for l in v_val.get('labels', [])]
                values[v_id] = VocabularyValue(v_id, labels)
            self.key_aliases[key_id] = VocabularyKey(key_id, key_labels, values, strict)

class VocabularyData(object):
    def __init__(self, identifier, aliases=[]):
        self.identifier = identifier
        self.aliases = set([x.lower() for x in aliases])

class VocabularyValue(VocabularyData):
    def __init__(self, identifier, aliases=[]):
        super(VocabularyValue, self).__init__(identifier, aliases)

class VocabularyKey(VocabularyData):
    def __init__(self, identifier, aliases=[], values={}, strict=False):
        super(VocabularyKey, self).__init__(identifier, aliases)
        self.values = values
        self.strict = strict