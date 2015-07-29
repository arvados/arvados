import Queue
import run_test_server
import unittest
import arvados
import arvados.events
import mock
import threading
from datetime import datetime, timedelta
import time

class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def setUp(self):
        self.ws = None

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(WebsocketTest, self).tearDown()

    def _test_subscribe(self, poll_fallback, expect_type, last_log_id=None, additional_filters=None, expected=1):
        run_test_server.authorize_with('active')
        events = Queue.Queue(100)

        # Create ancestor before subscribing.
        # When listening with start_time in the past, this should also be retrieved.
        # However, when start_time is omitted in subscribe, this should not be fetched.
        ancestor = arvados.api('v1').humans().create(body={}).execute()
        time.sleep(5)

        filters = [['object_uuid', 'is_a', 'arvados#human']]
        if additional_filters:
            filters = filters + additional_filters

        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put, poll_fallback=poll_fallback, last_log_id=last_log_id)
        self.assertIsInstance(self.ws, expect_type)
        self.assertEqual(200, events.get(True, 5)['status'])
        human = arvados.api('v1').humans().create(body={}).execute()

        if last_log_id == None or expected == 0:
            self.assertEqual(human['uuid'], events.get(True, 5)['object_uuid'])
            self.assertTrue(events.empty(), "got more events than expected")
        else:
            log_events = []
            for i in range(0, 20):
                try:
                    event = events.get(True, 5)
                    self.assertTrue(event['object_uuid'] is not None)
                    log_events.append(event['object_uuid'])
                except:
                    break;

            self.assertTrue(len(log_events)>1)
            self.assertTrue(human['uuid'] in log_events)
            self.assertTrue(ancestor['uuid'] in log_events)

    def test_subscribe_websocket(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=1, expect_type=arvados.events.PollClient)

    def test_subscribe_websocket_with_start_time_date_only(self):
        lastHour = datetime.today() - timedelta(hours = 1)
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient, last_log_id=1,
                additional_filters=[['created_at', '>=', lastHour.strftime('%Y-%m-%d')]])

    @mock.patch('arvados.events.EventClient.__init__')
    def test_poll_with_start_time_date_only(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        lastHour = datetime.today() - timedelta(hours = 1)
        self._test_subscribe(
            poll_fallback=1, expect_type=arvados.events.PollClient, last_log_id=1,
                additional_filters=[['created_at', '>=', lastHour.strftime('%Y-%m-%d')]])

    def test_subscribe_websocket_with_start_time_last_hour(self):
        lastHour = datetime.today() - timedelta(hours = 1)
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient, last_log_id=1,
                additional_filters=[['created_at', '>=', lastHour.strftime('%Y-%m-%d %H:%M:%S')]])

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll_with_start_time_last_hour(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        lastHour = datetime.today() - timedelta(hours = 1)
        self._test_subscribe(
            poll_fallback=1, expect_type=arvados.events.PollClient, last_log_id=1,
                additional_filters=[['created_at', '>=', lastHour.strftime('%Y-%m-%d %H:%M:%S')]])

    def test_subscribe_websocket_with_start_time_next_hour(self):
        nextHour = datetime.today() + timedelta(hours = 1)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=False, expect_type=arvados.events.EventClient, last_log_id=1,
                    additional_filters=[['created_at', '>=', nextHour.strftime('%Y-%m-%d %H:%M:%S')]], expected=0)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll_with_start_time_next_hour(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        nextHour = datetime.today() + timedelta(hours = 1)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=1, expect_type=arvados.events.PollClient, last_log_id=1,
                    additional_filters=[['created_at', '>=', nextHour.strftime('%Y-%m-%d %H:%M:%S')]], expected=0)

    def test_subscribe_websocket_with_start_time_tomorrow(self):
        tomorrow = datetime.today() + timedelta(hours = 24)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=False, expect_type=arvados.events.EventClient, last_log_id=1,
                    additional_filters=[['created_at', '>=', tomorrow.strftime('%Y-%m-%d')]], expected=0)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll_with_start_time_tomorrow(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        tomorrow = datetime.today() + timedelta(hours = 24)
        with self.assertRaises(Queue.Empty):
            self._test_subscribe(
                poll_fallback=1, expect_type=arvados.events.PollClient, last_log_id=1,
                    additional_filters=[['created_at', '>=', tomorrow.strftime('%Y-%m-%d')]], expected=0)
