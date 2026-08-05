# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import datetime
import itertools

import pytest

from arvados import util as arv_util
from arvados_cluster_activity import report as acar
from typing import Any

CREATED_AT = datetime.datetime(2026, 5, 4, 3, 2, 1)
COUNTER = itertools.count(int('11110', 16))

def about_now(microsecond=123456):
    return datetime.datetime.now(datetime.timezone.utc).replace(microsecond=microsecond)


def arv_dt_fmt(dt):
    return dt.strftime('%Y-%m-%dT%H:%M:%S.%fZ')


def new_support_object(src, prefix, cluster='zzzzz', name_key=None):
    if not src:
        return new_uuid(prefix, cluster), None
    elif isinstance(src, dict):
        return src['uuid'], src
    elif arv_util.uuid_pattern.fullmatch(src):
        return src, None
    else:
        uuid = new_uuid(prefix, cluster)
        if name_key is None:
            name_key = 'full_name' if prefix == 'tpzed' else 'name'
        return uuid, {'uuid': uuid, name_key: src}


def new_uuid(prefix, cluster='zzzzz'):
    suffix = next(COUNTER)
    return f'{cluster}-{prefix}-{suffix:015x}'


def new_run(
        requesting_container_uuid: str | None = None,
        created_at: datetime.datetime | datetime.timedelta | None = None,
        started_at: datetime.datetime | datetime.timedelta = datetime.timedelta(minutes=1),
        finished_at: datetime.datetime | datetime.timedelta = datetime.timedelta(hours=1, minutes=2, seconds=3),
        owner: dict[str, Any] | str | None = None,
        user: dict[str, Any] | str | None = None,
        workflow: dict[str, Any] | str | None = None,
        name: str = 'WorkflowRun Request',
        cost: float = 5.678,
        cumulative_cost: float | None = None,
        cluster: str = 'zzzzz',
):
    if created_at is None:
        created_at = datetime.timedelta(minutes=next(COUNTER))
    if isinstance(created_at, datetime.timedelta):
        created_at = CREATED_AT + created_at
    if isinstance(started_at, datetime.timedelta):
        started_at = created_at + started_at
    if isinstance(finished_at, datetime.timedelta):
        finished_at = started_at + finished_at

    owner_uuid, owner = new_support_object(owner, 'j7d0g', cluster)
    user_uuid, user = new_support_object(user, 'tpzed', cluster)
    if workflow is None:
        workflow_uuid = None
    else:
        workflow_uuid, workflow = new_support_object(workflow, '7fd4e', cluster)

    if cumulative_cost is not None:
        pass
    elif requesting_container_uuid is None:
        cumulative_cost = cost * 3
    else:
        cumulative_cost = cost

    container = {
        'uuid': new_uuid('dz642', cluster),
        'started_at': arv_dt_fmt(started_at),
        'finished_at': arv_dt_fmt(finished_at),
        'cost': cost,
    }
    request = {
        'container_uuid': container['uuid'],
        'created_at': arv_dt_fmt(created_at),
        'cumulative_cost': cumulative_cost,
        'modified_by_user_uuid': user_uuid,
        'name': name,
        'owner_uuid': owner_uuid,
        'requesting_container_uuid': requesting_container_uuid,
        'uuid': new_uuid('xvhdp', cluster),
    }
    if workflow_uuid is not None:
        request['properties'] = {'template_uuid': workflow_uuid}
    return acar.WorkflowRun(request, container, owner, user, workflow)


def test_container_cost():
    expected = 12.345
    assert new_run(cost=expected).container_cost() == pytest.approx(expected)


def test_cumulative_cost():
    expected = 12.345
    assert new_run(cumulative_cost=expected).cumulative_cost() == pytest.approx(expected)


@pytest.mark.parametrize('field', [
    'created_at',
    'started_at',
    'finished_at',
])
def test_run_datetime(field):
    dt = about_now()
    run = new_run(**{field: dt})
    assert getattr(run, field)() == dt


@pytest.mark.parametrize('secs', [.6, 66, 6789])
def test_runtime(secs):
    expected = datetime.timedelta(seconds=secs)
    assert new_run(finished_at=expected).runtime() == expected


