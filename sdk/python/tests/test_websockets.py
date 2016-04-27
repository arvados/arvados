import arvados
import arvados.events
import arvados.errors
from datetime import datetime, timedelta, tzinfo
import logging
import logging.handlers
import mock
import Queue
import run_test_server
import StringIO
import tempfile
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
        try:
            if self.ws:
                self.ws.close()
        except Exception as e:
            print("Error in teardown: ", e)
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

    def test_subscribe_poll_retry(self):
        api_mock = mock.MagicMock()
        n = []
        def on_ev(ev):
            n.append(ev)

        error_mock = mock.MagicMock()
        error_mock.resp.status = 0
        error_mock._get_reason.return_value = "testing"
        api_mock.logs().list().execute.side_effect = (arvados.errors.ApiError(error_mock, ""),
                                                      {"items": [{"id": 1}], "items_available": 1},
                                                      arvados.errors.ApiError(error_mock, ""),
                                                      {"items": [{"id": 1}], "items_available": 1})
        pc = arvados.events.PollClient(api_mock, [], on_ev, 15, None)
        pc.start()
        while len(n) < 2:
            time.sleep(.1)
        pc.close()

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

    # Test websocket reconnection on (un)execpted close
    def _test_websocket_reconnect(self, close_unexpected):
        run_test_server.authorize_with('active')
        events = Queue.Queue(100)

        logstream = StringIO.StringIO()
        rootLogger = logging.getLogger()
        streamHandler = logging.StreamHandler(logstream)
        rootLogger.addHandler(streamHandler)

        filters = [['object_uuid', 'is_a', 'arvados#human']]
        filters.append(['created_at', '>=', self.localiso(self.TIME_PAST)])
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put_nowait,
            poll_fallback=False,
            last_log_id=None)
        self.assertIsInstance(self.ws, arvados.events.EventClient)
        self.assertEqual(200, events.get(True, 5)['status'])

        # create obj
        human = arvados.api('v1').humans().create(body={}).execute()

        # expect an event
        self.assertIn(human['uuid'], events.get(True, 5)['object_uuid'])
        with self.assertRaises(Queue.Empty):
            self.assertEqual(events.get(True, 2), None)

        # close (im)properly
        if close_unexpected:
            self.ws.close_connection()
        else:
            self.ws.close()

        # create one more obj
        human2 = arvados.api('v1').humans().create(body={}).execute()

        # (un)expect the object creation event
        if close_unexpected:
            log_object_uuids = []
            for i in range(0, 2):
                event = events.get(True, 5)
                if event.get('object_uuid') != None:
                    log_object_uuids.append(event['object_uuid'])
            with self.assertRaises(Queue.Empty):
                self.assertEqual(events.get(True, 2), None)
            self.assertNotIn(human['uuid'], log_object_uuids)
            self.assertIn(human2['uuid'], log_object_uuids)
        else:
            with self.assertRaises(Queue.Empty):
                self.assertEqual(events.get(True, 2), None)

        # verify log message to ensure that an (un)expected close
        log_messages = logstream.getvalue()
        closeLogFound = log_messages.find("Unexpected close. Reconnecting.")
        retryLogFound = log_messages.find("Error during websocket reconnect. Will retry")
        if close_unexpected:
            self.assertNotEqual(closeLogFound, -1)
        else:
            self.assertEqual(closeLogFound, -1)
        rootLogger.removeHandler(streamHandler)

    def test_websocket_reconnect_on_unexpected_close(self):
        self._test_websocket_reconnect(True)

    def test_websocket_no_reconnect_on_close_by_user(self):
        self._test_websocket_reconnect(False)

    # Test websocket reconnection retry
    @mock.patch('arvados.events._EventClient.connect')
    def test_websocket_reconnect_retry(self, event_client_connect):
        event_client_connect.side_effect = [None, Exception('EventClient.connect error'), None]

        logstream = StringIO.StringIO()
        rootLogger = logging.getLogger()
        streamHandler = logging.StreamHandler(logstream)
        rootLogger.addHandler(streamHandler)

        run_test_server.authorize_with('active')
        events = Queue.Queue(100)

        filters = [['object_uuid', 'is_a', 'arvados#human']]
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put_nowait,
            poll_fallback=False,
            last_log_id=None)
        self.assertIsInstance(self.ws, arvados.events.EventClient)

        # simulate improper close
        self.ws.on_closed()

        # verify log messages to ensure retry happened
        log_messages = logstream.getvalue()
        found = log_messages.find("Error 'EventClient.connect error' during websocket reconnect.")
        self.assertNotEqual(found, -1)
        rootLogger.removeHandler(streamHandler)
