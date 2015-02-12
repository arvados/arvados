#!/usr/bin/env python

from __future__ import absolute_import, print_function

import threading
import time

import mock
import pykka

from . import pykka_timeout

no_sleep = mock.patch('time.sleep', lambda n: None)

def arvados_node_mock(node_num=99, job_uuid=None, age=0, **kwargs):
    if job_uuid is True:
        job_uuid = 'zzzzz-jjjjj-jobjobjobjobjob'
    crunch_worker_state = 'idle' if (job_uuid is None) else 'busy'
    node = {'uuid': 'zzzzz-yyyyy-{:015x}'.format(node_num),
            'created_at': '2014-01-01T01:02:03Z',
            'modified_at': time.strftime('%Y-%m-%dT%H:%M:%SZ',
                                         time.gmtime(time.time() - age)),
            'slot_number': node_num,
            'hostname': 'compute{}'.format(node_num),
            'domain': 'zzzzz.arvadosapi.com',
            'ip_address': ip_address_mock(node_num),
            'job_uuid': job_uuid,
            'crunch_worker_state': crunch_worker_state,
            'info': {'ping_secret': 'defaulttestsecret'}}
    node.update(kwargs)
    return node

def cloud_object_mock(name_id):
    # A very generic mock, useful for stubbing libcloud objects we
    # only search for and pass around, like locations, subnets, etc.
    cloud_object = mock.NonCallableMagicMock(['id', 'name'],
                                             name='cloud_object')
    cloud_object.id = str(name_id)
    cloud_object.name = cloud_object.id.upper()
    return cloud_object

def cloud_node_mock(node_num=99):
    node = mock.NonCallableMagicMock(
        ['id', 'name', 'state', 'public_ips', 'private_ips', 'driver', 'size',
         'image', 'extra'],
        name='cloud_node')
    node.id = str(node_num)
    node.name = node.id
    node.public_ips = []
    node.private_ips = [ip_address_mock(node_num)]
    return node

def ip_address_mock(last_octet):
    return '10.20.30.{}'.format(last_octet)

class MockShutdownTimer(object):
    def _set_state(self, is_open, next_opening):
        self.window_open = lambda: is_open
        self.next_opening = lambda: next_opening


class MockSize(object):
    def __init__(self, factor):
        self.id = 'z{}.test'.format(factor)
        self.name = self.id
        self.ram = 128 * factor
        self.disk = 100 * factor
        self.bandwidth = 16 * factor
        self.price = float(factor)
        self.extra = {}

    def __eq__(self, other):
        return self.id == other.id


class MockTimer(object):
    def __init__(self, deliver_immediately=True):
        self.deliver_immediately = deliver_immediately
        self.messages = []
        self.lock = threading.Lock()

    def deliver(self):
        with self.lock:
            to_deliver = self.messages
            self.messages = []
        for callback, args, kwargs in to_deliver:
            callback(*args, **kwargs)

    def schedule(self, want_time, callback, *args, **kwargs):
        with self.lock:
            self.messages.append((callback, args, kwargs))
        if self.deliver_immediately:
            self.deliver()


class ActorTestMixin(object):
    FUTURE_CLASS = pykka.ThreadingFuture
    TIMEOUT = pykka_timeout

    def tearDown(self):
        pykka.ActorRegistry.stop_all()

    def stop_proxy(self, proxy):
        return proxy.actor_ref.stop(timeout=self.TIMEOUT)

    def wait_for_assignment(self, proxy, attr_name, unassigned=None,
                            timeout=TIMEOUT):
        deadline = time.time() + timeout
        while True:
            loop_timeout = deadline - time.time()
            if loop_timeout <= 0:
                self.fail("actor did not assign {} in time".format(attr_name))
            result = getattr(proxy, attr_name).get(loop_timeout)
            if result is not unassigned:
                return result


class RemotePollLoopActorTestMixin(ActorTestMixin):
    def build_monitor(self, *args, **kwargs):
        self.timer = mock.MagicMock(name='timer_mock')
        self.client = mock.MagicMock(name='client_mock')
        self.subscriber = mock.Mock(name='subscriber_mock')
        self.monitor = self.TEST_CLASS.start(
            self.client, self.timer, *args, **kwargs).proxy()
