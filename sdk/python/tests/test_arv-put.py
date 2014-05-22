#!/usr/bin/env python

import os
import tempfile
import unittest

import arvados
import arvados.commands.put as arv_put
from arvados_testutil import ArvadosKeepLocalStoreTestCase

class ArvadosPutTest(ArvadosKeepLocalStoreTestCase):
    def test_simple_file_put(self):
        with self.make_test_file() as testfile:
            path = testfile.name
            arv_put.main(['--stream', '--no-progress', path])
        self.assertTrue(
            os.path.exists(os.path.join(os.environ['KEEP_LOCAL_STORE'],
                                        '098f6bcd4621d373cade4e832627b4f6')),
            "did not find file stream in Keep store")


if __name__ == '__main__':
    unittest.main()
