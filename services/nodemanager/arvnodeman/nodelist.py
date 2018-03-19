#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import subprocess

from . import clientactor
from . import config

import arvados.util

class ArvadosNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the Arvados node list.

    This actor regularly polls the list of Arvados node records,
    augments it with the latest SLURM node info (`sinfo`), and sends
    it to subscribers.
    """

    def is_common_error(self, exception):
        return isinstance(exception, config.ARVADOS_ERRORS)

    def _item_key(self, node):
        return node['uuid']

    def _send_request(self):
        nodelist = arvados.util.list_all(self._client.nodes().list)

        # node hostname, state
        sinfo_out = subprocess.check_output(["sinfo", "--noheader", "--format=%n|%t|%f"])
        nodestates = {}
        nodefeatures = {}
        for out in sinfo_out.splitlines():
            try:
                nodename, state, features = out.split("|", 3)
            except ValueError:
                continue
            if state in ('alloc', 'alloc*',
                         'comp',  'comp*',
                         'mix',   'mix*',
                         'drng',  'drng*'):
                nodestates[nodename] = 'busy'
            elif state in ('idle', 'fail'):
                nodestates[nodename] = state
            else:
                nodestates[nodename] = 'down'
            if features != "(null)":
                nodefeatures[nodename] = features

        for n in nodelist:
            if n["slot_number"] and n["hostname"] and n["hostname"] in nodestates:
                n["crunch_worker_state"] = nodestates[n["hostname"]]
            else:
                n["crunch_worker_state"] = 'down'
            n["slurm_node_features"] = nodefeatures.get(n["hostname"], "")

        return nodelist

class CloudNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the cloud node list.

    This actor regularly polls the cloud to get a list of running compute
    nodes, and sends it to subscribers.
    """

    def __init__(self, client, timer_actor, server_calc, *args, **kwargs):
        super(CloudNodeListMonitorActor, self).__init__(
            client, timer_actor, *args, **kwargs)
        self._calculator = server_calc

    def is_common_error(self, exception):
        return isinstance(exception, config.CLOUD_ERRORS)

    def _item_key(self, node):
        return node.id

    def _send_request(self):
        nodes = self._client.list_nodes()
        for n in nodes:
            # Replace with libcloud NodeSize object with compatible
            # CloudSizeWrapper object which merges the size info reported from
            # the cloud with size information from the configuration file.
            n.size = self._calculator.find_size(n.size.id)
        return nodes
