import __main__
import os
import unittest

slow_test = lambda _: unittest.skipIf(
    __main__.short_tests_only,
    "running --short tests only")
