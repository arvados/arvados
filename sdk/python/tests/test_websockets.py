import Queue
import run_test_server
import unittest
import arvados
import arvados.events
import threading

class EventTestBase(object):
    def runTest(self):
        run_test_server.authorize_with("admin")
        events = Queue.Queue(3)
        self.ws = arvados.events.subscribe(
            arvados.api('v1'), [['object_uuid', 'is_a', 'arvados#human']],
            events.put, poll_fallback=2)
        self.assertIsInstance(self.ws, self.WS_TYPE)
        self.assertEqual(200, events.get(True, 10)['status'])
        human = arvados.api('v1').humans().create(body={}).execute()
        self.assertEqual(human['uuid'], events.get(True, 10)['object_uuid'])
        self.assertTrue(events.empty(), "got more events than expected")

    def tearDown(self):
        try:
            self.ws.close()
        except AttributeError:
            pass
        super(EventTestBase, self).tearDown()


class WebsocketTest(EventTestBase, run_test_server.TestCaseWithServers):
    MAIN_SERVER = {'websockets': True}
    WS_TYPE = arvados.events.EventClient


class PollClientTest(EventTestBase, run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    WS_TYPE = arvados.events.PollClient
