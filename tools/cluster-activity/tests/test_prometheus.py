# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import base64
import dataclasses
import datetime
import errno
import os
import typing as t

import pytest

from arvados_cluster_activity import main as aca_main
from arvados_cluster_activity import prometheus
from testutil import FakePrometheusClient

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


class TestEmptyUsageChart:
    @pytest.fixture
    def empty_chart(self):
        return prometheus.PrometheusUsageChart(
            'Test Empty Chart',
            'value',
            iter(()),
        )

    def test_iter_values(self, empty_chart):
        assert list(empty_chart.iter_values()) == []

    def test_last_value_default(self, empty_chart):
        assert empty_chart.last_value() is None

    def test_last_value(self, empty_chart):
        sentinel = object()
        assert empty_chart.last_value(sentinel) is sentinel

    def test_json_object(self, empty_chart):
        assert empty_chart.json_object() == {
            'label': 'Test Empty Chart',
            'charts': [{
                'options': {
                    'legend': 'always',
                    'connectSeparatedPoints': True,
                    'labels': ['date', 'value'],
                    'includeZero': True,
                    'title': 'Test Empty Chart',
                },
                'data': [],
            }],
        }


class TestUsageChart:
    START_TIME = datetime.datetime(2026, 7, 8, 9, 10)

    @pytest.fixture
    def count_chart(self):
        prom_client = FakePrometheusClient('123456', self.START_TIME)
        metric = prometheus.get_metric_usage(
            prom_client,
            self.START_TIME,
            self.START_TIME.replace(hour=23),
            'test_count',
            '5min',
        )
        return prometheus.PrometheusUsageChart('Count Chart', 'count', metric)

    def test_iter_values(self, count_chart):
        assert list(count_chart.iter_values()) == list(range(1, 7))

    def test_last_value(self, count_chart):
        assert count_chart.last_value() == 6

    def test_last_value_default(self, count_chart):
        assert count_chart.last_value(-1) == 6

    def test_json_object(self, count_chart):
        start_ms = self.START_TIME.timestamp() * 1000
        delta_ms = datetime.timedelta(minutes=5).total_seconds() * 1000
        assert count_chart.json_object() == {
            'label': 'Count Chart',
            'charts': [{
                'options': {
                    'legend': 'always',
                    'connectSeparatedPoints': True,
                    'labels': ['date', 'count'],
                    'includeZero': True,
                    'title': 'Count Chart',
                },
                'data': [
                    [int(start_ms + delta_ms * n), n + 1]
                    for n in range(6)
                ],
            }],
        }
