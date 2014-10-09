#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time

import mock
import pykka

from . import pykka_timeout

no_sleep = mock.patch('time.sleep', lambda n: None)

def arvados_node_mock(node_num=99, job_uuid=None, age=0, **kwargs):
    if job_uuid is True:
        job_uuid = 'zzzzz-jjjjj-jobjobjobjobjob'
    slurm_state = 'idle' if (job_uuid is None) else 'alloc'
    node = {'uuid': 'zzzzz-yyyyy-12345abcde67890',
            'created_at': '2014-01-01T01:02:03Z',
            'modified_at': time.strftime('%Y-%m-%dT%H:%M:%SZ',
                                         time.gmtime(time.time() - age)),
            'hostname': 'compute{}'.format(node_num),
            'domain': 'zzzzz.arvadosapi.com',
            'ip_address': ip_address_mock(node_num),
            'job_uuid': job_uuid,
            'info': {'slurm_state': slurm_state}}
    node.update(kwargs)
    return node

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
    def schedule(self, want_time, callback, *args, **kwargs):
        return callback(*args, **kwargs)


class ActorTestMixin(object):
    FUTURE_CLASS = pykka.ThreadingFuture
    TIMEOUT = pykka_timeout

    def tearDown(self):
        pykka.ActorRegistry.stop_all()

    def wait_for_call(self, mock_func, timeout=TIMEOUT):
        deadline = time.time() + timeout
        while (not mock_func.called) and (time.time() < deadline):
            time.sleep(.1)
        self.assertTrue(mock_func.called, "{} not called".format(mock_func))


class RemotePollLoopActorTestMixin(ActorTestMixin):
    def build_monitor(self, *args, **kwargs):
        self.timer = mock.MagicMock(name='timer_mock')
        self.client = mock.MagicMock(name='client_mock')
        self.subscriber = mock.Mock(name='subscriber_mock')
        self.monitor = self.TEST_CLASS.start(
            self.client, self.timer, *args, **kwargs).proxy()
