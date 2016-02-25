from __future__ import absolute_import, print_function

import errno
import logging
import os
import threading
import traceback

import pykka

class FullStopActor(pykka.ThreadingActor):
    def on_failure(self, exception_type, exception_value, tb):
        lg = getattr(self, "_logger", logging)
        if (exception_type in (threading.ThreadError, MemoryError) or
            exception_type is OSError and exception_value.errno == errno.ENOMEM):
            lg.critical("Unhandled exception is a fatal error, killing Node Manager")
            os.killpg(os.getpgid(0), 9)
