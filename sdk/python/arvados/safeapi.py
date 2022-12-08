# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Thread-safe wrapper for an Arvados API client

This module provides `ThreadSafeApiCache`, a thread-safe, API-compatible
Arvados API client.
"""

from __future__ import absolute_import

from builtins import object
import threading

from . import api
from . import config
from . import keep
from . import util

class ThreadSafeApiCache(object):
    """Thread-safe wrapper for an Arvados API client

    This class takes all the arguments necessary to build a lower-level
    Arvados API client `googleapiclient.discovery.Resource`, then
    transparently builds and wraps a unique object per thread. This works
    around the fact that the client's underlying HTTP client object is not
    thread-safe.

    Arguments:

    apiconfig: Mapping[str, str] | None
    : A mapping with entries for `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`,
      and optionally `ARVADOS_API_HOST_INSECURE`. If not provided, uses
      `arvados.config.settings` to get these parameters from user
      configuration.  You can pass an empty mapping to build the client
      solely from `api_params`.

    keep_params: Mapping[str, Any]
    : Keyword arguments used to construct an associated
      `arvados.keep.KeepClient`.

    api_params: Mapping[str, Any]
    : Keyword arguments used to construct each thread's API client. These
      have the same meaning as in the `arvados.api.api` function.

    version: str | None
    : A string naming the version of the Arvados API to use. If not specified,
      the code will log a warning and fall back to 'v1'.
    """

    def __init__(self, apiconfig=None, keep_params={}, api_params={}, version=None):
        if apiconfig or apiconfig is None:
            self._api_kwargs = api.api_kwargs_from_config(version, apiconfig, **api_params)
        else:
            self._api_kwargs = api.normalize_api_kwargs(version, **api_params)
        self.api_token = self._api_kwargs['token']
        self.request_id = self._api_kwargs.get('request_id')
        self.local = threading.local()
        self.keep = keep.KeepClient(api_client=self, **keep_params)

    def localapi(self):
        try:
            client = self.local.api
        except AttributeError:
            client = api.api_client(**self._api_kwargs)
            client._http._request_id = lambda: self.request_id or util.new_request_id()
            self.local.api = client
        return client

    def __getattr__(self, name):
        # Proxy nonexistent attributes to the thread-local API client.
        return getattr(self.localapi(), name)
