# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import json
import logging
import queue
import sys
import threading
import time
import unittest

from unittest import mock

import websockets.exceptions as ws_exc

import arvados
from . import arvados_testutil as tutil
from . import run_test_server

class FakeWebsocketClient:
    """Fake self-contained version of websockets.sync.client.ClientConnection

    This provides enough of the API to test EventClient. It loosely mimics
    the Arvados WebSocket API by acknowledging subscribe messages. You can use
    `mock_wrapper` to test calls. You can set `_check_lock` to test that the
    given lock is acquired before `send` is called.
    """

    def __init__(self):
        self._check_lock = None
        self._closed = threading.Event()
        self._messages = queue.Queue()

    def mock_wrapper(self):
        wrapper = mock.Mock(wraps=self)
        wrapper.__iter__ = lambda _: self.__iter__()
        return wrapper

    def __iter__(self):
        while True:
            msg = self._messages.get()
            self._messages.task_done()
            if isinstance(msg, Exception):
                raise msg
            else:
                yield msg

    def close(self, code=1000, reason=''):
        if not self._closed.is_set():
            self._closed.set()
            self.force_disconnect()

    def force_disconnect(self):
        self._messages.put(ws_exc.ConnectionClosed(None, None))

    def send(self, msg):
        if self._check_lock is not None and self._check_lock.acquire(blocking=False):
            self._check_lock.release()
            raise AssertionError(f"called ws_client.send() without lock")
        elif self._closed.is_set():
            raise ws_exc.ConnectionClosed(None, None)
        try:
            msg = json.loads(msg)
        except ValueError:
            status = 400
        else:
            status = 200
        self._messages.put(json.dumps({'status': status}))


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
        ancestor = arvados.api('v1').collections().create(body={}).execute()

        filters = [['object_uuid', 'is_a', 'arvados#collection']]
        if start_time:
            filters.append(['created_at', '>=', start_time])

        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put_nowait,
            poll_fallback=poll_fallback,
            last_log_id=(1 if start_time else None))
        self.assertIsInstance(self.ws, expect_type)
        self.assertEqual(200, events.get(True, 5)['status'])

        if hasattr(self.ws, '_skip_old_events'):
            # Avoid race by waiting for the first "find ID threshold"
            # poll to finish.
            deadline = time.time() + 10
            while not self.ws._skip_old_events:
                self.assertLess(time.time(), deadline)
                time.sleep(0.1)
        collection = arvados.api('v1').collections().create(body={}).execute()

        want_uuids = []
        if expected > 0:
            want_uuids.append(collection['uuid'])
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

    # Test websocket reconnection on (un)expected close
    def _test_websocket_reconnect(self, close_unexpected):
        run_test_server.authorize_with('active')
        events = queue.Queue(100)

        logstream = tutil.StringIO()
        rootLogger = logging.getLogger()
        streamHandler = logging.StreamHandler(logstream)
        rootLogger.addHandler(streamHandler)

        filters = [['object_uuid', 'is_a', 'arvados#collection']]
        filters.append(['created_at', '>=', self.localiso(self.TIME_PAST)])
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), filters,
            events.put_nowait,
            poll_fallback=False,
            last_log_id=None)
        self.assertIsInstance(self.ws, arvados.events.EventClient)
        self.assertEqual(200, events.get(True, 5)['status'])

        # create obj
        collection = arvados.api('v1').collections().create(body={}).execute()

        # expect an event
        self.assertIn(collection['uuid'], events.get(True, 5)['object_uuid'])
        with self.assertRaises(queue.Empty):
            self.assertEqual(events.get(True, 2), None)

        # close (im)properly
        if close_unexpected:
            self.ws._client.close()
        else:
            self.ws.close()

        # create one more obj
        collection2 = arvados.api('v1').collections().create(body={}).execute()

        # (un)expect the object creation event
        if close_unexpected:
            log_object_uuids = []
            for i in range(0, 2):
                event = events.get(True, 5)
                if event.get('object_uuid') != None:
                    log_object_uuids.append(event['object_uuid'])
            with self.assertRaises(queue.Empty):
                self.assertEqual(events.get(True, 2), None)
            self.assertNotIn(collection['uuid'], log_object_uuids)
            self.assertIn(collection2['uuid'], log_object_uuids)
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
    @mock.patch('arvados.events.ws_client.connect')
    def test_websocket_reconnect_retry(self, ws_conn):
        logstream = tutil.StringIO()
        rootLogger = logging.getLogger()
        streamHandler = logging.StreamHandler(logstream)
        rootLogger.addHandler(streamHandler)
        try:
            msg_event, wss_client, self.ws = self.fake_client(ws_conn)
            self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for setup callback")
            msg_event.clear()
            ws_conn.side_effect = [Exception('EventClient.connect error'), wss_client]
            wss_client.force_disconnect()
            self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for reconnect callback")
            # verify log messages to ensure retry happened
            self.assertIn("Error 'EventClient.connect error' during websocket reconnect.", logstream.getvalue())
            self.assertEqual(ws_conn.call_count, 3)
        finally:
            rootLogger.removeHandler(streamHandler)

    @mock.patch('arvados.events.ws_client.connect')
    def test_run_forever_survives_reconnects(self, websocket_client):
        client = arvados.events.EventClient(
            self.MOCK_WS_URL, [], lambda event: None, None)
        forever_thread = threading.Thread(target=client.run_forever)
        forever_thread.start()
        # Simulate an unexpected disconnect, and wait for reconnect.
        try:
            client.on_closed()
            self.assertTrue(forever_thread.is_alive())
            self.assertEqual(2, websocket_client.call_count)
        finally:
            client.close()
            forever_thread.join()

    @staticmethod
    def fake_client(conn_patch, filters=None, url=MOCK_WS_URL):
        """Set up EventClient test infrastructure

        Given a patch of `arvados.events.ws_client.connect`,
        this returns a 3-tuple:

        * `msg_event` is a `threading.Event` that is set as the test client
          event callback. You can wait for this event to confirm that a
          sent message has been acknowledged and processed.

        * `mock_client` is a `mock.Mock` wrapper around `FakeWebsocketClient`.
          Use this to assert `EventClient` calls the right methods. It tests
          that `EventClient` acquires a lock before calling `send`.

        * `client` is the `EventClient` that uses `mock_client` under the hood
          that you exercise methods of.

        Other arguments are passed to initialize `EventClient`.
        """
        msg_event = threading.Event()
        fake_client = FakeWebsocketClient()
        mock_client = fake_client.mock_wrapper()
        conn_patch.return_value = mock_client
        client = arvados.events.EventClient(url, filters, lambda _: msg_event.set())
        fake_client._check_lock = client._subscribe_lock
        return msg_event, mock_client, client

    @mock.patch('arvados.events.ws_client.connect')
    def test_subscribe_locking(self, ws_conn):
        f = [['created_at', '>=', '2023-12-01T00:00:00.000Z']]
        msg_event, wss_client, self.ws = self.fake_client(ws_conn)
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for setup callback")
        msg_event.clear()
        wss_client.send.reset_mock()
        self.ws.subscribe(f)
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for subscribe callback")
        wss_client.send.assert_called()
        (msg,), _ = wss_client.send.call_args
        self.assertEqual(
            json.loads(msg),
            {'method': 'subscribe', 'filters': f},
        )

    @mock.patch('arvados.events.ws_client.connect')
    def test_unsubscribe_locking(self, ws_conn):
        f = [['created_at', '>=', '2023-12-01T01:00:00.000Z']]
        msg_event, wss_client, self.ws = self.fake_client(ws_conn, f)
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for setup callback")
        msg_event.clear()
        wss_client.send.reset_mock()
        self.ws.unsubscribe(f)
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for unsubscribe callback")
        wss_client.send.assert_called()
        (msg,), _ = wss_client.send.call_args
        self.assertEqual(
            json.loads(msg),
            {'method': 'unsubscribe', 'filters': f},
        )

    @mock.patch('arvados.events.ws_client.connect')
    def test_resubscribe_locking(self, ws_conn):
        f = [['created_at', '>=', '2023-12-01T02:00:00.000Z']]
        msg_event, wss_client, self.ws = self.fake_client(ws_conn, f)
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for setup callback")
        msg_event.clear()
        wss_client.send.reset_mock()
        wss_client.force_disconnect()
        self.assertTrue(msg_event.wait(timeout=1), "timed out waiting for resubscribe callback")
        wss_client.send.assert_called()
        (msg,), _ = wss_client.send.call_args
        self.assertEqual(
            json.loads(msg),
            {'method': 'subscribe', 'filters': f},
        )


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
