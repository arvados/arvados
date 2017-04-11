from __future__ import print_function
from __future__ import absolute_import
from __future__ import division
from future import standard_library
standard_library.install_aliases()
from builtins import range
from builtins import object
import logging
import mock
import queue
import sys
import threading
import time
import unittest

import arvados
from . import arvados_testutil as tutil
from . import run_test_server


class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    TIME_PAST = time.time()-3600
    TIME_FUTURE = time.time()+3600
    MOCK_WS_URL = 'wss://[{}]/'.format(tutil.TEST_HOST)

    TEST_TIMEOUT = 10.0

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
        events = queue.Queue(100)

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

        want_uuids = []
        if expected > 0:
            want_uuids.append(human['uuid'])
        if expected > 1:
            want_uuids.append(ancestor['uuid'])
        log_object_uuids = []
        while set(want_uuids) - set(log_object_uuids):
            log_object_uuids.append(events.get(True, 5)['object_uuid'])

        if expected < 2:
            with self.assertRaises(queue.Empty):
                # assertEqual just serves to show us what unexpected
                # thing comes out of the queue when the assertRaises
                # fails; when the test passes, this assertEqual
                # doesn't get called.
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
        api_mock.logs().list().execute.side_effect = (
            arvados.errors.ApiError(error_mock, b""),
            {"items": [{"id": 1}], "items_available": 1},
            arvados.errors.ApiError(error_mock, b""),
            {"items": [{"id": 1}], "items_available": 1},
        )
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
        return time.strftime('%Y-%m-%dT%H:%M:%S', time.localtime(t)) + self.isotz(-time.timezone//60)

    def isotz(self, offset):
        """Convert minutes-east-of-UTC to RFC3339- and ISO-compatible time zone designator"""
        return '{:+03d}:{:02d}'.format(offset//60, offset%60)

    # Test websocket reconnection on (un)execpted close
    def _test_websocket_reconnect(self, close_unexpected):
        run_test_server.authorize_with('active')
        events = queue.Queue(100)

        logstream = tutil.StringIO()
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
        with self.assertRaises(queue.Empty):
            self.assertEqual(events.get(True, 2), None)

        # close (im)properly
        if close_unexpected:
            self.ws.ec.close_connection()
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
            with self.assertRaises(queue.Empty):
                self.assertEqual(events.get(True, 2), None)
            self.assertNotIn(human['uuid'], log_object_uuids)
            self.assertIn(human2['uuid'], log_object_uuids)
        else:
            with self.assertRaises(queue.Empty):
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

        logstream = tutil.StringIO()
        rootLogger = logging.getLogger()
        streamHandler = logging.StreamHandler(logstream)
        rootLogger.addHandler(streamHandler)

        run_test_server.authorize_with('active')
        events = queue.Queue(100)

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

    @mock.patch('arvados.events._EventClient')
    def test_subscribe_method(self, websocket_client):
        filters = [['object_uuid', 'is_a', 'arvados#human']]
        client = arvados.events.EventClient(
            self.MOCK_WS_URL, [], lambda event: None, None)
        client.subscribe(filters[:], 99)
        websocket_client().subscribe.assert_called_with(filters, 99)

    @mock.patch('arvados.events._EventClient')
    def test_unsubscribe(self, websocket_client):
        filters = [['object_uuid', 'is_a', 'arvados#human']]
        client = arvados.events.EventClient(
            self.MOCK_WS_URL, filters[:], lambda event: None, None)
        client.unsubscribe(filters[:])
        websocket_client().unsubscribe.assert_called_with(filters)

    @mock.patch('arvados.events._EventClient')
    def test_run_forever_survives_reconnects(self, websocket_client):
        connected = threading.Event()
        websocket_client().connect.side_effect = connected.set
        client = arvados.events.EventClient(
            self.MOCK_WS_URL, [], lambda event: None, None)
        forever_thread = threading.Thread(target=client.run_forever)
        forever_thread.start()
        # Simulate an unexpected disconnect, and wait for reconnect.
        close_thread = threading.Thread(target=client.on_closed)
        close_thread.start()
        self.assertTrue(connected.wait(timeout=self.TEST_TIMEOUT))
        close_thread.join()
        run_forever_alive = forever_thread.is_alive()
        client.close()
        forever_thread.join()
        self.assertTrue(run_forever_alive)
        self.assertEqual(2, websocket_client().connect.call_count)


class PollClientTestCase(unittest.TestCase):
    TEST_TIMEOUT = 10.0

    class MockLogs(object):

        def __init__(self):
            self.logs = []
            self.lock = threading.Lock()
            self.api_called = threading.Event()

        def add(self, log):
            with self.lock:
                self.logs.append(log)

        def return_list(self, num_retries=None):
            self.api_called.set()
            args, kwargs = self.list_func.call_args_list[-1]
            filters = kwargs.get('filters', [])
            if not any(True for f in filters if f[0] == 'id' and f[1] == '>'):
                # No 'id' filter was given -- this must be the probe
                # to determine the most recent id.
                return {'items': [{'id': 1}], 'items_available': 1}
            with self.lock:
                retval = self.logs
                self.logs = []
            return {'items': retval, 'items_available': len(retval)}

    def setUp(self):
        self.logs = self.MockLogs()
        self.arv = mock.MagicMock(name='arvados.api()')
        self.arv.logs().list().execute.side_effect = self.logs.return_list
        # our MockLogs object's "execute" stub will need to inspect
        # the call history to determine X in
        # ....logs().list(filters=X).execute():
        self.logs.list_func = self.arv.logs().list
        self.status_ok = threading.Event()
        self.event_received = threading.Event()
        self.recv_events = []

    def tearDown(self):
        if hasattr(self, 'client'):
            self.client.close(timeout=None)

    def callback(self, event):
        if event.get('status') == 200:
            self.status_ok.set()
        else:
            self.recv_events.append(event)
            self.event_received.set()

    def build_client(self, filters=None, callback=None, last_log_id=None, poll_time=99):
        if filters is None:
            filters = []
        if callback is None:
            callback = self.callback
        self.client = arvados.events.PollClient(
            self.arv, filters, callback, poll_time, last_log_id)

    def was_filter_used(self, target):
        return any(target in call[-1].get('filters', [])
                   for call in self.arv.logs().list.call_args_list)

    def test_callback(self):
        test_log = {'id': 12345, 'testkey': 'testtext'}
        self.logs.add({'id': 123})
        self.build_client(poll_time=.01)
        self.client.start()
        self.assertTrue(self.status_ok.wait(self.TEST_TIMEOUT))
        self.assertTrue(self.event_received.wait(self.TEST_TIMEOUT))
        self.event_received.clear()
        self.logs.add(test_log.copy())
        self.assertTrue(self.event_received.wait(self.TEST_TIMEOUT))
        self.assertIn(test_log, self.recv_events)

    def test_subscribe(self):
        client_filter = ['kind', '=', 'arvados#test']
        self.build_client()
        self.client.unsubscribe([])
        self.client.subscribe([client_filter[:]])
        self.client.start()
        self.assertTrue(self.status_ok.wait(self.TEST_TIMEOUT))
        self.assertTrue(self.logs.api_called.wait(self.TEST_TIMEOUT))
        self.assertTrue(self.was_filter_used(client_filter))

    def test_unsubscribe(self):
        should_filter = ['foo', '=', 'foo']
        should_not_filter = ['foo', '=', 'bar']
        self.build_client(poll_time=0.01)
        self.client.unsubscribe([])
        self.client.subscribe([should_not_filter[:]])
        self.client.subscribe([should_filter[:]])
        self.client.unsubscribe([should_not_filter[:]])
        self.client.start()
        self.logs.add({'id': 123})
        self.assertTrue(self.status_ok.wait(self.TEST_TIMEOUT))
        self.assertTrue(self.event_received.wait(self.TEST_TIMEOUT))
        self.assertTrue(self.was_filter_used(should_filter))
        self.assertFalse(self.was_filter_used(should_not_filter))

    def test_run_forever(self):
        self.build_client()
        self.client.start()
        forever_thread = threading.Thread(target=self.client.run_forever)
        forever_thread.start()
        self.assertTrue(self.status_ok.wait(self.TEST_TIMEOUT))
        self.assertTrue(forever_thread.is_alive())
        self.client.close()
        forever_thread.join()
        del self.client
