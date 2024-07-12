# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados internal utilities

Everything in `arvados._internal` is support code for the Arvados Python SDK
and tools. Nothing in this module is intended to be part of the public-facing
SDK API. Classes and functions in this module may be changed or removed at any
time.
"""

import functools
import re
import time
import warnings

class Timer:
    def __init__(self, verbose=False):
        self.verbose = verbose

    def __enter__(self):
        self.start = time.time()
        return self

    def __exit__(self, *args):
        self.end = time.time()
        self.secs = self.end - self.start
        self.msecs = self.secs * 1000  # millisecs
        if self.verbose:
            print('elapsed time: %f ms' % self.msecs)


def deprecated(version=None, preferred=None):
    """Mark a callable as deprecated in the SDK

    This will wrap the callable to emit as a DeprecationWarning
    and add a deprecation notice to its docstring.

    If the following arguments are given, they'll be included in the
    notices:

    * preferred: str | None --- The name of an alternative that users should
      use instead.

    * version: str | None --- The version of Arvados when the callable is
      scheduled to be removed.
    """
    if version is None:
        version = ''
    else:
        version = f' and scheduled to be removed in Arvados {version}'
    if preferred is None:
        preferred = ''
    else:
        preferred = f' Prefer {preferred} instead.'
    def deprecated_decorator(func):
        fullname = f'{func.__module__}.{func.__qualname__}'
        parent, _, name = fullname.rpartition('.')
        if name == '__init__':
            fullname = parent
        warning_msg = f'{fullname} is deprecated{version}.{preferred}'
        @functools.wraps(func)
        def deprecated_wrapper(*args, **kwargs):
            warnings.warn(warning_msg, DeprecationWarning, 2)
            return func(*args, **kwargs)
        # Get func's docstring without any trailing newline or empty lines.
        func_doc = re.sub(r'\n\s*$', '', func.__doc__ or '')
        match = re.search(r'\n([ \t]+)\S', func_doc)
        indent = '' if match is None else match.group(1)
        warning_doc = f'\n\n{indent}.. WARNING:: Deprecated\n{indent}   {warning_msg}'
        # Make the deprecation notice the second "paragraph" of the
        # docstring if possible. Otherwise append it.
        docstring, count = re.subn(
            rf'\n[ \t]*\n{indent}',
            f'{warning_doc}\n\n{indent}',
            func_doc,
            count=1,
        )
        if not count:
            docstring = f'{func_doc.lstrip()}{warning_doc}'
        deprecated_wrapper.__doc__ = docstring
        return deprecated_wrapper
    return deprecated_decorator
