import unittest
import os
import arvados.util

class MkdirDashPTest(unittest.TestCase):
    def setUp(self):
        try:
            os.path.mkdir('./tmp')
        except:
            pass
    def tearDown(self):
        try:
            os.unlink('./tmp/bar')
            os.rmdir('./tmp/foo')
            os.rmdir('./tmp')
        except:
            pass
    def runTest(self):
        arvados.util.mkdir_dash_p('./tmp/foo')
        with open('./tmp/bar', 'wb') as f:
            f.write('bar')
        self.assertRaises(OSError, arvados.util.mkdir_dash_p, './tmp/bar')
