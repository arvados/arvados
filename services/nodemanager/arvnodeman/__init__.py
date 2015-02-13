#!/usr/bin/env python

from __future__ import absolute_import, print_function

import _strptime  # See <http://bugs.python.org/issue7980#msg221094>.
import logging

logger = logging.getLogger('arvnodeman')
logger.addHandler(logging.NullHandler())
