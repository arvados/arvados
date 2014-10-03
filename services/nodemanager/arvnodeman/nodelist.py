#!/usr/bin/env python

import arvados.errors as arverror

from . import clientactor
from . import config

class ArvadosNodeListMonitorActor(clientactor.RemotePollLoopActor):
    CLIENT_ERRORS = config.ARVADOS_ERRORS
    LOGGER_NAME = 'arvnodeman.arvados_nodes'

    def _item_key(self, node):
        return node['uuid']

    def _send_request(self):
        return self._client.nodes().list(limit=10000).execute()['items']


class CloudNodeListMonitorActor(clientactor.RemotePollLoopActor):
    CLIENT_ERRORS = config.CLOUD_ERRORS
    LOGGER_NAME = 'arvnodeman.cloud_nodes'

    def _item_key(self, node):
        return node.id

    def _send_request(self):
        return self._client.list_nodes()
