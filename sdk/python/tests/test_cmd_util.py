# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import contextlib
import re
import copy
import itertools
import functools
import json
import os
import io
import tempfile
import unittest
import argparse

from pathlib import Path

import pytest
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


# Used for matching verbatim error messages.
def verbatim(text: str) -> str:
    return "^" + re.escape(text) + "$"


def _get_json_decode_error(text: str) -> str:
    try:
        json.loads(text)
    except json.JSONDecodeError as err:
        return str(err)


JSON_OBJECTS = (
    None,
    123,
    -456.789,
    'string',
    ['list', 1],
    {'object': True, 'yaml': False},
)
INVALID_JSON = ("", "\n", "\0", "foo", "[0, 1,]", "{", "'foo'")


class TestJSONStringArgument:

    def test_init_loader_not_callable(self):
        bad_arg_type = cmd_util.JSONStringArgument(loader=1)
        with pytest.raises(TypeError, match='is not callable'):
            bad_arg_type('"foo"')

    def test_init_validator_not_callable(self):
        value = '"foo"'
        bad_fcn = 1
        bad_fcn_type_name = type(bad_fcn).__name__
        arg_type_name = "test widget"
        msg_match = re.escape(
            f"{value!r} is not valid {arg_type_name}:"
            f" {bad_fcn_type_name!r} object is not callable"
        )
        bad_arg_type = cmd_util.JSONStringArgument(
            validator=bad_fcn, pretty_name=arg_type_name)
        with pytest.raises(argparse.ArgumentTypeError, match=msg_match):
            bad_arg_type(value)

    def test_init_pretty_name_false(self):
        parser = cmd_util.JSONStringArgument(pretty_name=False)
        assert parser.pretty_name == "JSON"

    @pytest.mark.parametrize("expected", JSON_OBJECTS)
    def test_plain_valid(self, expected):
        value = json.dumps(expected)
        parser = cmd_util.JSONStringArgument()
        assert parser(value) == expected

    @pytest.mark.parametrize("value", INVALID_JSON)
    def test_plain_invalid(self, value):
        parser = cmd_util.JSONStringArgument()
        details = _get_json_decode_error(value)
        with pytest.raises(
            argparse.ArgumentTypeError,
            match=verbatim(f"{value!r} is not valid JSON: {details}")
        ):
            parser(value)

    def test_custom_loader(self):
        def reject(text):
            raise json.JSONDecodeError(
                f"invalid float constant: {text!r}",
                text,
                0
            )
        loader = functools.partial(json.loads, parse_constant=reject)
        parser = cmd_util.JSONStringArgument(loader=loader)
        value = "NaN"
        # Obtain detailed error message produced by the callback.
        try:
            loader(value)
        except json.JSONDecodeError as err:
            loader_err_msg = str(err)

        with pytest.raises(
            argparse.ArgumentTypeError,
            match=verbatim(f"{value!r} is not valid JSON: {loader_err_msg}")
        ):
            parser(value)

    @pytest.mark.parametrize("expected_valid,value", (
        (False, "0"), (True, "1")
    ))
    def test_custom_validator_pretty_name(self, expected_valid, value):
        further_msg = "{0!s} is small"
        name = "big JSON number"

        def is_big(number):
            if number < 1:
                raise ValueError(further_msg.format(number))
            return number

        parser = cmd_util.JSONStringArgument(
            validator=is_big, pretty_name=name
        )
        if expected_valid:
            assert parser(value) == json.loads(value)
        else:
            with pytest.raises(
                argparse.ArgumentTypeError,
                match=verbatim(
                    f"{value!r} is not valid {name}: "
                    + further_msg.format(json.loads(value))
                )
            ):
                parser(value)


class _CountOpenFDs(contextlib.AbstractContextManager):
    """Rudimentary context manager that checks for possible file descriptor
    leaks, by opening /dev/null and noting its numeric value before entering
    and after exiting.
    """
    def __init__(self):
        self.before = -1
        self.after = -1

    def __enter__(self):
        null_fd = os.open(os.devnull, os.O_RDONLY)
        os.close(null_fd)
        self.before = null_fd

    def __exit__(self, exc_type, exc_val, exc_tb):
        null_fd = os.open(os.devnull, os.O_RDONLY)
        os.close(null_fd)
        self.after = null_fd

    def assert_no_leak(self):
        assert self.before >= 0
        assert self.after >= 0
        assert self.before == self.after


# Private context manager for cleanly and temporarily switching the working
# directory.
@contextlib.contextmanager
def _pushd(target):
    oldpwd = os.getcwd()
    try:
        os.chdir(target)
        yield
    finally:
        os.chdir(oldpwd)