@pytest.mark.parametrize('secs', [.6, 66, 6789])
def test_begin_end_runtime(secs):
    exp_finish = about_now()
    exp_runtime = datetime.timedelta(seconds=secs)
    exp_started = exp_finish - exp_runtime
    run = new_run(started_at=exp_started, finished_at=exp_finish)
    start, end, runtime = run.start_end_runtime()
    assert start == exp_started
    assert end == exp_finish
    assert runtime == exp_runtime


@pytest.mark.parametrize('prefix', ['j7d0g', 'tpzed'])
def test_owner_name(prefix):
    name = f'Test {prefix}'
    assert new_run(owner=name).owner_name() == name


def test_owner_name_none():
    assert new_run(owner=None).owner_name() == 'unknown owner'


@pytest.mark.parametrize('name,expected', [
    ('merge_indices', 'merge_indices'),
    ('merge_indices_2', 'merge_indices'),
    ('merge_indices_25', 'merge_indices'),
    ('merge_indices_255', 'merge_indices'),
    ('merge_indices_2_5', 'merge_indices_2'),
    ('merge_indices 2.5', 'merge_indices 2.5'),
])
def test_step_name(name, expected):
    run = new_run(name=name, requesting_container_uuid=new_uuid('dz642'))
    assert run.step_name() == expected


@pytest.mark.parametrize('name', [
    'merge_indices',
    'merge_indices_2',
    'merge_indices_25',
    'merge_indices_255',
    'merge_indices_2_5',
    'merge_indices 2.5',
])
def test_step_name_workflow_runner(name):
    assert new_run(name=name).step_name() == 'workflow runner'


def test_template_uuid():
    uuid = new_uuid('7fd4e')
    assert new_run(workflow=uuid).template_uuid() == uuid


@pytest.mark.parametrize('default', ['no workflow', None])
def test_template_uuid_default(default):
    assert new_run().template_uuid(default) == default


@pytest.mark.parametrize('name,expected', [
    (None, 'unknown user'),
    ('Test User', 'Test User'),
    ('Alex Zig', 'Alex Zig'),
])
def test_user_name(name, expected):
    assert new_run(user=name).user_name() == expected


@pytest.mark.parametrize('name', [
    'Test Workflow',
    'WGS scattered over samples',
])
def test_workflow_name(name):
    assert new_run(workflow=name).workflow_name() == name


@pytest.mark.parametrize('default', [None, 'no workflow'])
def test_workflow_name_default(default):
    assert new_run().workflow_name(default) == default


@pytest.mark.parametrize('default', [None, 'no workflow'])
def test_workflow_name_uuid(default):
    uuid = new_uuid('7fd4e')
    assert new_run(workflow=uuid).workflow_name(default) == uuid


@pytest.mark.parametrize('name,requesting_container_uuid', [
    ('CSV Builder', None),
    ('CSV Collator_3', new_uuid('dz642')),
])
def test_csv_row(name, requesting_container_uuid):
    owner_uuid, owner = new_support_object('CSV Owner', 'j7d0g')
    user_uuid, user = new_support_object('CSV User', 'tpzed')
    wf_uuid, workflow = new_support_object('CSV Template', '7fd4e')
    run = new_run(
        name=name,
        owner=owner,
        user=user,
        workflow=workflow,
        requesting_container_uuid=requesting_container_uuid,
        created_at=datetime.datetime(2025, 6, 7, 8, 9, 10),
        cost=0.2,
        cumulative_cost=1.2,
    )
    exp_uuid = run.request['uuid']
    if requesting_container_uuid is None:
        exp_step = 'workflow runner'
    else:
        exp_step = 'CSV Collator'
    assert run.csv_row() == {
        'Project': 'CSV Owner',
        'ProjectUUID': owner_uuid,
        'Workflow': 'CSV Template',
        'WorkflowUUID': wf_uuid,
        'Step': exp_step,
        'StepUUID': exp_uuid,
        'Sample': name,
        'SampleUUID': exp_uuid,
        'User': 'CSV User',
        'UserUUID': user_uuid,
        'Submitted': '2025-06-07 08:09:10',
        'Started': '2025-06-07 08:10:10',
        'Finished': '2025-06-07 09:12:13',
        'Runtime': '1:02:03',
        'Cost': 0.2,
        'CumulativeCost': 1.2,
    }


