# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""arvados.cache - Shim compatibility module

This module used to define `arvados.cache.SafeHTTPCache`. Now it only exists
to provide backwards compatible imports. New code should prefer to import
`arvados.api.ThreadSafeHTTPCache`.

@private
"""

from .api import ThreadSafeHTTPCache as SafeHTTPCache
