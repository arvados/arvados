import os
if os.path.exists('/etc/pam.d/arvados-pam-test'):
    import pam
    import unittest

    ACTIVE_TOKEN = '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi'
    SPECTATOR_TOKEN = 'zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu'

    class IntegrationTest(unittest.TestCase):
        def setUp(self):
            self.p = pam.pam()

        def test_allow(self):
            self.assertTrue(self.p.authenticate('active', ACTIVE_TOKEN, service='arvados-pam-test'))

        def test_deny_service(self):
            self.assertFalse(self.p.authenticate('active', ACTIVE_TOKEN, service='login'))

        def test_deny_token(self):
            self.assertFalse(self.p.authenticate('active', 'bogustoken', service='arvados-pam-test'))

        def test_deny_permission(self):
            self.assertFalse(self.p.authenticate('spectator', SPECTATOR_TOKEN, service='arvados-pam-test'))
