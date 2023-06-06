# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Logging utilities for Arvados clients"""

import logging

log_format = '%(asctime)s %(name)s[%(process)d] %(levelname)s: %(message)s'
log_date_format = '%Y-%m-%d %H:%M:%S'
log_handler = logging.StreamHandler()
log_handler.setFormatter(logging.Formatter(log_format, log_date_format))

class GoogleHTTPClientFilter:
    """Common googleapiclient.http log filters for Arvados clients

    This filter makes `googleapiclient.http` log messages more useful for
    typical Arvados applications. Currently it only changes the level of
    retry messages (to INFO by default), but its functionality may be
    extended in the future. Typical usage looks like:

        logging.getLogger('googleapiclient.http').addFilter(GoogleHTTPClientFilter())
    """
    def __init__(self, *, retry_level='INFO'):
        self.retry_levelname = retry_level
        self.retry_levelno = getattr(logging, retry_level)

    def filter(self, record):
        if record.msg.startswith(('Sleeping ', 'Retry ')):
            record.levelname = self.retry_levelname
            record.levelno = self.retry_levelno
        return True
