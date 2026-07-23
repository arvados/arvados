# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import dataclasses
import datetime
import pathlib

import arvados
import pytest

from testutil import FakePrometheusClient
from typing import ClassVar
from unittest import mock

from arvados_cluster_activity import report as acar

TODAY = datetime.date(2024, 4, 6)
REPORT_END = datetime.datetime.combine(TODAY, datetime.time(tzinfo=datetime.timezone.utc))
REPORT_START = REPORT_END - datetime.timedelta(days=2)

class _TestingClusterActivityReport(acar.ClusterActivityReport):
    _group: ClassVar[dict[str, str]] = {
        'name': 'WGS chr19 test for 2.7.2~rc3',
        'uuid': 'pirca-j7d0g-cukk4aw4iamj90c',
    }
    _user: ClassVar[dict[str, str]] = {
        'full_name': 'User1',
        'uuid': 'jutro-tpzed-a4qnxq3pcfcgtkz',
    }
    _workflow: ClassVar[dict[str, str]] = {
        'name': 'WGS processing workflow scattered over samples (v1.1-2-gcf002b3)',
        'uuid': 'pirca-7fd4e-01234abcde56789',
    }

    def _iter_runs(self):
        container = {
            'uuid': 'pirca-dz642-01234abcde56789',
            'started_at': '2024-04-05T20:40:31.234Z',
            'finished_at': '2024-04-05T21:59:52.345Z',
            'cost': 0.113,
        }
        request = {
            'container_uuid': container['uuid'],
            'created_at': '2024-04-05T20:38:07.123Z',
            'cumulative_cost': 1.371,
            'modified_by_user_uuid': self._user['uuid'],
            'name': self._workflow['name'],
            'owner_uuid': self._group['uuid'],
            'properties': {'template_uuid': self._workflow['uuid']},
            'requesting_container_uuid': None,
            'uuid': 'pirca-xvhdp-zyv7bm0tl3lm2nv',
        }
        self.groups.update(acar.key_by_uuid([self._group]))
        self.users.update(acar.key_by_uuid([self._user]))
        self.workflows.update(acar.key_by_uuid([self._workflow]))
        self.workflow_map[request['uuid']] = self._workflow['uuid']
        self.ctr_uuids.add(container['uuid'])
        yield acar.WorkflowRun(request, container, self._group, self._user, self._workflow)

    def iter_steps(self):
        container = {
            'uuid': 'pirca-dz642-12345abcde56789',
            'started_at': '2024-04-05T20:43:20.057Z',
            'finished_at': '2024-04-05T20:51:42.175Z',
            'cost': 0.116,
        }
        request = {
            'container_uuid': container['uuid'],
            'created_at': '2024-04-05T20:40:42.269Z',
            'cumulative_cost': 0.116,
            'modified_by_user_uuid': self._user['uuid'],
            'name': 'bwamem-samtools-view_2',
            'owner_uuid': self._group['uuid'],
            'requesting_container_uuid': next(iter(self.ctr_uuids)),
            'uuid': 'pirca-xvhdp-e63h0f57of5cr3t',
        }
        self.workflow_map[request['uuid']] = self._workflow['uuid']
        yield acar.WorkflowRun(request, container, self._group, self._user, self._workflow)


class FakeDate(datetime.date):
    @classmethod
    def today(cls):
        return TODAY


@pytest.fixture
def set_today(monkeypatch):
    monkeypatch.setattr(datetime, 'date', FakeDate)


@pytest.fixture
def mock_arv_client():
    arv_client = mock.Mock()
    arv_client.config.return_value = {
        "ClusterID": "xzzz1",
        "Services": {
            "Workbench2": {
                "ExternalURL": "https://xzzz1.arvadosapi.com"
            },
        },
    }
    arv_client.groups().list().execute.return_value = {
        'items_available': 6,
    }
    arv_client.users().list().execute.return_value = {
        'items_available': 4,
    }
    return arv_client


def test_csv_report(set_today, mock_arv_client, tmp_path):
    report = _TestingClusterActivityReport(
        REPORT_START,
        REPORT_END,
        mock_arv_client,
        None,
    )
    with (tmp_path / 'actual_report.csv').open('w+') as out_file:
        report.csv_report(out_file, columns=None, include_steps=True)
        out_file.seek(0)
        actual = out_file.read()
    assert actual == pathlib.Path('tests', 'test_report.csv').read_text()


def test_html_report(set_today, mock_arv_client, tmp_path):
    prom_client = FakePrometheusClient(
        '35253',
        REPORT_START,
        datetime.timedelta(hours=10),
    )
    report = _TestingClusterActivityReport(
        REPORT_START,
        REPORT_END,
        mock_arv_client,
        prom_client,
    )
    actual = report.html_report()
    (tmp_path / 'actual_report.html').write_text(actual)
    assert actual == pathlib.Path('tests', 'test_report.html').read_text()
