# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging

from . import api

_logger = logging.getLogger('arvados.vocabulary')

def load_vocabulary(api_client=None):
    """Load the Arvados vocabulary from the API.
    """
    if api_client is None:
        api_client = api('v1')
    return Vocabulary(api_client.vocabulary())

class VocabularyError(Exception):
    """Base class for all vocabulary errors.
    """
    pass

class VocabularyKeyError(VocabularyError):
    pass

class VocabularyValueError(VocabularyError):
    pass

class Vocabulary(object):
    def __init__(self, voc_definition={}):
        self.strict_keys = voc_definition.get('strict_tags', False)
        self.key_aliases = {}

        for key_id, val in (voc_definition.get('tags') or {}).items():
            strict = val.get('strict', False)
            key_labels = [l['label'] for l in val.get('labels', [])]
            values = {}
            for v_id, v_val in (val.get('values') or {}).items():
                labels = [l['label'] for l in v_val.get('labels', [])]
                values[v_id] = VocabularyValue(v_id, labels)
            vk = VocabularyKey(key_id, key_labels, values, strict)
            self.key_aliases[key_id.lower()] = vk
            for alias in vk.aliases:
                self.key_aliases[alias.lower()] = vk

    def __getitem__(self, key):
        return self.key_aliases[key.lower()]

    def convert_to_identifiers(self, obj={}):
        """Translate key/value pairs to machine readable identifiers.
        """
        return self._convert_to_what(obj, 'identifier')

    def convert_to_labels(self, obj={}):
        """Translate key/value pairs to human readable labels.
        """
        return self._convert_to_what(obj, 'preferred_label')

    def _convert_to_what(self, obj={}, what=None):
        if not isinstance(obj, dict):
            raise ValueError("obj must be a dict")
        if what not in ['preferred_label', 'identifier']:
            raise ValueError("what attr must be 'preferred_label' or 'identifier'")
        r = {}
        for k, v in obj.items():
            # Key validation & lookup
            key_found = False
            if not isinstance(k, str):
                raise VocabularyKeyError("key '{}' must be a string".format(k))
            k_what, v_what = k, v
            try:
                k_what = getattr(self[k], what)
                key_found = True
            except KeyError:
                if self.strict_keys:
                    raise VocabularyKeyError("key '{}' not found in vocabulary".format(k))

            # Value validation & lookup
            if isinstance(v, list):
                v_what = []
                for x in v:
                    if not isinstance(x, str):
                        raise VocabularyValueError("value '{}' for key '{}' must be a string".format(x, k))
                    try:
                        v_what.append(getattr(self[k][x], what))
                    except KeyError:
                        if self[k].strict:
                            raise VocabularyValueError("value '{}' not found for key '{}'".format(x, k))
                        v_what.append(x)
            else:
                if not isinstance(v, str):
                    raise VocabularyValueError("{} value '{}' for key '{}' must be a string".format(type(v).__name__, v, k))
                try:
                    v_what = getattr(self[k][v], what)
                except KeyError:
                    if key_found and self[k].strict:
                        raise VocabularyValueError("value '{}' not found for key '{}'".format(v, k))

            r[k_what] = v_what
        return r

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
            self.value_aliases[v_id.lower()] = v_val
            for v_alias in v_val.aliases:
                self.value_aliases[v_alias.lower()] = v_val

    def __getitem__(self, key):
        return self.value_aliases[key.lower()]