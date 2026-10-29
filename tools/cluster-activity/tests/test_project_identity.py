# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import datetime
from unittest import mock

from arvados_cluster_activity import report as acar


class DuplicateProjectNameReport(acar.ClusterActivityReport):
    projects = (
        {'uuid': 'zzzzz-j7d0g-proj00000000001', 'name': 'Shared Project'},
        {'uuid': 'zzzzz-j7d0g-proj00000000002', 'name': 'Shared Project'},
    )
    user = {'uuid': 'zzzzz-tpzed-user00000000001', 'full_name': 'Synthetic User'}
    workflow = {
        'uuid': 'zzzzz-7fd4e-wf0000000000001',
        'name': 'Synthetic Workflow',
    }

    def _iter_runs(self):
        self.groups.update(acar.key_by_uuid(self.projects))
        self.users.update(acar.key_by_uuid([self.user]))
        self.workflows.update(acar.key_by_uuid([self.workflow]))

        for index, project in enumerate(self.projects, start=1):
            container = {
                'uuid': f'zzzzz-dz642-cont0000000000{index}',
                'started_at': '2026-09-01T10:00:00Z',
                'finished_at': '2026-09-01T11:00:00Z',
                'cost': 1.0,
            }
            request = {
                'container_uuid': container['uuid'],
                'created_at': '2026-09-01T09:59:00Z',
                'cumulative_cost': 1.0,
                'modified_by_user_uuid': self.user['uuid'],
                'name': 'Synthetic Workflow',
                'owner_uuid': project['uuid'],
                'properties': {'template_uuid': self.workflow['uuid']},
                'requesting_container_uuid': None,
                'uuid': f'zzzzz-xvhdp-request0000000{index}',
            }
            yield acar.WorkflowRun(request, container, project, self.user, self.workflow)


def test_same_project_name_keeps_project_and_workflow_totals_separate():
    arv_client = mock.Mock()
    arv_client.config.return_value = {
        'ClusterID': 'zzzzz',
        'Services': {'Workbench2': {'ExternalURL': 'https://example.invalid/'}},
    }
    arv_client.groups().list().execute.return_value = {'items_available': 2}
    arv_client.users().list().execute.return_value = {'items_available': 1}

    report = DuplicateProjectNameReport(
        datetime.datetime(2026, 9, 1, tzinfo=datetime.timezone.utc),
        datetime.datetime(2026, 9, 2, tzinfo=datetime.timezone.utc),
        arv_client,
        None,
    )

    html = report.html_report()

    project_uuids = [project['uuid'] for project in report.projects]
    for project_uuid in project_uuids:
        assert report.summaries[project_uuid].total_runs() == 1
        assert report.summaries[project_uuid].total_cost() == 1.0
        assert report.statistics[project_uuid][report.workflow['uuid']].total_runs() == 1
        assert f'<h2><a href="https://example.invalid/projects/{project_uuid}">Shared Project</a></h2>' in html
