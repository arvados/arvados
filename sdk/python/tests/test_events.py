from __future__ import absolute_import, print_function

import contextlib
import unittest

from . import arvados_testutil
import arvados.events
import mock

TEST_WS_URL = 'wss://[{}]/'.format(arvados_testutil.TEST_HOST)

class EventClientTestCase(unittest.TestCase):
    def setUp(self):
        self.recv_events = []

    def callback(self, event):
        self.recv_events.append(event)

    @contextlib.contextmanager
    def mocked_client(self, filters=None, callback=None, last_log_id=None):
        if filters is None:
            filters = []
        if callback is None:
            callback = self.callback
        with mock.patch('arvados.events._EventClient') as ws_mock:
            yield arvados.events.EventClient(TEST_WS_URL, filters, callback,
                                             last_log_id), ws_mock

    def test_client_init(self):
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            ws_mock.assert_called_with(TEST_WS_URL, [[]], client.on_event, None,
                                       client.on_closed)
            ws_mock().connect.assert_called()
            self.assertIs(ws_mock().daemon, True)

    def test_subscribe_calls_ws(self):
        ws_filter = ['kind', '=', 'arvados#test']
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            client.subscribe(ws_filter)
            ws_mock().subscribe.assert_called_with(ws_filter, None)

    def test_unsubscribe_calls_ws(self):
        ws_filter = ['kind', '=', 'arvados#test']
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            client.subscribe(ws_filter)
            client.unsubscribe(ws_filter)
            ws_mock().unsubscribe.assert_called_with(ws_filter)

    def test_close_calls_ws(self):
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            ws_mock().close.side_effect = lambda *args: client.on_closed()
            client.close()
            ws_mock().close.assert_called()
            # Check on_closed did not try to reconnect.
            ws_mock().connect.assert_called_once()

    def test_run_forever_calls_ws(self):
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            client.run_forever()
            ws_mock().run_forever.assert_called()

    def test_reconnect_rebuilds_and_reconnects_underlying_client(self):
        with self.mocked_client() as client_tuple:
            client, ws_mock = client_tuple
            client.on_closed()
            self.assertEqual(2, ws_mock.call_count)
            self.assertEqual(2, ws_mock().connect.call_count)
