#!/usr/bin/env python

import logging
import os

# Set the ANMTEST_LOGLEVEL environment variable to enable logging at that level.
loglevel = os.environ.get('ANMTEST_LOGLEVEL', 'CRITICAL')
logging.basicConfig(level=getattr(logging, loglevel.upper()))

# Set the ANMTEST_TIMEOUT environment variable to the maximum amount of time to
# wait for tested actors to respond to important messages.  The default value
# is very conservative, because a small value may produce false negatives on
# slower systems.  If you're debugging a known timeout issue, however, you may
# want to set this lower to speed up tests.
pykka_timeout = int(os.environ.get('ANMTEST_TIMEOUT', '10'))
