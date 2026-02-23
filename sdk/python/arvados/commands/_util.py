# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import dataclasses
import errno
import json
import logging
import operator
import os
import re
import signal
import sys
import functools

import typing as t

from .. import _internal

FILTER_STR_RE = re.compile(r'''
^\(
\ *(\w+)
\ *(<|<=|=|>=|>)
\ *(\w+)
\ *\)$
''', re.ASCII | re.VERBOSE)

T = t.TypeVar('T')

@dataclasses.dataclass(unsafe_hash=True)
class RangedValue(t.Generic[T]):
    """Validate that an argument string is within a valid range of values"""
    parse_func: t.Callable[[str], T]
    valid_range: t.Container[T]

    def __call__(self, s: str) -> T:
        value = self.parse_func(s)
        if value in self.valid_range:
            return value
        else:
            raise ValueError(f"{value!r} is not a valid value")


@dataclasses.dataclass(unsafe_hash=True)
class UniqueSplit(t.Generic[T]):
    """Parse a string into a list of unique values"""
    split: t.Callable[[str], t.Iterable[str]]=operator.methodcaller('split', ',')
    clean: t.Callable[[str], str]=operator.methodcaller('strip')
    check: t.Callable[[str], bool]=bool

    def __call__(self, s: str) -> T:
        return list(_internal.uniq(_internal.parse_seq(s, self.split, self.clean, self.check)))


retry_opt = argparse.ArgumentParser(add_help=False)
retry_opt.add_argument(
    '--retries',
    type=RangedValue(int, range(0, sys.maxsize)),
    default=10,
    help="""Maximum number of times to retry server requests that encounter
temporary failures (e.g., server down).  Default %(default)r.
""")

def _ignore_error(error):
    return None

def _raise_error(error):
    raise error

CAUGHT_SIGNALS = [signal.SIGINT, signal.SIGQUIT, signal.SIGTERM]

def exit_signal_handler(sigcode, frame):
    logging.getLogger('arvados').error("Caught signal {}, exiting.".format(sigcode))
    sys.exit(-sigcode)

def install_signal_handlers():
    global orig_signal_handlers
    orig_signal_handlers = {sigcode: signal.signal(sigcode, exit_signal_handler)
                            for sigcode in CAUGHT_SIGNALS}

def restore_signal_handlers():
    for sigcode, orig_handler in orig_signal_handlers.items():
        signal.signal(sigcode, orig_handler)

def validate_filters(filters):
    """Validate user-provided filters

    This function validates that a user-defined object represents valid
    Arvados filters that can be passed to an API client: that it's a list of
    3-element lists with the field name and operator given as strings. If any
    of these conditions are not true, it raises a ValueError with details about
    the problem.

    It returns validated filters. Currently the provided filters are returned
    unmodified. Future versions of this function may clean up the filters with
    "obvious" type conversions, so callers SHOULD use the returned value for
    Arvados API calls.
    """
    if not isinstance(filters, list):
        raise ValueError(f"filters are not a list: {filters!r}")
    for index, f in enumerate(filters):
        if isinstance(f, str):
            match = FILTER_STR_RE.fullmatch(f)
            if match is None:
                raise ValueError(f"filter at index {index} has invalid syntax: {f!r}")
            s, op, o = match.groups()
            if s[0].isdigit():
                raise ValueError(f"filter at index {index} has invalid syntax: bad field name {s!r}")
            if o[0].isdigit():
                raise ValueError(f"filter at index {index} has invalid syntax: bad field name {o!r}")
            continue
        elif not isinstance(f, list):
            raise ValueError(f"filter at index {index} is not a string or list: {f!r}")
        try:
            s, op, o = f
        except ValueError:
            raise ValueError(
                f"filter at index {index} does not have three items (field name, operator, operand): {f!r}",
            ) from None
        if not isinstance(s, str):
            raise ValueError(f"filter at index {index} field name is not a string: {s!r}")
        if not isinstance(op, str):
            raise ValueError(f"filter at index {index} operator is not a string: {op!r}")
    return filters


