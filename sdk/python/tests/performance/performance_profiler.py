# Use the PerformanceProfiler class to write your performance tests.
#
# Usage:
#   from performance_profiler import PerformanceProfiler
#   self.run_profiler(...
#
#   See "test_a_sample.py" for a working example.
#
# To run performance tests:
#     cd arvados/sdk/python
#     python -m unittest discover tests.performance
#
#     Alternatively, using run-tests.sh
#         ./run-tests.sh WORKSPACE=~/arvados --only sdk/python sdk/python_test="--test-suite=tests.performance"
#

import os
import unittest
import sys
from datetime import datetime
try:
    import cProfile as profile
except ImportError:
    import profile

class PerformanceProfiler(unittest.TestCase):
    def run_profiler(self, function, test_name):
        filename = os.getcwd()+'/tmp/performance/'+ datetime.now().strftime('%Y-%m-%d-%H-%M-%S') +'-' +test_name

        directory = os.path.dirname(filename)
        if not os.path.exists(directory):
            os.makedirs(directory)

        sys.stdout = open(filename, 'w')
        profile.runctx(function, globals(), locals())
