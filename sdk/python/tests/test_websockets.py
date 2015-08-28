import arvados
import arvados.events
from datetime import datetime, timedelta, tzinfo
import mock
import Queue
import run_test_server
import threading
import time
import unittest

class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    TIME_PAST = time.time()-3600
    TIME_FUTURE = time.time()+3600

    def setUp(self):
        self.ws = None

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(WebsocketTest, self).tearDown()
        run_test_server.reset()

    def _test_subscribe(self, poll_fallback, expect_type, start_time=None, expected=1):
        run_test_server.authorize_with('active')
        events = Queue.Queue(100)

        # Create ancestor before subscribing.
        # When listening with start_time in the past, this should also be retrieved.
        # However, when start_time is omitted in subscribe, this should not be fetched.
        ancestor = arvados.api('v1').humans().create(body={}).execute()

        filters = [['object_uuid', 'is_a', 'arvados#human']]
        if start_time:
            filters.append(['created_at', '>=', start_time])

        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put_nowait,
            poll_fallback=poll_fallback,
            last_log_id=(1 if start_time else None))
        self.assertIsInstance(self.ws, expect_type)
        self.assertEqual(200, events.get(True, 5)['status'])
        human = arvados.api('v1').humans().create(body={}).execute()

        log_object_uuids = []
        for i in range(0, expected):
            log_object_uuids.append(events.get(True, 5)['object_uuid'])

        if expected > 0:
            self.assertIn(human['uuid'], log_object_uuids)

        if expected > 1:
            self.assertIn(ancestor['uuid'], log_object_uuids)

        with self.assertRaises(Queue.Empty):
            # assertEqual just serves to show us what unexpected thing
            # comes out of the queue when the assertRaises fails; when
            # the test passes, this assertEqual doesn't get called.
            self.assertEqual(events.get(True, 2), None)

    def test_subscribe_websocket(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient, expected=1)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=0.25, expect_type=arvados.events.PollClient, expected=1)

    def test_subscribe_websocket_with_start_time_past(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
            start_time=self.localiso(self.TIME_PAST),
            expected=2)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll_with_start_time_past(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=0.25, expect_type=arvados.events.PollClient,
            start_time=self.localiso(self.TIME_PAST),
            expected=2)

    def test_subscribe_websocket_with_start_time_future(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
            start_time=self.localiso(self.TIME_FUTURE),
            expected=0)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll_with_start_time_future(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=0.25, expect_type=arvados.events.PollClient,
            start_time=self.localiso(self.TIME_FUTURE),
            expected=0)

    def test_subscribe_websocket_with_start_time_past_utc(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
            start_time=self.utciso(self.TIME_PAST),
            expected=2)

    def test_subscribe_websocket_with_start_time_future_utc(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient,
            start_time=self.utciso(self.TIME_FUTURE),
            expected=0)

    def utciso(self, t):
        return time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(t))

    def localiso(self, t):
        return time.strftime('%Y-%m-%dT%H:%M:%S', time.localtime(t)) + self.isotz(-time.timezone/60)

    def isotz(self, offset):
        """Convert minutes-east-of-UTC to ISO8601 time zone designator"""
        return '{:+03d}{:02d}'.format(offset/60, offset%60)
