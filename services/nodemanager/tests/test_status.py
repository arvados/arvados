#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function
from future import standard_library

import json
import mock
import random
import requests
import unittest

import arvnodeman.status as status
import arvnodeman.config as config


class TestServer(object):
    def __init__(self, management_token=None):
        self.mgmt_token = management_token

    def __enter__(self):
        cfg = config.NodeManagerConfig()
        cfg.set('Manage', 'port', '0')
        cfg.set('Manage', 'address', '127.0.0.1')
        if self.mgmt_token != None:
            cfg.set('Manage', 'ManagementToken', self.mgmt_token)
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

    def get_healthcheck_ping(self, auth_header=None):
        headers = {}
        if auth_header != None:
            headers['Authorization'] = auth_header
        return requests.get(self.srv_base+'/_health/ping', headers=headers)

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
            self.assertIn('Version', resp)
            self.assertIn('config_max_nodes', resp)

    def test_counters(self):
        with TestServer() as srv:
            resp = srv.get_status()
            # Test counters existance
            for counter in ['list_nodes_errors', 'create_node_errors',
                'destroy_node_errors', 'boot_failures', 'actor_exceptions']:
                self.assertIn(counter, resp)
            # Test counter increment
            for count in range(1, 3):
                status.tracker.counter_add('a_counter')
                resp = srv.get_status()
                self.assertEqual(count, resp['a_counter'])

    @mock.patch('time.time')
    def test_idle_times(self, time_mock):
        with TestServer() as srv:
            resp = srv.get_status()
            node_name = 'idle_compute{}'.format(random.randint(1, 1024))
            self.assertIn('idle_times', resp)
            # Test add an idle node
            time_mock.return_value = 10
            status.tracker.idle_in(node_name)
            time_mock.return_value += 10
            resp = srv.get_status()
            self.assertEqual(10, resp['idle_times'][node_name])
            # Test adding the same idle node a 2nd time
            time_mock.return_value += 10
            status.tracker.idle_in(node_name)
            time_mock.return_value += 10
            resp = srv.get_status()
            # Idle timestamp doesn't get reset if already exists
            self.assertEqual(30, resp['idle_times'][node_name])
            # Test remove idle node
            status.tracker.idle_out(node_name)
            resp = srv.get_status()
            self.assertNotIn(node_name, resp['idle_times'])


class StatusServerDisabled(unittest.TestCase):
    def test_config_disabled(self):
        cfg = config.NodeManagerConfig()
        cfg.set('Manage', 'port', '-1')
        cfg.set('Manage', 'address', '127.0.0.1')
        self.srv = status.Server(cfg)
        self.srv.start()
        self.assertFalse(self.srv.enabled)
        self.assertFalse(getattr(self.srv, '_thread', False))

class HealthcheckPing(unittest.TestCase):
    def test_ping_disabled(self):
        with TestServer() as srv:
            r = srv.get_healthcheck_ping()
            self.assertEqual(404, r.status_code)

    def test_ping_no_auth(self):
        with TestServer('configuredmanagementtoken') as srv:
            r = srv.get_healthcheck_ping()
            self.assertEqual(401, r.status_code)

    def test_ping_bad_auth_format(self):
        with TestServer('configuredmanagementtoken') as srv:
            r = srv.get_healthcheck_ping('noBearer')
            self.assertEqual(403, r.status_code)

    def test_ping_bad_auth_token(self):
        with TestServer('configuredmanagementtoken') as srv:
            r = srv.get_healthcheck_ping('Bearer badtoken')
            self.assertEqual(403, r.status_code)

    def test_ping_success(self):
        with TestServer('configuredmanagementtoken') as srv:
            r = srv.get_healthcheck_ping('Bearer configuredmanagementtoken')
            self.assertEqual(200, r.status_code)
            self.assertEqual('application/json', r.headers['content-type'])
            resp = r.json()
            self.assertEqual('{"health": "OK"}', json.dumps(resp))
