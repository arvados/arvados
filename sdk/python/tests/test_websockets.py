import Queue
import run_test_server
import unittest
import arvados
import arvados.events
import mock
import threading

class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def setUp(self):
        self.ws = None

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(WebsocketTest, self).tearDown()

    def _test_subscribe(self, poll_fallback, expect_type):
        run_test_server.authorize_with('active')
        events = Queue.Queue(3)
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), [['object_uuid', 'is_a', 'arvados#human']],
            events.put, poll_fallback=poll_fallback)
        self.assertIsInstance(self.ws, expect_type)
        self.assertEqual(200, events.get(True, 10)['status'])
        human = arvados.api('v1').humans().create(body={}).execute()
        self.assertEqual(human['uuid'], events.get(True, 10)['object_uuid'])
        self.assertTrue(events.empty(), "got more events than expected")

    def test_subscribe_websocket(self):
        self._test_subscribe(
            poll_fallback=False, expect_type=arvados.events.EventClient)

    @mock.patch('arvados.events.EventClient.__init__')
    def test_subscribe_poll(self, event_client_constr):
        event_client_constr.side_effect = Exception('All is well')
        self._test_subscribe(
            poll_fallback=1, expect_type=arvados.events.PollClient)
