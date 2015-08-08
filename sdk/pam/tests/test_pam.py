#!/usr/bin/env python

import arvados
import arvados_pam
import mock
import re
import StringIO
import unittest

class ConfigTest(unittest.TestCase):
    def test_ok_config(self):
        self.assertConfig(
            """servicename:
              ARVADOS_API_HOST: xyzzy.example
              virtual_machine_hostname: foo.shell
            """, 'servicename', 'xyzzy.example', 'foo.shell')

    @mock.patch('arvados_pam.config_file')
    def assertConfig(self, txt, svcname, apihost, shellhost, config_file):
        configfake = StringIO.StringIO(txt)
        config_file.side_effect = [configfake]
        c = arvados_pam.config()
        self.assertEqual(apihost, c[svcname]['ARVADOS_API_HOST'])
        self.assertEqual(shellhost, c[svcname]['virtual_machine_hostname'])

class AuthTest(unittest.TestCase):

    default_config = {
        'test_service': {
            'ARVADOS_API_HOST': 'zzzzz.api_host.example',
            'virtual_machine_hostname': 'testvm2.shell',
        }
    }

    default_request = {
        'client_host': '::1',
        'token': '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi',
        'username': 'active',
    }

    default_response = {
        'links': lambda: {
            'items': [{
                'uuid': 'zzzzz-o0j2j-rah2ya1ohx9xaev',
                'tail_uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
                'head_uuid': 'zzzzz-2x53u-382brsig8rp3065',
                'link_class': 'permission',
                'name': 'can_login',
                'properties': {
                    'username': 'active',
                },
            }],
        },
        'users': lambda: {
            'uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
            'full_name': 'Active User',
        },
        'virtual_machines': lambda: {
            'items': [{
                'uuid': 'zzzzz-2x53u-382brsig8rp3065',
                'hostname': 'testvm2.shell',
            }],
            'items_available': 1,
        },
    }

    def attempt(self):
        return arvados_pam.AuthEvent(config=self.config, service='test_service', **self.request).can_login()

    def test_success(self):
        self.assertTrue(self.attempt())

        cfg = self.config['test_service']
        self.api_client.virtual_machines().list.assert_called_with(
            filters=[['hostname','=',cfg['virtual_machine_hostname']]])
        self.api.assert_called_with(
            'v1', host=cfg['ARVADOS_API_HOST'], token=self.request['token'], cache=None)
        self.assertEqual(1, len(self.syslogged))
        for i in ['test_service',
                  self.request['username'],
                  self.config['test_service']['ARVADOS_API_HOST'],
                  self.response['virtual_machines']()['items'][0]['uuid']]:
            self.assertRegexpMatches(self.syslogged[0], re.escape(i))
        self.assertRegexpMatches(self.syslogged[0], re.escape(self.request['token'][0:15]), 'token prefix not logged')
        self.assertNotRegexpMatches(self.syslogged[0], re.escape(self.request['token'][15:30]), 'too much token logged')

    def test_fail_vm_lookup(self):
        self.response['virtual_machines'] = self._raise
        self.assertFalse(self.attempt())
        self.assertRegexpMatches(self.syslogged[0], 'Test-induced failure')

    def test_vm_hostname_not_found(self):
        self.response['virtual_machines'] = lambda: {
            'items': [],
            'items_available': 0,
        }
        self.assertFalse(self.attempt())

    def test_vm_hostname_ambiguous(self):
        self.response['virtual_machines'] = lambda: {
            'items': [
                {
                    'uuid': 'zzzzz-2x53u-382brsig8rp3065',
                    'hostname': 'testvm2.shell',
                },
                {
                    'uuid': 'zzzzz-2x53u-382brsig8rp3065',
                    'hostname': 'testvm2.shell',
                },
            ],
            'items_available': 2,
        }
        self.assertFalse(self.attempt())

    def test_server_ignores_vm_filters(self):
        self.response['virtual_machines'] = lambda: {
            'items': [
                {
                    'uuid': 'zzzzz-2x53u-382brsig8rp3065',
                    'hostname': 'testvm22.shell', # <-----
                },
            ],
            'items_available': 1,
        }
        self.assertFalse(self.attempt())

    def test_fail_user_lookup(self):
        self.response['users'] = self._raise
        self.assertFalse(self.attempt())

    def test_fail_permission_check(self):
        self.response['links'] = self._raise
        self.assertFalse(self.attempt())

    def test_no_login_permission(self):
        self.response['links'] = lambda: {
            'items': [],
        }
        self.assertFalse(self.attempt())

    def test_server_ignores_permission_filters(self):
        self.response['links'] = lambda: {
            'items': [{
                'uuid': 'zzzzz-o0j2j-rah2ya1ohx9xaev',
                'tail_uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
                'head_uuid': 'zzzzz-2x53u-382brsig8rp3065',
                'link_class': 'permission',
                'name': 'CANT_login', # <-----
                'properties': {
                    'username': 'active',
                },
            }],
        }
        self.assertFalse(self.attempt())

    def setUp(self):
        self.config = self.default_config.copy()
        self.request = self.default_request.copy()
        self.response = self.default_response.copy()
        self.api_client = mock.MagicMock(name='api_client')
        self.api_client.users().current().execute.side_effect = lambda: self.response['users']()
        self.api_client.virtual_machines().list().execute.side_effect = lambda: self.response['virtual_machines']()
        self.api_client.links().list().execute.side_effect = lambda: self.response['links']()
        patcher = mock.patch('arvados.api')
        self.api = patcher.start()
        self.addCleanup(patcher.stop)
        self.api.side_effect = [self.api_client]

        self.syslogged = []
        patcher = mock.patch('syslog.syslog')
        self.syslog = patcher.start()
        self.addCleanup(patcher.stop)
        self.syslog.side_effect = lambda s: self.syslogged.append(s)

    def _raise(self, exception=Exception("Test-induced failure"), *args, **kwargs):
        raise exception