def test_summary_cost():
    summary = acar.WorkflowRunSummary()
    expected = 0.0
    for cost in [1.1, 2.2, 3.3]:
        expected += cost
        summary.tally(new_run(cumulative_cost=cost))
        assert summary.total_cost() == pytest.approx(expected)


def test_summary_runtime():
    summary = acar.WorkflowRunSummary()
    expected = 0.0
    for secs in [90.9, 990, 7234.5]:
        expected += secs
        summary.tally(new_run(finished_at=datetime.timedelta(seconds=secs)))
        assert summary.total_runtime() == pytest.approx(expected)


def test_summary_runs():
    summary = acar.WorkflowRunSummary()
    for expected in range(1, 6):
        summary.tally(new_run())
        assert summary.total_runs() == expected


def test_summary_users():
    runs = [new_run(user=name) for name in ['User One', 'User Two']]
    summary = acar.WorkflowRunSummary()
    for expected, run in enumerate(runs, 1):
        summary.tally(run)
        assert summary.total_users() == expected
    for run in runs:
        summary.tally(new_run(user=run.user))
        assert summary.total_users() == 2


def test_summary_usernames():
    usernames = dict([
        new_support_object('User One', 'tpzed'),
        new_support_object('User Two', 'tpzed'),
    ])
    users = iter(usernames.values())
    _, extra_user = new_support_object('User Three', 'tpzed')
    summary = acar.WorkflowRunSummary()

    summary.tally(new_run(user=next(users)))
    assert summary.usernames(usernames) == ['User One']
    assert summary.usernames(usernames, '?') == ['User One']

    summary.tally(new_run(user=extra_user))
    assert summary.usernames(usernames) == ['User One', 'unknown user']
    assert summary.usernames(usernames, '?') == ['?', 'User One']

    summary.tally(new_run(user=next(users)))
    assert summary.usernames(usernames) == ['User One', 'User Two', 'unknown user']
    assert summary.usernames(usernames, '?') == ['?', 'User One', 'User Two']

    for user in [*usernames.values(), extra_user]:
        summary.tally(new_run(user=user))
        assert summary.usernames(usernames) == ['User One', 'User Two', 'unknown user']
        assert summary.usernames(usernames, '?') == ['?', 'User One', 'User Two']


def test_stats_cost():
    stats = acar.WorkflowRunStatistics()
    stats.tally(new_run(cumulative_cost=1.1))
    assert stats.mean_cost() == 1.1
    assert stats.median_cost() == 1.1
    assert stats.total_cost() == 1.1

    stats.tally(new_run(cumulative_cost=2.2))
    assert stats.mean_cost() == pytest.approx(1.65)
    assert stats.median_cost() == pytest.approx(1.65)
    assert stats.total_cost() == pytest.approx(3.3)

    stats.tally(new_run(cumulative_cost=5.5))
    assert stats.mean_cost() == pytest.approx(8.8 / 3)
    assert stats.median_cost() == pytest.approx(2.2)
    assert stats.total_cost() == pytest.approx(8.8)


def test_stats_runtime():
    stats = acar.WorkflowRunStatistics()
    stats.tally(new_run(finished_at=datetime.timedelta(seconds=30.3)))
    assert stats.mean_runtime() == 30.3
    assert stats.median_runtime() == 30.3
    assert stats.total_runtime() == 30.3

    stats.tally(new_run(finished_at=datetime.timedelta(seconds=60.6)))
    assert stats.mean_runtime() == pytest.approx(45.45)
    assert stats.median_runtime() == pytest.approx(45.45)
    assert stats.total_runtime() == pytest.approx(90.9)

    stats.tally(new_run(finished_at=datetime.timedelta(seconds=963.9)))
    assert stats.mean_runtime() == pytest.approx(351.6)
    assert stats.median_runtime() == pytest.approx(60.6)
    assert stats.total_runtime() == pytest.approx(1054.8)


def test_stats_runs():
    stats = acar.WorkflowRunStatistics()
    for expected in range(1, 6):
        stats.tally(new_run(cumulative_cost=expected))
        assert stats.total_runs() == expected
