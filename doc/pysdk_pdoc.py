#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
"""pysdk_pdoc.py - Run pdoc with extra rendering options

This script is a wrapper around the standard `pdoc` tool that enables the
`admonitions` and `smarty-pants` extras for nicer rendering. It checks that
the version of `markdown2` included with `pdoc` supports those extras.

If run without arguments, it uses arguments to build the Arvados Python SDK
documentation.
"""

import collections
import functools
import os
import sys

try:
    import pdoc.__main__
    import pdoc.markdown2
    import pdoc.render_helpers
except ImportError as err:
    if __name__ == '__main__':
        _imp_err = err
    else:
        raise
else:
    _imp_err = None

DEFAULT_ARGLIST = [
    '--output-directory=sdk/python',
    '../sdk/python/build/lib/arvados/',
    # Because the module is prviate, pdoc does not build documentation for any
    # of it. The exclusion below additionally prevents pdoc from hyperlinking
    # references under arvados._internal that appear in method signatures, etc.
    '!arvados._internal',
]
MD_EXTENSIONS = {
    'admonitions': None,
    'smarty-pants': None,
}

def main(arglist=None):
    if _imp_err is not None:
        print("error: failed to import pdoc:", _imp_err, file=sys.stderr)
        return os.EX_SOFTWARE
    # Ensure markdown2 is new enough to support our desired extras.
    elif pdoc.markdown2.__version_info__ < (2, 4, 3):
        print("error: need markdown2>=2.4.3 to render admonitions", file=sys.stderr)
        return os.EX_SOFTWARE

    # Configure pdoc to use extras we want.
    pdoc.render_helpers.markdown_extensions = collections.ChainMap(
        pdoc.render_helpers.markdown_extensions,
        MD_EXTENSIONS,
    )
    pdoc.__main__.cli(arglist)
    return os.EX_OK

if __name__ == '__main__':
    sys.exit(main(sys.argv[1:] or DEFAULT_ARGLIST))
