# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import contextlib
import copy
import itertools
import json
import os
import tempfile
import unittest

from pathlib import Path

from parameterized import parameterized

import arvados.commands._util as cmd_util

FILE_PATH = Path(__file__)

class ValidateFiltersTestCase(unittest.TestCase):
    NON_FIELD_TYPES = [
        None,
        123,
        ('name', '=', 'tuple'),
        {'filters': ['name', '=', 'object']},
    ]
    NON_FILTER_TYPES = NON_FIELD_TYPES + ['string']
    VALID_FILTERS = [
        ['owner_uuid', '=', 'zzzzz-tpzed-12345abcde67890'],
        ['name', 'in', ['foo', 'bar']],
        '(replication_desired > replication_cofirmed)',
        '(replication_confirmed>=replication_desired)',
    ]

    @parameterized.expand(itertools.combinations(VALID_FILTERS, 2))
    def test_valid_filters(self, f1, f2):
        expected = [f1, f2]
        actual = cmd_util.validate_filters(copy.deepcopy(expected))
        self.assertEqual(actual, expected)

    @parameterized.expand([(t,) for t in NON_FILTER_TYPES])
    def test_filters_wrong_type(self, value):
        with self.assertRaisesRegex(ValueError, r'^filters are not a list\b'):
            cmd_util.validate_filters(value)

    @parameterized.expand([(t,) for t in NON_FIELD_TYPES])
    def test_single_filter_wrong_type(self, value):
        with self.assertRaisesRegex(ValueError, r'^filter at index 0 is not a string or list\b'):
            cmd_util.validate_filters([value])

    @parameterized.expand([
        ([],),
        (['owner_uuid'],),
        (['owner_uuid', 'zzzzz-tpzed-12345abcde67890'],),
        (['name', 'not in', 'foo', 'bar'],),
        (['name', 'in', 'foo', 'bar', 'baz'],),
    ])
    def test_filters_wrong_arity(self, value):
        with self.assertRaisesRegex(ValueError, r'^filter at index 0 does not have three items\b'):
            cmd_util.validate_filters([value])

    @parameterized.expand(itertools.product(
        [0, 1],
        NON_FIELD_TYPES,
    ))
    def test_filter_definition_wrong_type(self, index, bad_value):
        value = ['owner_uuid', '=', 'zzzzz-tpzed-12345abcde67890']
        value[index] = bad_value
        name = ('field name', 'operator')[index]
        with self.assertRaisesRegex(ValueError, rf'^filter at index 0 {name} is not a string\b'):
            cmd_util.validate_filters([value])

    @parameterized.expand([
        # Not enclosed in parentheses
        'foo = bar',
        '(foo) < bar',
        'foo > (bar)',
        # Not exactly one operator
        '(a >= b >= c)',
        '(foo)',
        '(file_count version)',
        # Invalid field identifiers
        '(version = 1)',
        '(2 = file_count)',
        '(replication.desired <= replication.confirmed)',
        # Invalid whitespace
        '(file_count\t=\tversion)',
        '(file_count >= version\n)',
    ])
    def test_invalid_string_filter(self, value):
        with self.assertRaisesRegex(ValueError, r'^filter at index 0 has invalid syntax\b'):
            cmd_util.validate_filters([value])


class JSONArgumentTestCase(unittest.TestCase):
    JSON_OBJECTS = [
        None,
        123,
        456.789,
        'string',
        ['list', 1],
        {'object': True, 'yaml': False},
    ]

    @classmethod
    def setUpClass(cls):
        cls.json_file = tempfile.NamedTemporaryFile(
            'w+',
            encoding='utf-8',
            prefix='argtest',
            suffix='.json',
        )
        cls.parser = cmd_util.JSONArgument()

    @classmethod
    def tearDownClass(cls):
        cls.json_file.close()

    def setUp(self):
        self.json_file.seek(0)
        self.json_file.truncate()

    @parameterized.expand((obj,) for obj in JSON_OBJECTS)
    def test_valid_argument_string(self, obj):
        actual = self.parser(json.dumps(obj))
        self.assertEqual(actual, obj)

    @parameterized.expand((obj,) for obj in JSON_OBJECTS)
    def test_valid_argument_path(self, obj):
        json.dump(obj, self.json_file)
        self.json_file.flush()
        actual = self.parser(self.json_file.name)
        self.assertEqual(actual, obj)

    @parameterized.expand([
        '',
        '\0',
        None,
    ])
    def test_argument_not_json_or_path(self, value):
        if value is None:
            with tempfile.NamedTemporaryFile() as gone_file:
                value = gone_file.name
        with self.assertRaisesRegex(ValueError, r'\bnot a valid JSON string or file path\b'):
            self.parser(value)

    @parameterized.expand([
        FILE_PATH.parent,
        FILE_PATH / 'nonexistent.json',
        None,
    ])
    def test_argument_path_unreadable(self, path):
        if path is None:
            bad_file = tempfile.NamedTemporaryFile()
            os.chmod(bad_file.fileno(), 0o000)
            path = bad_file.name
            @contextlib.contextmanager
            def ctx():
                try:
                    yield
                finally:
                    os.chmod(bad_file.fileno(), 0o600)
        else:
            ctx = contextlib.nullcontext
        with self.assertRaisesRegex(ValueError, rf'^error reading JSON file path {str(path)!r}: '), ctx():
            self.parser(str(path))

    @parameterized.expand([
        FILE_PATH,
        None,
    ])
    def test_argument_path_not_json(self, path):
        if path is None:
            path = self.json_file.name
        with self.assertRaisesRegex(ValueError, rf'^error decoding JSON from file {str(path)!r}'):
            self.parser(str(path))


class JSONArgumentValidationTestCase(unittest.TestCase):
    @parameterized.expand((obj,) for obj in JSONArgumentTestCase.JSON_OBJECTS)
    def test_object_returned_from_validator(self, value):
        parser = cmd_util.JSONArgument(lambda _: copy.deepcopy(value))
        self.assertEqual(parser('{}'), value)

    @parameterized.expand((obj,) for obj in JSONArgumentTestCase.JSON_OBJECTS)
    def test_exception_raised_from_validator(self, value):
        json_value = json.dumps(value)
        def raise_func(_):
            raise ValueError(json_value)
        parser = cmd_util.JSONArgument(raise_func)
        with self.assertRaises(ValueError) as exc_check:
            parser(json_value)
        self.assertEqual(exc_check.exception.args, (json_value,))
