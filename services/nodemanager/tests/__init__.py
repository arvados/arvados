#!/usr/bin/env python

import logging
import os

# Set the ANMTEST_LOGLEVEL environment variable to enable logging at that level.
loglevel = os.environ.get('ANMTEST_LOGLEVEL', 'CRITICAL')
logging.basicConfig(level=getattr(logging, loglevel.upper()))

# Many tests wait for an actor to call a mock method.  They poll very
# regularly (see wait_for_call in ActorTestMixin), but if you've
# broken something, a long timeout can mean you'll spend a lot of time
# watching failures come in.  You can set the ANMTEST_TIMEOUT
# environment variable to arrange a shorter timeout while you're doing
# regular development.
pykka_timeout = int(os.environ.get('ANMTEST_TIMEOUT', '10'))
