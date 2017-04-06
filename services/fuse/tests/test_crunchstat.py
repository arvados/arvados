import subprocess

from integration_test import IntegrationTest


class CrunchstatTest(IntegrationTest):
    def test_crunchstat(self):
        output = subprocess.check_output(
            ['./bin/arv-mount',
             '--crunchstat-interval', '1',
             self.mnt,
             '--exec', 'echo', 'ok'])
        self.assertEqual("ok\n", output)
