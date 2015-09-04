import mock
import unittest

class Mocker(unittest.TestCase):
    ACTIVE_TOKEN = '3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi'

    default_config = {
        'arvados_api_host': 'zzzzz.api_host.example',
        'virtual_machine_hostname': 'testvm2.shell',
    }
    default_request = {
        'client_host': '::1',
        'token': ACTIVE_TOKEN,
        'username': 'active',
    }
    default_response = {
        'links': {
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
        'users': {
            'uuid': 'zzzzz-tpzed-xurymjxw79nv3jz',
            'full_name': 'Active User',
        },
        'virtual_machines': {
            'items': [{
                'uuid': 'zzzzz-2x53u-382brsig8rp3065',
                'hostname': 'testvm2.shell',
            }],
            'items_available': 1,
        },
    }

    def setUp(self):
        self.config = self.default_config.copy()
        self.request = self.default_request.copy()
        self.response = self.default_response.copy()
        self.api_client = mock.MagicMock(name='api_client')
        self.api_client.users().current().execute.side_effect = lambda: self.response['users']
        self.api_client.virtual_machines().list().execute.side_effect = lambda: self.response['virtual_machines']
        self.api_client.links().list().execute.side_effect = lambda: self.response['links']
        patcher = mock.patch('arvados.api')
        self.api = patcher.start()
        self.addCleanup(patcher.stop)
        self.api.side_effect = [self.api_client]

        self.syslogged = []
        patcher = mock.patch('syslog.syslog')
        self.syslog = patcher.start()
        self.addCleanup(patcher.stop)
        self.syslog.side_effect = lambda s: self.syslogged.append(s)
