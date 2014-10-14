#!/usr/bin/env python

from __future__ import absolute_import, print_function

from . import clientactor
from . import config

class ArvadosNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the Arvados node list.

    This actor regularly polls the list of Arvados node records, and
    sends it to subscribers.
    """

    CLIENT_ERRORS = config.ARVADOS_ERRORS
    LOGGER_NAME = 'arvnodeman.arvados_nodes'

    def _item_key(self, node):
        return node['uuid']

    def _send_request(self):
        return self._client.nodes().list(limit=10000).execute()['items']


class CloudNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the cloud node list.

    This actor regularly polls the cloud to get a list of running compute
    nodes, and sends it to subscribers.
    """

    CLIENT_ERRORS = config.CLOUD_ERRORS
    LOGGER_NAME = 'arvnodeman.cloud_nodes'

    def _item_key(self, node):
        return node.id

    def _send_request(self):
        return self._client.list_nodes()
