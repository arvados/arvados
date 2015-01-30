import run_test_server
import unittest
import arvados
import arvados.events
import threading

class EventTestBase(object):
    def on_event(self, ev):
        if self.state == 1:
            self.assertEqual(200, ev['status'])
            self.state = 2
            self.subscribed.set()
        elif self.state == 2:
            self.assertEqual(self.h[u'uuid'], ev[u'object_uuid'])
            self.state = 3
            self.done.set()
        elif self.state == 3:
            self.fail()

    def runTest(self):
        self.ws = None
        self.state = 1
        self.subscribed = threading.Event()
        self.done = threading.Event()

        run_test_server.authorize_with("admin")
        self.ws = arvados.events.subscribe(arvados.api('v1'), [['object_uuid', 'is_a', 'arvados#human']], self.on_event, poll_fallback=2)
        self.assertIsInstance(self.ws, self.WS_TYPE)
        self.subscribed.wait(10)
        self.h = arvados.api('v1').humans().create(body={}).execute()
        self.done.wait(10)
        self.assertEqual(3, self.state)

class WebsocketTest(run_test_server.TestCaseWithServers, EventTestBase):
    MAIN_SERVER = {'websockets': True}
    WS_TYPE = arvados.events.EventClient

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(WebsocketTest, self).tearDown()


class PollClientTest(run_test_server.TestCaseWithServers, EventTestBase):
    MAIN_SERVER = {}
    WS_TYPE = arvados.events.PollClient

    def tearDown(self):
        if self.ws:
            self.ws.close()
        super(PollClientTest, self).tearDown()
