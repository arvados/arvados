"""These tests assume we are running (in a docker container) with
arvados_pam configured and a test API server running.
"""
import pam
import unittest

# From services/api/test/fixtures/api_client_authorizations.yml
# because that file is not available during integration tests:
ACTIVE_TOKEN = '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi'
SPECTATOR_TOKEN = 'zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu'

class IntegrationTest(unittest.TestCase):
    def setUp(self):
        self.p = pam.pam()

    def test_allow(self):
        self.assertTrue(self.p.authenticate('active', ACTIVE_TOKEN, service='login'))

    def test_deny_bad_token(self):
        self.assertFalse(self.p.authenticate('active', 'thisisaverybadtoken', service='login'))

    def test_deny_empty_token(self):
        self.assertFalse(self.p.authenticate('active', '', service='login'))

    def test_deny_permission(self):
        self.assertFalse(self.p.authenticate('spectator', SPECTATOR_TOKEN, service='login'))