@pytest.mark.usefixtures("tmp_path")
class TestJsonOrFileLoader:
    """Lower-level tests for the json_or_file_loader function that is plugged
    as a loader callback to JSONStringArgument to make JSONArgument.
    """
    @pytest.mark.parametrize(
        "test_id,content", enumerate(("invalid json", '["valid json"]'))
    )
    def test_no_file_descriptor_leak(self, tmp_path, test_id, content):
        f = tmp_path / f"test{test_id!s}.json"
        f.write_text(content)
        check_fd = _CountOpenFDs()
        with check_fd:
            try:
                cmd_util.json_or_file_loader(str(f))
            except argparse.ArgumentTypeError:
                pass
        check_fd.assert_no_leak()

    @pytest.mark.parametrize("expected", JSON_OBJECTS)
    def test_read_stdin(self, expected):
        with unittest.mock.patch(
            "sys.stdin", new_callable=io.StringIO
        ) as mock_stdin:
            mock_stdin.write(json.dumps(expected))
            mock_stdin.seek(0)
            actual = cmd_util.json_or_file_loader("-")
            assert not mock_stdin.closed
            assert actual == expected

    @pytest.mark.parametrize("input_value", INVALID_JSON)
    def test_reject_invalid_stdin_content(self, input_value):
        details = _get_json_decode_error(input_value)
        err_notes = verbatim(
            f"Content of standard input is not valid JSON: {details}"
        )
        with unittest.mock.patch(
            "sys.stdin", new_callable=io.StringIO
        ) as mock_stdin:
            mock_stdin.write(input_value)
            mock_stdin.seek(0)
            with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
                cmd_util.json_or_file_loader("-")

    @pytest.mark.parametrize("value", INVALID_JSON)
    def test_reject_file_with_invalid_json(self, tmp_path, value):
        f = tmp_path / "test.not-json"
        f.write_text(value)
        details = _get_json_decode_error(value)
        err_notes = verbatim(
            f"Content of file {str(f)!r} is not valid JSON: {details}"
        )
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            cmd_util.json_or_file_loader(str(f))

    def test_reject_file_name_resembling_json(self, tmp_path):
        crafted_basename = '"foo"'  # basename is valid JSON
        tmp_file = tmp_path / crafted_basename
        tmp_file.write_text(" ")  # ensure file exists; content doesn't matter.
        err_notes = verbatim(
            f"{crafted_basename!r} is both valid JSON and a readable file."
            " Please consider renaming the file."
        )
        check_fd = _CountOpenFDs()
        # cd into the temp directory so that we can refer to the file with its
        # basename (which is valid JSON)
        with pytest.raises(
            argparse.ArgumentTypeError, match=err_notes
        ), check_fd, _pushd(tmp_path):
            cmd_util.json_or_file_loader(crafted_basename)
        check_fd.assert_no_leak()

    def test_path_resembles_json_and_is_not_readable_file(self, tmp_path):
        # Input is both valid JSON string and existing directory (not readable
        # file). The resemblance of file name to JSON should not matter; it is
        # treated as just another OSError case, and we expect that the
        # offending path appears in the exception details.
        crafted_name = '"bar"'
        tmp_dir = tmp_path / crafted_name
        os.mkdir(tmp_dir)
        # cd into the temp directory so that we can refer to the subdir with
        # its name (which is valid JSON)
        with pytest.raises(
            IsADirectoryError,  # A subclass of OSError.
            match=f"^.*: {re.escape(repr(crafted_name))}"
        ), _pushd(tmp_path):
            cmd_util.json_or_file_loader(crafted_name)

    def test_not_json_and_is_directory(self, tmp_path):
        path = tmp_path / "subdir"
        os.mkdir(path)
        with pytest.raises(
            IsADirectoryError,
            match=f"^.*: {re.escape(repr(str(path)))}"
        ):
            cmd_util.json_or_file_loader(str(path))

    def test_not_json_and_file_unreadable(self):
        bad_file = tempfile.NamedTemporaryFile()
        os.chmod(bad_file.fileno(), 0o000)
        path = bad_file.name

        @contextlib.contextmanager
        def ctx():  # restore mode
            try:
                yield
            finally:
                os.chmod(bad_file.fileno(), 0o600)

        with pytest.raises(
            PermissionError,  # A subclass of OSError
            match=f"^.*: {re.escape(repr(path))}"
        ), ctx():
            cmd_util.json_or_file_loader(path)

    def test_not_json_and_not_path(self):
        # This is a simple "file not found" case (open() raises
        # FileNotFoundError), and the error message should not contain the path
        # in the trailing part.
        with tempfile.NamedTemporaryFile() as gone_file:
            path = gone_file.name
        details = _get_json_decode_error(path)
        err_notes = verbatim(
            f"{path!r} is not a readable file or valid JSON"
            f" [JSON decoding error: {details}]"
        )
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            cmd_util.json_or_file_loader(path)

    def test_not_json_and_illegal_path(self):
        # Null byte in path, illegal on almost all platforms.
        path = "\0"
        details = _get_json_decode_error(path)
        err_notes = verbatim(f"{path!r} is not valid JSON: {details}")
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            cmd_util.json_or_file_loader(path)


