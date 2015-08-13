import arvados_pam
import re
from . import mocker

class AuthEventTest(mocker.Mocker):
    def attempt(self):
        return arvados_pam.auth_event.AuthEvent(config=self.config, service='test_service', **self.request).can_login()

    def test_success(self):
        self.assertTrue(self.attempt())

        self.api_client.virtual_machines().list.assert_called_with(
            filters=[['hostname','=',self.config['virtual_machine_hostname']]])
        self.api.assert_called_with(
            'v1',
            host=self.config['arvados_api_host'], token=self.request['token'],
            insecure=False,
            cache=False)
        self.assertEqual(1, len(self.syslogged))
        for i in ['test_service',
                  self.request['username'],
                  self.config['arvados_api_host'],
                  self.response['virtual_machines']['items'][0]['uuid']]:
            self.assertRegexpMatches(self.syslogged[0], re.escape(i))
        self.assertRegexpMatches(self.syslogged[0], re.escape(self.request['token'][0:15]), 'token prefix not logged')
        self.assertNotRegexpMatches(self.syslogged[0], re.escape(self.request['token'][15:30]), 'too much token logged')

    def test_fail_vm_lookup(self):
        self.api_client.virtual_machines().list().execute.side_effect = Exception("Test-induced failure")
        self.assertFalse(self.attempt())
        self.assertRegexpMatches(self.syslogged[0], 'Test-induced failure')

    def test_vm_hostname_not_found(self):
        self.response['virtual_machines'] = {
            'items': [],
            'items_available': 0,
        }
        self.assertFalse(self.attempt())

    def test_vm_hostname_ambiguous(self):
        self.response['virtual_machines'] = {
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
        self.response['virtual_machines'] = {
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
        self.api_client.users().current().execute.side_effect = Exception("Test-induced failure")
        self.assertFalse(self.attempt())

    def test_fail_permission_check(self):
        self.api_client.links().list().execute.side_effect = Exception("Test-induced failure")
        self.assertFalse(self.attempt())

    def test_no_login_permission(self):
        self.response['links'] = {
            'items': [],
        }
        self.assertFalse(self.attempt())

    def test_server_ignores_permission_filters(self):
        self.response['links'] = {
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
