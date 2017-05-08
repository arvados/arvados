import os
import subprocess
import unittest

import arvados

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
            f.write(b'bar')
        self.assertRaises(OSError, arvados.util.mkdir_dash_p, './tmp/bar')


class RunCommandTestCase(unittest.TestCase):
    def test_success(self):
        stdout, stderr = arvados.util.run_command(['echo', 'test'],
                                                  stderr=subprocess.PIPE)
        self.assertEqual("test\n".encode(), stdout)
        self.assertEqual("".encode(), stderr)

    def test_failure(self):
        with self.assertRaises(arvados.errors.CommandFailedError):
            arvados.util.run_command(['false'])
