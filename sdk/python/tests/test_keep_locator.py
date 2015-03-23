#!/usr/bin/env python
# -*- coding: utf-8 -*-

import datetime
import itertools
import random
import unittest

from arvados.keep import KeepLocator

class ArvadosKeepLocatorTest(unittest.TestCase):
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

    def base_locators(self, count=DEFAULT_TEST_COUNT):
        return ('+'.join(pair) for pair in
                itertools.izip(self.checksums(count), self.sizes(count)))

    def perm_hints(self, count=DEFAULT_TEST_COUNT):
        for sig, ts in itertools.izip(self.signatures(count),
                                      self.timestamps(count)):
            yield 'A{}@{}'.format(sig, ts)

    def test_good_locators_returned(self):
        for hint_gens in [(), (self.sizes(),),
                          (self.sizes(), self.perm_hints())]:
            for loc_data in itertools.izip(self.checksums(), *hint_gens):
                locator = '+'.join(loc_data)
                self.assertEqual(locator, str(KeepLocator(locator)))

    def test_nonchecksum_rejected(self):
        for badstr in ['', 'badbadbad', '8f9e68d957b504a29ba76c526c3145dj',
                       '+8f9e68d957b504a29ba76c526c3145d9',
                       '3+8f9e68d957b504a29ba76c526c3145d9']:
            self.assertRaises(ValueError, KeepLocator, badstr)

    def test_unknown_hints_accepted(self):
        base = next(self.base_locators(1))
        for weirdhint in ['Zfoo', 'Ybar234', 'Xa@b_c-372', 'W99']:
            locator = '+'.join([base, weirdhint])
            self.assertEqual(locator, str(KeepLocator(locator)))

    def test_bad_hints_rejected(self):
        base = next(self.base_locators(1))
        for badhint in ['', 'A', 'lowercase', '+32']:
            self.assertRaises(ValueError, KeepLocator,
                              '+'.join([base, badhint]))

    def test_multiple_locator_hints_accepted(self):
        base = next(self.base_locators(1))
        for loc_hints in itertools.permutations(['Kab1cd', 'Kef2gh', 'Kij3kl']):
            locator = '+'.join((base,) + loc_hints)
            self.assertEqual(locator, str(KeepLocator(locator)))

    def test_expiry_passed(self):
        base = next(self.base_locators(1))
        signature = next(self.signatures(1))
        dt1980 = datetime.datetime(1980, 1, 1)
        dt2000 = datetime.datetime(2000, 2, 2)
        dt2080 = datetime.datetime(2080, 3, 3)
        locator = KeepLocator(base)
        self.assertFalse(locator.permission_expired())
        self.assertFalse(locator.permission_expired(dt1980))
        self.assertFalse(locator.permission_expired(dt2080))
        # Timestamped to 1987-01-05 18:48:32.
        locator = KeepLocator('{}+A{}@20000000'.format(base, signature))
        self.assertTrue(locator.permission_expired())
        self.assertTrue(locator.permission_expired(dt2000))
        self.assertFalse(locator.permission_expired(dt1980))


if __name__ == '__main__':
    unittest.main()
