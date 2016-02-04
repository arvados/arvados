#!/usr/bin/env python

from __future__ import absolute_import, print_function

from . import clientactor
from . import config

class ArvadosNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the Arvados node list.

    This actor regularly polls the list of Arvados node records, and
    sends it to subscribers.
    """

    def is_common_error(self, exception):
        return isinstance(exception, config.ARVADOS_ERRORS)

    def _item_key(self, node):
        return node['uuid']

    def _send_request(self):
        return self._client.nodes().list(limit=10000).execute()['items']


class CloudNodeListMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to poll the cloud node list.

    This actor regularly polls the cloud to get a list of running compute
    nodes, and sends it to subscribers.
    """

    def is_common_error(self, exception):
        return self._client.is_cloud_exception(exception)

    def _item_key(self, node):
        return node.id

    def _send_request(self):
        n = self._client.list_nodes()
        return n