class TestJSONArgument:

    @classmethod
    def setup_class(cls):
        cls.json_file = tempfile.NamedTemporaryFile(
            'w+',
            encoding='utf-8',
            prefix='argtest',
            suffix='.json',
        )
        cls.parser = cmd_util.JSONArgument()

    @classmethod
    def teardown_class(cls):
        cls.json_file.close()

    def setup_method(self):
        self.json_file.seek(0)
        self.json_file.truncate()

    @pytest.mark.parametrize("obj", JSON_OBJECTS)
    def test_valid_argument_string(self, obj):
        actual = self.parser(json.dumps(obj))
        assert actual == obj

    @pytest.mark.parametrize("obj", JSON_OBJECTS)
    def test_valid_argument_path(self, obj):
        json.dump(obj, self.json_file)
        self.json_file.flush()
        actual = self.parser(self.json_file.name)
        assert actual == obj

    @pytest.mark.parametrize("path", [FILE_PATH, None])
    def test_argument_path_not_json(self, path):
        if path is None:
            path = self.json_file.name
        details = _get_json_decode_error(str(path))
        err_notes = verbatim(
            f"Content of file {str(path)!r} is not valid JSON: {details}"
        )
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            self.parser(str(path))


class TestJSONArgumentValidation:
    @pytest.mark.parametrize("value", JSON_OBJECTS)
    def test_value_returned_from_validator(self, value):
        # This validator fakes validation by discarding the result of actual
        # JSON parsing (of the JSON string '{}') and replacing it with the
        # arbitrary object "value".
        parser = cmd_util.JSONArgument(lambda _: copy.deepcopy(value))
        assert parser('{}') == value

    @pytest.mark.parametrize("value", JSON_OBJECTS)
    def test_exception_raised_from_validator(self, value):
        pretty_name = "type for testing"
        json_value = json.dumps(value)
        err_detail = f"{json_value} fails validation"

        def raise_func(_):
            raise ValueError(err_detail)

        parser = cmd_util.JSONArgument(
            validator=raise_func, pretty_name=pretty_name
        )
        err_notes = verbatim(
            f"{json_value!r} is not valid {pretty_name}: {err_detail}"
        )
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            parser(json_value)

    @pytest.mark.parametrize(
        "filter_input",
        itertools.combinations(
            ValidateFiltersTestCase.VALID_FILTERS, 2
        )
    )
    def test_with_filter_validator_valid_filter(self, filter_input):
        parser = cmd_util.JSONArgument(
            validator=cmd_util.validate_filters,
            pretty_name="filter"
        )
        expected = list(filter_input)
        input_str = json.dumps(expected)
        assert parser(input_str) == expected

    def test_with_filter_validator_invalid_filter(self):
        parser = cmd_util.JSONArgument(
            validator=cmd_util.validate_filters,
            pretty_name="filter"
        )
        input_str = '[1]'
        # Obtain a copy of the detailed validation error message from the
        # lower-level function.
        try:
            cmd_util.validate_filters(json.loads(input_str))
        except ValueError as exc:
            validation_err = str(exc)
        # Check that the detailed validation error message is attached to the
        # argparse-generated message.
        err_notes = verbatim(
            f"{input_str!r} is not valid filter: {validation_err}"
        )
        with pytest.raises(argparse.ArgumentTypeError, match=err_notes):
            parser(input_str)


class TestRangedValue:
    @pytest.fixture(scope='class')
    def cmpint(self):
        return cmd_util.RangedValue(int, range(-1, 2))

    @pytest.mark.parametrize('s', ['-1', '0', '1'])
    def test_valid_values(self, cmpint, s):
        assert cmpint(s) == int(s)

    @pytest.mark.parametrize('s', ['foo', '-2', '2', '0.2', '', ' '])
    def test_invalid_values(self, cmpint, s):
        with pytest.raises(ValueError):
            cmpint(s)


class TestUniqueSplit:
    @pytest.fixture(scope='class')
    def argtype(self):
        return cmd_util.UniqueSplit()

    @pytest.mark.parametrize('arg', [
        'foo',
        'foo,bar',
        'foo, bar, baz',
        'foo , bar , baz , quux',
    ])
    def test_basic_parse(self, arg, argtype):
        expected = ['foo', 'bar', 'baz', 'quux'][:arg.count(',') + 1]
        assert argtype(arg) == expected

    @pytest.mark.parametrize('arg', [
        'foo, foo, bar',
        'foo, bar, foo',
        'foo, bar, bar',
    ])
    def test_uniqueness(self, arg, argtype):
        assert argtype(arg) == ['foo', 'bar']
