# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import print_function
from __future__ import absolute_import
from builtins import range
import unittest

from .performance_profiler import profiled

class PerformanceTestSample(unittest.TestCase):
    def foo(self):
        bar = 64

    @profiled
    def test_profiled_decorator(self):
        j = 0
        for i in range(0,2**20):
            j += i
        self.foo()
        print('Hello')
