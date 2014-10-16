import run_test_server
import unittest
import arvados
import arvados.events
import time
import threading

class WebsocketTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {'websockets': True}

    def on_event(self, ev):
        if self.state == 1:
            self.assertEqual(200, ev['status'])
            self.state = 2
        elif self.state == 2:
            self.assertEqual(self.h[u'uuid'], ev[u'object_uuid'])
            self.state = 3
            self.done.set()
        elif self.state == 3:
            self.fail()

    def runTest(self):
        self.state = 1
        self.done = threading.Event()

        run_test_server.authorize_with("admin")
        api = arvados.api('v1', cache=False)
        ws = arvados.events.subscribe(api, [['object_uuid', 'is_a', 'arvados#human']], self.on_event)
        time.sleep(1)
        self.h = api.humans().create(body={}).execute()
        self.done.wait(10)
        self.assertEqual(3, self.state)
        ws.close()

class PollClientTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def on_event(self, ev):
        if self.state == 1:
            self.assertEqual(200, ev['status'])
            self.state = 2
        elif self.state == 2:
            self.assertEqual(self.h[u'uuid'], ev[u'object_uuid'])
            self.state = 3
            self.done.set()
        elif self.state == 3:
            self.fail()

    def runTest(self):
        self.state = 1
        self.done = threading.Event()

        run_test_server.authorize_with("admin")
        api = arvados.api('v1', cache=False)
        ws = arvados.events.subscribe(api, [['object_uuid', 'is_a', 'arvados#human']], self.on_event, poll_fallback=2)
        time.sleep(1)
        self.h = api.humans().create(body={}).execute()
        self.done.wait(10)
        self.assertEqual(3, self.state)
        ws.close()
