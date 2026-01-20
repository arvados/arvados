# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import base64
import dataclasses
import errno
import os
import typing as t

import pytest

from arvados_cluster_activity import main as aca_main

_PROMETHEUS_ENVKEYS = [key for key in os.environ if key.startswith('PROMETHEUS_')]

@dataclasses.dataclass
class PrometheusConnect:
    url: str = 'http://127.0.0.1:9090'
    headers: dict[str, str] | None = None
    disable_ssl: bool = False
    retry: 'urllib3.util.retry.Retry | None' = None
    auth: tuple | None = None
    proxy: dict | None = None
    session: 'requests.sessions.Session | None' = None
    timeout: int = None
    method: str = 'GET'

    _NONE_HOST: t.ClassVar[str] = 'http://[100::1234:5678:90ab:cdef]:48084/'
    _NONE_HOST_ERR: t.ClassVar[int] = errno.ENETUNREACH

    def __post_init__(self):
        if self.url == self._NONE_HOST:
            raise OSError(self._NONE_HOST_ERR, os.strerror(self._NONE_HOST_ERR))

    def _check_host(self, expected):
        assert self.url == expected

    def _check_auth(self, token_or_user, password=None):
        method, sep, auth = self.headers.get('Authorization', '').partition(' ')
        assert sep, "Authorization header has malformed value"
        if password is None:
            assert method == 'Bearer'
            assert auth == token_or_user
        else:
            assert method == 'Basic'
            actual = base64.b64decode(auth)
            assert actual == f"{token_or_user}:{password}".encode('utf-8')


@pytest.fixture(autouse=True)
def clean_env(monkeypatch):
    for key in _PROMETHEUS_ENVKEYS:
        monkeypatch.delenv(key)
    monkeypatch.setattr(aca_main, 'PrometheusConnect', PrometheusConnect)


def test_no_host():
    assert aca_main.get_prometheus_client() is None


def test_no_creds(monkeypatch):
    monkeypatch.setenv('PROMETHEUS_HOST', PrometheusConnect._NONE_HOST)
    assert aca_main.get_prometheus_client() is None


def test_no_connection(monkeypatch):
    monkeypatch.setenv('PROMETHEUS_HOST', PrometheusConnect._NONE_HOST)
    monkeypatch.setenv('PROMETHEUS_APIKEY', 'NoAPIkey')
    assert aca_main.get_prometheus_client() is None


def test_apikey(monkeypatch):
    monkeypatch.setenv('PROMETHEUS_HOST', 'https://token.prom.invalid/')
    monkeypatch.setenv('PROMETHEUS_APIKEY', 'testAPIKEY')
    actual = aca_main.get_prometheus_client()
    assert actual is not None
    actual._check_host('https://token.prom.invalid/')
    actual._check_auth('testAPIKEY')


def test_username_password(monkeypatch):
    monkeypatch.setattr(aca_main, 'PrometheusConnect', PrometheusConnect)
    monkeypatch.setenv('PROMETHEUS_HOST', 'https://namepass.prom.invalid/')
    monkeypatch.setenv('PROMETHEUS_USER', 'testname')
    monkeypatch.setenv('PROMETHEUS_PASSWORD', 'testpass')
    actual = aca_main.get_prometheus_client()
    assert actual is not None
    actual._check_host('https://namepass.prom.invalid/')
    actual._check_auth('testname', 'testpass')
