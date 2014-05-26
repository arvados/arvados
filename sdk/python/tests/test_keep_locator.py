#!/usr/bin/env python
# -*- coding: utf-8 -*-

import datetime
import itertools
import random
import unittest

from arvados.keep import KeepLocator

class ArvadosPutResumeCacheTest(unittest.TestCase):
    DEFAULT_TEST_COUNT = 10

    def numstrs(fmtstr, base, exponent):
        def genstrs(self, count=None):
            return (fmtstr.format(random.randint(0, base ** exponent))
                    for c in xrange(count or self.DEFAULT_TEST_COUNT))
        return genstrs

    checksums = numstrs('{:032x}', 16, 32)
    sizes = numstrs('{:d}', 2, 26)
    signatures = numstrs('{:040x}', 16, 40)
    timestamps = numstrs('{:08x}', 16, 8)

    def perm_hints(self, count=DEFAULT_TEST_COUNT):
        for sig, ts in itertools.izip(self.signatures(count),
                                      self.timestamps(count)):
            yield 'A{}@{}'.format(sig, ts)

    def test_good_locators_returned(self):
        for hint_gens in [(), (self.sizes(),), (self.perm_hints(),),
                          (self.sizes(), self.perm_hints())]:
            for loc_data in itertools.izip(self.checksums(), *hint_gens):
                locator = '+'.join(loc_data)
                self.assertEquals(locator, str(KeepLocator(locator)))

    def test_nonchecksum_rejected(self):
        for badstr in ['', 'badbadbad', '8f9e68d957b504a29ba76c526c3145dj',
                       '+8f9e68d957b504a29ba76c526c3145d9',
                       '3+8f9e68d957b504a29ba76c526c3145d9']:
            self.assertRaises(ValueError, KeepLocator, badstr)

    def test_bad_hints_rejected(self):
        checksum = next(self.checksums(1))
        for badhint in ['', 'nonsense', '+32', checksum]:
            self.assertRaises(ValueError, KeepLocator,
                              '+'.join([checksum, badhint]))

    def test_expiry_passed(self):
        checksum = next(self.checksums(1))
        signature = next(self.signatures(1))
        dt1980 = datetime.datetime(1980, 1, 1)
        dt2000 = datetime.datetime(2000, 2, 2)
        dt2080 = datetime.datetime(2080, 3, 3)
        locator = KeepLocator(checksum)
        self.assertFalse(locator.permission_expired())
        self.assertFalse(locator.permission_expired(dt1980))
        self.assertFalse(locator.permission_expired(dt2080))
        # Timestamped to 1987-01-05 18:48:32.
        locator = KeepLocator('{}+A{}@20000000'.format(checksum, signature))
        self.assertTrue(locator.permission_expired())
        self.assertTrue(locator.permission_expired(dt2000))
        self.assertFalse(locator.permission_expired(dt1980))


if __name__ == '__main__':
    unittest.main()