class JSONStringArgument:
    """Callable JSON input parser with post-parsing validation function.

    This is designed to be used as an argparse argument type. Typical usage
    looks like:

        parser = argparse.ArgumentParser()
        parser.add_argument('--object', type=JSONStringArgument(), ...)

    When called on one string value, returns the result of parsing the value as
    JSON.

    If the parsing fails, or if the parsing succeeds but the result fails the
    further validation (if any), raises argparse.ArgumentTypeError with a
    suitable error message that will be printed to the stderr by argparse.

    The behavior may be further customized by providing the "validator" or
    "loader" callback functions; see the __init__ method documentation for
    details.

    By default, when initialized without any keyword arguments, it functions as
    a simple JSON loader.
    """
    def __init__(
        self,
        validator: t.Optional[t.Callable[[t.Any], t.Any]] = None,
        loader: t.Optional[t.Callable[[str], t.Any]] = None,
        pretty_name: str = "JSON"
    ):
        """Keyword arguments:

        * validator: callable --- optional callable that takes the JSON-parsing
          result (Python object) as input and performs additional validation
          after JSON-parsing. It should raise TypeError or ValueError to signal
          validation failure, and return the validated object (possibly
          modified) when validation succeeds. Its return value will become the
          return value of __call__() (i.e. type conversion for the input
          argument value). In addition, it may raise argparse.ArgumentTypeError
          directly for finer-grained control of messaging.

        * loader: callable --- optional callable that is used to load the
          value passed to __call__(). By default, json.loads is used, but you
          may supply your own loader to handle exceptions. The loader should
          raise ValueError (of which json.JSONDecodeError is a subtype) to
          signal failure to handle the input value, or raise
          argparse.ArgumentTypeError directly for finer-grained control of
          messaging.

        * pretty_name: str --- used by argparse to pretty-print the error
          message when the input fails validation. It should be a brief
          human-readable name for the kind of value that the argument takes.
          Default: "JSON".
        """
        self.loader = loader if callable(loader) else json.loads
        self.post_validator = validator if callable(validator) else None
        self.pretty_name = pretty_name or "JSON"

    def __call__(self, value: str):
        is_ok = True
        callback_exc_msg = ""
        try:
            retval = self.loader(value)
        except ValueError as err:  # This covers json.JSONDecodeError too.
            is_ok = False
            calback_exc_msg = str(err)
        else:
            if self.post_validator is not None:
                try:
                    retval = self.post_validator(retval)
                except (ValueError, TypeError) as err:
                    is_ok = False
                    callback_exc_msg = str(err)
        if not is_ok:
            msg = f"{value!r} is not valid {self.pretty_name}."
            if callback_exc_msg:
                msg += f" Further info: {callback_exc_msg}"
            raise argparse.ArgumentTypeError(msg) from None
        return retval


def json_or_file_loader(value: str):
    """Loader function that accepts either a JSON string, or a file whose
    content can be read and parsed as JSON (including "-" which represents the
    standard input). This is intended to be used as a custom loader function
    for JSONStringArgument.
    """
    value_is_json = False
    value_is_path = False
    try:
        content = json.loads(value)
        value_is_json = True
    except json.JSONDecodeError:
        pass

    fh = None
    fh_open_error_msg = ""
    if value == "-":
        fh = sys.stdin
        value_is_path = True  # technically not path but we get fh anyway.
    else:
        try:
            fh = open(value, "rb")
            value_is_path = True
        # For "FileNotFoundError" (a specific subtype of OSError) we don't need
        # to print the redundant "further info" (the path); "ValueError"
        # indicates illegal characters in path and wouldn't contain much useful
        # info.
        except (FileNotFoundError, ValueError):
            pass
        # Other kinds of OSError typically indicate bona-fide file-access
        # error for existing file.
        except OSError as exc:
            fh_open_error_msg = str(exc)

    if value_is_json and value_is_path:
        assert value != "-"
        fh.close()
        raise argparse.ArgumentTypeError(
            f"{value!r} is both valid JSON and a readable file."
            " Please consider renaming the file."
        ) from None

    if not (value_is_json or value_is_path):
        msg = f"{value!r} is neither valid JSON nor a readable file."
        if fh_open_error_msg:
            msg += f" Further info when opening file: {fh_open_error_msg}"
        raise argparse.ArgumentTypeError(msg) from None

    if value_is_path:
        try:
            content = json.load(fh)
        except json.JSONDecodeError:
            if value == "-":
                msg = "content of standard input is not valid JSON."
            else:
                msg = (
                    f"{value!r} is neither valid JSON"
                    " nor a readable file containing valid JSON."
                )
            raise argparse.ArgumentTypeError(msg) from None
        finally:
            if value != "-":
                fh.close()

    return content


JSONArgument = functools.partial(
    JSONStringArgument, loader=json_or_file_loader
)
JSONArgument.__doc__ = """
Parse a JSON file from a command line argument string or path

JSONArgument objects can be called with a string and return an arbitrary
object. First it will try to decode the string as JSON. If that fails, it will
try to open a file at the path named by the string, and decode its content as
JSON. Or, if the input is the string "-" (a single dash), it will read the
standard input and try to decode the content as JSON.

You can construct JSONArgument with an optional validation function. If given,
it is called with the Python object decoded from the input JSON string. The
return value of the validation function replaces the original JSON-decoded
object. The validation function should raise ValueError or TypeError,
preferablly with a suitable message, if the object fails validation.
Alternatively, it can directly raise argparse.ArgumentTypeError for
finer-grained error message control.

Typical usage with argparse looks like:

    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--object',
        type=JSONArgument(/...optional validation function.../),
        ...
    )

Please see the documentation for JSONStringArgument for more details about the
optional validation function.
"""
