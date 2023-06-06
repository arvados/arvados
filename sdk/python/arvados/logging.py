# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Logging utilities for Arvados clients"""

import logging

log_format = '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s'
log_date_format = '%Y-%m-%d %H:%M:%S'
log_handler = logging.StreamHandler()
log_handler.setFormatter(logging.Formatter(log_format, log_date_format))
