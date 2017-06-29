#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import _strptime  # See <http://bugs.python.org/issue7980#msg221094>.
import logging

logger = logging.getLogger('arvnodeman')
logger.addHandler(logging.NullHandler())
