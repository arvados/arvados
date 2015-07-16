import Queue
import run_test_server
import unittest
import arvados
import arvados.events
import mock
import threading
from datetime import datetime, timedelta

class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def setUp(self):
        self.ws = None

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(WebsocketTest, self).tearDown()

    def _test_subscribe(self, poll_fallback, expect_type, additional_filters=None):
        run_test_server.authorize_with('active')
        events = Queue.Queue(3)
        filters = [['object_uuid', 'is_a', 'arvados#human']]
        if additional_filters:
            filters = filters + additional_filters
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put, poll_fallback=poll_fallback)
        self.assertIsInstance(self.ws, expect_type)
        self.assertEqual(200, events.get(True, 10)['status'])
        human = arvados.api('v1').humans().create(body={}).execute()
        self.assertEqual(human['uuid'], events.get(True, 10)['object_uuid'])
        self.assertTrue(events.empty(), "got more events than expected")

    def test_subscribe_websocket(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient)

    def test_subscribe_websocket_with_start_time_today(self):
        now = datetime.today()
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
                additional_filters=[['created_at', '>', now.strftime('%Y-%m-%d')]])

    def test_subscribe_websocket_with_start_time_last_hour(self):
        lastHour = datetime.today() - timedelta(hours = 1)
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
                additional_filters=[['created_at', '>', lastHour.strftime('%Y-%m-%d %H:%M:%S')]])

    def test_subscribe_websocket_with_start_time_next_hour(self):
        nextHour = datetime.today() + timedelta(hours = 1)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=False, expect_type=arvados.events.EventClient,
                    additional_filters=[['created_at', '>', nextHour.strftime('%Y-%m-%d %H:%M:%S')]])

    def test_subscribe_websocket_with_start_time_tomorrow(self):
        tomorrow = datetime.today() + timedelta(hours = 24)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=False, expect_type=arvados.events.EventClient,
                    additional_filters=[['created_at', '>', tomorrow.strftime('%Y-%m-%d')]])

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=1, expect_type=arvados.events.PollClient)
