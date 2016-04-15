import os
import unittest

slow_test = lambda _: unittest.skipIf(
    os.environ.get('ENABLE_SLOW_TESTS', '') != '',
    "ENABLE_SLOW_TESTS is not set")
