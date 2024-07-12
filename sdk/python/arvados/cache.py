# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""arvados.cache - Shim compatibility module

This module used to define `arvados.api.SafeHTTPCache`. Now it only exists
to provide backwards compatible imports. New code should prefer to import
`arvados.api`.

@private
"""

from .api import SafeHTTPCache
