#!/usr/bin/env python

from __future__ import absolute_import, print_function
from future import standard_library

import requests
import unittest

import arvnodeman.status as status
import arvnodeman.config as config


class TestServer(object):
    def __enter__(self):
        cfg = config.NodeManagerConfig()
        cfg.set('Manage', 'port', '0')
        cfg.set('Manage', 'address', '127.0.0.1')
        self.srv = status.Server(cfg)
        self.srv.start()
        addr, port = self.srv.server_address
        self.srv_base = 'http://127.0.0.1:'+str(port)
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.srv.shutdown()

    def get_status_response(self):
        return requests.get(self.srv_base+'/status.json')

    def get_status(self):
        return self.get_status_response().json()


class StatusServerUpdates(unittest.TestCase):
    def test_updates(self):
        with TestServer() as srv:
            for n in [1, 2, 3]:
                status.tracker.update({'nodes_'+str(n): n})
                r = srv.get_status_response()
                self.assertEqual(200, r.status_code)
                self.assertEqual('application/json', r.headers['content-type'])
                resp = r.json()
                self.assertEqual(n, resp['nodes_'+str(n)])
            self.assertEqual(1, resp['nodes_1'])


class StatusServerDisabled(unittest.TestCase):
    def test_config_disabled(self):
        cfg = config.NodeManagerConfig()
        cfg.set('Manage', 'port', '-1')
        cfg.set('Manage', 'address', '127.0.0.1')
        self.srv = status.Server(cfg)
        self.srv.start()
        self.assertFalse(self.srv.enabled)
        self.assertFalse(getattr(self.srv, '_thread', False))
