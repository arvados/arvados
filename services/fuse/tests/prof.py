import time

class CountTime(object):
    def __init__(self, tag="", size=None):
        self.tag = tag
        self.size = size

    def __enter__(self):
        self.start = time.time()
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        sec = (time.time() - self.start)
        th = ""
        if self.size:
            th = "throughput %s/sec" % (self.size / sec)
        print "%s time %s micoseconds %s" % (self.tag, sec*1000000, th)
