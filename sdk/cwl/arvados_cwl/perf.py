import time
import uuid

class Perf(object):
    def __init__(self, logger, name):
        self.logger = logger
        self.name = name

    def __enter__(self):
        self.time = time.time()
        self.logger.debug("ENTER %s %s", self.name, self.time)

    def __exit__(self, exc_type=None, exc_value=None, traceback=None):
        now = time.time()
        self.logger.debug("EXIT %s %s %s", self.name, now, now - self.time)
