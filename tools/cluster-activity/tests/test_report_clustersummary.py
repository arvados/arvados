# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import dataclasses
import datetime
import json

from unittest import mock

import pytest

from arvados_cluster_activity import report as acar
from testutil import FakePrometheusClient

START_TIME = datetime.datetime(2026, 7, 11, tzinfo=datetime.timezone.utc)
END_TIME = START_TIME + datetime.timedelta(days=1)

@dataclasses.dataclass
class FakeArvadosClient:
    user_count: int = 0
    proj_count: int = 0
    config: dict = dataclasses.field(default_factory=lambda: {
        'ClusterID': 'zzzzz',
        'Services': {
            'Workbench2': {
                'ExternalURL': 'https://workbench.zzzzz.example/',
            },
        },
    })

    def count_items(self, attr):
        def execute():
            return {'items_available': getattr(self, attr)}
        return execute

    def build_mock(self):
        mock_arv = mock.Mock()
        mock_arv.config.return_value = self.config
        mock_arv.groups().list().execute.side_effect = self.count_items('proj_count')
        mock_arv.users().list().execute.side_effect = self.count_items('user_count')
        return mock_arv


class TestWithoutPrometheus:
    @pytest.fixture
    def summary(self):
        arv_client = FakeArvadosClient(10, 20).build_mock()
        return acar.ClusterSummary.query(arv_client, None, START_TIME, END_TIME)

    def test_counts(self, summary):
        assert summary.total_users == 10
        assert summary.total_projects == 20
        assert summary.managed_data_end is None
        assert summary.stored_data_end is None
        assert summary.storage_cost is None

    def test_have_charts(self, summary):
        assert not summary.have_charts()

    def test_chart_json(self, summary):
        assert summary.chart_json() == '[]'

    def test_dedup_ratio(self, summary):
        assert summary.dedup_ratio() is None

    def test_monthly_storage_cost(self, summary):
        assert summary.monthly_storage_cost() is None




class TestWithPrometheus:
    @pytest.fixture
    def summary(self):
        arv_client = FakeArvadosClient(15, 35).build_mock()
        prom_client = FakePrometheusClient('5', START_TIME)
        return acar.ClusterSummary.query(arv_client, prom_client, START_TIME, END_TIME)

    def test_counts(self, summary):
        assert summary.total_users == 15
        assert summary.total_projects == 35
        assert summary.managed_data_end == 5
        assert summary.stored_data_end == 5
        assert 0 < summary.storage_cost < 0.01

    def test_have_charts(self, summary):
        assert summary.have_charts()

    def test_chart_json(self, summary):
        charts = json.loads(summary.chart_json())
        for ii, (title, label) in enumerate([
                ("Concurrent running containers", "containers"),
                ("Data under management", "managed"),
                ("Storage usage", "used"),
        ]):
            chart = charts[ii]['charts'][0]
            assert chart['options']['labels'] == ['date', label]
            assert chart['options']['title'] == title
            assert chart['data'][0][0] == int(START_TIME.timestamp() * 1000)
            assert all(value == 5 for _, value in chart['data'])

    def test_dedup_ratio(self, summary):
        assert summary.dedup_ratio() == 1

    def test_monthly_storage_cost(self, summary):
        assert summary.monthly_storage_cost() == pytest.approx(0)
