import unittest

from performance_profiler import profiled

class PerformanceTestSample(unittest.TestCase):
    def foo(self):
        bar = 64

    @profiled
    def test_profiled_decorator(self):
        j = 0
        for i in range(0,2**20):
            j += i
        self.foo()
        print 'Hello'
