#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import io
import logging
import unittest

import arvnodeman.computenode.dispatch as dispatch
import arvnodeman.computenode.dispatch.slurm as slurm_dispatch
import arvnodeman.config as nmconfig

class NodeManagerConfigTestCase(unittest.TestCase):
    TEST_CONFIG = u"""
[Cloud]
provider = dummy
shutdown_windows = 52, 6, 2

[Cloud Credentials]
creds = dummy_creds

[Cloud List]
[Cloud Create]

[Size 1]
cores = 1
price = 0.8

[Size 1.preemptible]
instance_type = 1
preemptible = true
cores = 1
price = 0.8

[Logging]
file = /dev/null
level = DEBUG
testlogger = INFO
"""

    def load_config(self, config=None, config_str=None):
        if config is None:
            config = nmconfig.NodeManagerConfig()
        if config_str is None:
            config_str = self.TEST_CONFIG
        with io.StringIO(config_str) as config_fp:
            config.readfp(config_fp)
        return config

    def test_seeded_defaults(self):
        config = nmconfig.NodeManagerConfig()
        sec_names = set(config.sections())
        self.assertIn('Arvados', sec_names)
        self.assertIn('Daemon', sec_names)
        self.assertFalse(any(name.startswith('Size ') for name in sec_names))

    def test_list_sizes(self):
        config = self.load_config()
        sizes = config.node_sizes()
        self.assertEqual(2, len(sizes))
        size, kwargs = sizes[0]
        self.assertEqual('Small', size.name)
        self.assertEqual(1, kwargs['cores'])
        self.assertEqual(0.8, kwargs['price'])
        # preemptible is False by default
        self.assertEqual(False, kwargs['preemptible'])
        # instance_type == arvados node size id by default
        self.assertEqual(kwargs['id'], kwargs['instance_type'])
        # Now retrieve the preemptible version
        size, kwargs = sizes[1]
        self.assertEqual('Small', size.name)
        self.assertEqual('1.preemptible', kwargs['id'])
        self.assertEqual(1, kwargs['cores'])
        self.assertEqual(0.8, kwargs['price'])
        self.assertEqual(True, kwargs['preemptible'])
        self.assertEqual('1', kwargs['instance_type'])


    def test_default_node_mem_scaling(self):
        config = self.load_config()
        self.assertEqual(0.95, config.getfloat('Daemon', 'node_mem_scaling'))

    def test_shutdown_windows(self):
        config = self.load_config()
        self.assertEqual([52, 6, 2], config.shutdown_windows())

    def test_log_levels(self):
        config = self.load_config()
        self.assertEqual({'level': logging.DEBUG,
                          'testlogger': logging.INFO},
                         config.log_levels())

    def check_dispatch_classes(self, config, module):
        setup, shutdown, update, monitor = config.dispatch_classes()
        self.assertIs(setup, module.ComputeNodeSetupActor)
        self.assertIs(shutdown, module.ComputeNodeShutdownActor)
        self.assertIs(update, module.ComputeNodeUpdateActor)
        self.assertIs(monitor, module.ComputeNodeMonitorActor)

    def test_default_dispatch(self):
        config = self.load_config()
        self.check_dispatch_classes(config, dispatch)

    def test_custom_dispatch(self):
        config = self.load_config(
            config_str=self.TEST_CONFIG + "[Daemon]\ndispatcher=slurm\n")
        self.check_dispatch_classes(config, slurm_dispatch)
