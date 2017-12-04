#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function
from future import standard_library

import json
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
