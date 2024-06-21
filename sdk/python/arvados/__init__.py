# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados Python SDK

This module provides the entire Python SDK for Arvados. The most useful modules
include:

* arvados.api - After you `import arvados`, you can call `arvados.api` as a
  shortcut to the client constructor function `arvados.api.api`.

* arvados.collection - The `arvados.collection.Collection` class provides a
  high-level interface to read and write collections. It coordinates sending
  data to and from Keep, and synchronizing updates with the collection object.

* arvados.util - Utility functions to use mostly in conjunction with the API
  client object and the results it returns.

Other submodules provide lower-level functionality.
"""

import logging as stdliblog
import os
import sys
import types

from collections import UserDict

from . import api, errors, util
from .api import api_from_config, http_cache
from .collection import CollectionReader
from arvados.keep import *
from .arvfile import StreamFileReader
from .logging import log_format, log_date_format, log_handler
from .retry import RetryLoop

# Previous versions of the PySDK used to say `from .api import api`.  This
# made it convenient to call the API client constructor, but difficult to
# access the rest of the `arvados.api` module. The magic below fixes that
# bug while retaining backwards compatibility: `arvados.api` is now the
# module and you can import it normally, but we make that module callable so
# all the existing code that says `arvados.api('v1', ...)` still works.
class _CallableAPIModule(api.__class__):
    __call__ = staticmethod(api.api)
api.__class__ = _CallableAPIModule

# Override logging module pulled in via `from ... import *`
# so users can `import arvados.logging`.
logging = sys.modules['arvados.logging']

# Set up Arvados logging based on the user's configuration.
# All Arvados code should log under the arvados hierarchy.
logger = stdliblog.getLogger('arvados')
logger.addHandler(log_handler)
logger.setLevel(stdliblog.DEBUG if config.get('ARVADOS_DEBUG')
                else stdliblog.WARNING)
