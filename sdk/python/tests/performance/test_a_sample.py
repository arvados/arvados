import unittest

from performance_profiler import PerformanceProfiler

class PerformanceTestSample(PerformanceProfiler):
    def func(self):
        print 'Hello'

    def test_performance(self):
        self.run_profiler('self.func()', 'test_sample')
