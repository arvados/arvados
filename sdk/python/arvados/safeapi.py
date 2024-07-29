# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""arvados.safeapi - Shim compatibility module

This module used to define `arvados.safeapi.ThreadSafeApiCache`. Now it only
exists to provide backwards compatible imports. New code should prefer to
import `arvados.api`.

@private
"""

from .api import ThreadSafeAPIClient as ThreadSafeApiCache
