# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import errno
import json
import logging
import os
import re
import signal
import sys

FILTER_STR_RE = re.compile(r'''
^\(
\ *(\w+)
\ *(<|<=|=|>=|>)
\ *(\w+)
\ *\)$
''', re.ASCII | re.VERBOSE)

def _pos_int(s):
    num = int(s)
    if num < 0:
        raise ValueError("can't accept negative value: %s" % (num,))
    return num

retry_opt = argparse.ArgumentParser(add_help=False)
retry_opt.add_argument('--retries', type=_pos_int, default=10, help="""
Maximum number of times to retry server requests that encounter temporary
failures (e.g., server down).  Default 10.""")

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


class JSONArgument:
    """Parse a JSON file from a command line argument string or path

    JSONArgument objects can be called with a string and return an arbitrary
    object. First it will try to decode the string as JSON. If that fails, it
    will try to open a file at the path named by the string, and decode it as
    JSON. If that fails, it raises ValueError with more detail.

    This is designed to be used as an argparse argument type.
    Typical usage looks like:

        parser = argparse.ArgumentParser()
        parser.add_argument('--object', type=JSONArgument(), ...)

    You can construct JSONArgument with an optional validation function. If
    given, it is called with the object decoded from user input, and its
    return value replaces it. It should raise ValueError if there is a problem
    with the input. (argparse turns ValueError into a useful error message.)

        filters_type = JSONArgument(validate_filters)
        parser.add_argument('--filters', type=filters_type, ...)
    """
    def __init__(self, validator=None):
        self.validator = validator

    def __call__(self, value):
        try:
            retval = json.loads(value)
        except json.JSONDecodeError:
            try:
                with open(value, 'rb') as json_file:
                    retval = json.load(json_file)
            except json.JSONDecodeError as error:
                raise ValueError(f"error decoding JSON from file {value!r}: {error}") from None
            except (FileNotFoundError, ValueError):
                raise ValueError(f"not a valid JSON string or file path: {value!r}") from None
            except OSError as error:
                raise ValueError(f"error reading JSON file path {value!r}: {error.strerror}") from None
        if self.validator is not None:
            retval = self.validator(retval)
        return retval
