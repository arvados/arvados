#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import collections
import csv
import dataclasses
import datetime
import enum
import functools
import itertools
import json
import logging
import re
import statistics
import urllib.parse

from pathlib import PurePath
from typing import Callable, ClassVar, Dict, List, Mapping

import arvados.api_resources as arv_types
import arvados.util
import ciso8601
import jinja2
import markupsafe

from arvados_cluster_activity.prometheus import get_metric_usage, PrometheusUsageChart

@dataclasses.dataclass
class BytesFormatter:
    """Function to format a number of bytes as a human-friendly string"""
    exp: int
    suffixes: list[str]

    def __call__(self, value):
        if value is None:
            return None
        for suffix in self.suffixes:
            value /= self.exp
            if value < self.exp:
                break
        return f'{value:.3f} {suffix}'


bytes_base2_fmt = BytesFormatter(1024, ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB'])
bytes_si_fmt = BytesFormatter(1000, ['KB', 'MB', 'GB', 'TB', 'PB', 'EB'])

_JINJA_FILTERS = {
    'bytes_base2': bytes_base2_fmt,
    'bytes_si': bytes_si_fmt,
    'bytes': bytes_si_fmt,
}
def _jinja_filter(func):
    _JINJA_FILTERS[func.__name__.removesuffix('_fmt')] = func
    return func


@_jinja_filter
def cost_fmt(value):
    if value is None:
        return None
    return f'${value:,.02f}'


@_jinja_filter
def date_fmt(value):
    return value.strftime('%Y-%m-%d')


@_jinja_filter
def datespan_fmt(value, end):
    begin_isodate = date_fmt(value)
    end_isodate = date_fmt(end)
    if begin_isodate == end_isodate:
        return begin_isodate
    else:
        return f'{begin_isodate} to {end_isodate}'


@_jinja_filter
def datetime_fmt(value):
    return value.strftime('%Y-%m-%d %H:%M:%S')


@_jinja_filter
def duration_hms_fmt(value):
    if isinstance(value, datetime.timedelta):
        value = value.total_seconds()
    mins, secs = divmod(round(value), 60)
    hrs, mins = divmod(mins, 60)
    return f'{hrs}:{mins:02d}:{secs:02d}'


@_jinja_filter
def duration_hrs_fmt(value, suffix=''):
    if isinstance(value, datetime.timedelta):
        value = value.total_seconds()
    sep = ' ' if suffix and not suffix.startswith(' ') else ''
    return f'{value / 3600:,.1f}{sep}{suffix}'


@_jinja_filter
def numeric_fmt(value, prec=None):
    if value is None:
        return None
    elif prec is None:
        fmt = '{:,d}'
    else:
        fmt = f'{{:,.{prec}f}}'
    return fmt.format(value)


class S3CostPeriod(enum.IntEnum):
    MONTHLY = 1
    DAILY = 30
    HOURLY = 30 * 24


def s3_cost(num_bytes, period):
    """Calculate the USD cost of storing `num_bytes` over the time `period`"""
    if num_bytes is None:
        return None
    gb = num_bytes / (1 << 30)
    first_50tb = min(1024*50, gb)
    next_450tb = max(min(1024*450, gb-1024*50), 0)
    over_500tb = max(gb-1024*500, 0)
    monthly_cost = (first_50tb * 0.023) + (next_450tb * 0.022) + (over_500tb * 0.021)
    return monthly_cost / period.value


def aws_monthly_cost(value):
    return s3_cost(value, S3CostPeriod.MONTHLY)


@dataclasses.dataclass(slots=True)
class WorkflowRunCSVRow:
    Project: str
    ProjectUUID: str
    Workflow: str
    WorkflowUUID: str
    Step: str
    StepUUID: str
    Sample: str
    SampleUUID: str
    User: str
    UserUUID: str
    Submitted: str
    Started: str
    Finished: str
    Runtime: str
    Cost: str
    CumulativeCost: str

    @classmethod
    def field_names(cls) -> tuple[str]:
        return tuple(f.name for f in dataclasses.fields(cls))


@dataclasses.dataclass(slots=True)
class WorkflowRun:
    """Encapsulate and query a run container

    This class bundles together a satisfied container request, its container,
    and other associated records like owner, modifying user, and workflow.
    Methods fetch data from the low-level records with appropriate fallbacks.

    It's called `WorkflowRun` because the report is focused on workflows and
    defaults reflect that, but in principle it can wrap any container request
    plus container.
    """
    request: arv_types.ContainerRequest
    container: arv_types.Container
    owner: arv_types.Group | arv_types.User | None = None
    user: arv_types.User | None = None
    workflow: arv_types.Workflow | None = None

    def container_cost(self):
        return self.container['cost']

    def created_at(self):
        return ciso8601.parse_datetime(self.request['created_at'])

    def cumulative_cost(self):
        return self.request['cumulative_cost']

    def finished_at(self):
        return ciso8601.parse_datetime(self.container['finished_at'])

    def name_key(self):
        if self.workflow is None:
            return self.request_name_without_suffix()
        else:
            return self.workflow['name']

    def owner_name(self):
        if self.owner is None:
            return 'unknown owner'
        try:
            return self.owner['full_name']
        except KeyError:
            return self.owner['name']

    def request_name_without_suffix(self):
        """Return the request name without any numbered suffix

        This provides a normalized name for workflow steps with fanout.
        """
        return re.sub(r'_[0-9]+$', '', self.request['name'])

    def runtime(self):
        return self.finished_at() - self.started_at()

    def started_at(self):
        return ciso8601.parse_datetime(self.container['started_at'])

    def start_end_runtime(self):
        """Return start time, finish time, and runtime delta in one call

        Most use cases want all three and this prevents needless re-parsing."""
        start = self.started_at()
        finish = self.finished_at()
        return start, finish, finish - start

    def step_name(self):
        if self.request['requesting_container_uuid'] is None:
            return 'workflow runner'
        else:
            return self.request_name_without_suffix()

    def template_uuid(self, default=None):
        try:
            return self.request['properties']['template_uuid']
        except KeyError:
            return default

    def user_name(self):
        if self.user is None:
            return 'unknown user'
        else:
            return self.user['full_name']

    def workflow_name(self, default=None):
        if self.workflow is not None:
            return self.workflow['name']
        else:
            return self.template_uuid(default)

    def csv_row(self) -> dict[str, str]:
        started, finished, runtime = self.start_end_runtime()
        return dataclasses.asdict(WorkflowRunCSVRow(
            Project=self.owner_name(),
            ProjectUUID=self.request['owner_uuid'],
            Workflow=self.workflow_name('workflow run from command line'),
            WorkflowUUID=self.template_uuid('none'),
            Step=self.step_name(),
            StepUUID=self.request['uuid'],
            Sample=self.request['name'],
            SampleUUID=self.request['uuid'],
            User=self.user_name(),
            UserUUID=self.request['modified_by_user_uuid'],
            Submitted=datetime_fmt(self.created_at()),
            Started=datetime_fmt(started),
            Finished=datetime_fmt(finished),
            Runtime=duration_hms_fmt(runtime),
            Cost=self.container_cost(),
            CumulativeCost=self.cumulative_cost(),
        ))


@dataclasses.dataclass(slots=True)
class WorkflowRunSummary:
    """Collect summary statistics about a series of related WorkflowRuns

    Call the `tally()` method with a series of WorkflowRuns, then call the
    other methods to report collected statistics. The caller determines how
    the WorkflowRuns are related; nothing in this class enforces any relation.
    Refer to the `summaries` field of `ClusterActivityReport`.

    This class collects basic high-level statistics about related runs. Most
    fields are a single cumulative data point. The only exception is `users`
    which will grow much slower than the number of runs in most real-world
    applications.
    """
    owner_uuid: str | None = None

    cost: float = dataclasses.field(init=False, default=0.0)
    count: int = dataclasses.field(init=False, default=0)
    earliest: datetime.datetime = dataclasses.field(
        init=False,
        default=datetime.datetime.max.replace(tzinfo=datetime.timezone.utc),
    )
    latest: datetime.datetime = dataclasses.field(
        init=False,
        default=datetime.datetime.min.replace(tzinfo=datetime.timezone.utc),
    )
    runtime: datetime.timedelta = dataclasses.field(init=False, default_factory=datetime.timedelta)
    users: set[str] = dataclasses.field(init=False, default_factory=set)

    def tally(self, run):
        started, finished, runtime = run.start_end_runtime()
        self.earliest = min(self.earliest, started)
        self.latest = max(self.latest, finished)
        self.cost += run.cumulative_cost()
        self.count += 1
        self.runtime += runtime
        self.users.add(run.request['modified_by_user_uuid'])

    def total_cost(self):
        return self.cost

    def total_runtime(self):
        return self.runtime.total_seconds()

    def total_runs(self):
        return self.count

    def total_users(self):
        return len(self.users)

    def usernames(self, users_map, default_name='unknown user'):
        default_user = {'full_name': default_name}
        return sorted(
            users_map.get(uuid, default_user)['full_name']
            for uuid in self.users
        )


@dataclasses.dataclass(slots=True)
class WorkflowRunStatistics:
    """Collect detailed statistics about a series of related WorkflowRuns

    Call the `tally()` method with a series of WorkflowRuns, then call the
    other methods to report collected statistics. The caller determines how
    the WorkflowRuns are related; nothing in this class enforces any relation.
    Refer to the `statistics` field of `ClusterActivityReport`.

    This class collects a few data points per tallied run, so you can expect
    it to grow in RAM use O(n) with the number of runs.
    """
    owner_uuid: str | None = None
    workflow_uuid: str | None = None

    costs: list[float] = dataclasses.field(init=False, default_factory=list)
    runtimes: list[float] = dataclasses.field(init=False, default_factory=list)

    def tally(self, run):
        self.costs.append(run.cumulative_cost())
        self.runtimes.append(run.runtime().total_seconds())

    def mean_cost(self):
        return statistics.mean(self.costs)

    def median_cost(self):
        return statistics.median(self.costs)

    def total_cost(self):
        return sum(self.costs)

    def mean_runtime(self):
        return statistics.mean(self.runtimes)

    def median_runtime(self):
        return statistics.median(self.runtimes)

    def total_runtime(self):
        return sum(self.runtimes)

    def total_runs(self):
        return len(self.costs)


@dataclasses.dataclass
class ClusterSummary:
    containers: PrometheusUsageChart
    managed_data: PrometheusUsageChart
    stored_data: PrometheusUsageChart
    total_users: int = 0
    total_projects: int = 0

    managed_data_end: int | None = dataclasses.field(init=False)
    stored_data_end: int | None = dataclasses.field(init=False)
    storage_cost: float | None = dataclasses.field(init=False)

    def __post_init__(self):
        self.managed_data_end = self.managed_data.last_value()
        self.stored_data_end = self.stored_data.last_value()
        self.storage_cost = s3_cost(self.stored_data_end, S3CostPeriod.MONTHLY)

    @staticmethod
    def _count_items(arv_resource, *filters):
        return arv_resource().list(
            filters=filters,
            limit=0,
            count='exact',
        ).execute()['items_available']

    @classmethod
    def query(cls, arv_client, prom_client, since, to):
        cluster = arv_client.config()['ClusterID']
        def _get_metric_usage(name, resampleTo):
            metric_name = f"{name}{{cluster='{cluster}'}}"
            return get_metric_usage(prom_client, since, to, metric_name, resampleTo)

        logging.info("Getting container hours time series")
        containers = PrometheusUsageChart(
            "Concurrent running containers",
            "containers",
            _get_metric_usage("arvados_dispatchcloud_containers_running", "5min"),
        )
        logging.info("Getting data usage time series")
        managed_data = PrometheusUsageChart(
            "Data under management",
            "managed",
            _get_metric_usage("arvados_keep_collection_bytes", "60min")
        )
        stored_data = PrometheusUsageChart(
            "Storage usage",
            "used",
            _get_metric_usage("arvados_keep_total_bytes", "60min"),
        )

        total_users = cls._count_items(arv_client.users, ['is_active', '=', True])
        total_projects = cls._count_items(arv_client.groups, ['group_class', '=', 'project'])
        return cls(containers, managed_data, stored_data, total_users, total_projects)

    def iter_charts(self):
        yield self.containers
        yield self.managed_data
        yield self.stored_data

    def have_charts(self):
        return any(chart.data for chart in self.iter_charts())

    def chart_json(self):
        return markupsafe.Markup(json.dumps([
            chart.json_object()
            for chart in self.iter_charts()
            if chart.data
        ]))

    def dedup_ratio(self):
        if self.managed_data_end and self.stored_data_end:
            return self.managed_data_end / self.stored_data_end
        else:
            return None

    def monthly_storage_cost(self):
        if not self.stored_data.data:
            return None
        else:
            return sum(s3_cost(v, S3CostPeriod.HOURLY) for v in self.stored_data.iter_values())


@dataclasses.dataclass
class ProgressLogger:
    """Log progress over subsequences of a larger sequence"""
    log_fmt: str
    logger: logging.Logger = dataclasses.field(default_factory=logging.getLogger)
    loglevel: int = dataclasses.field(default=logging.INFO)
    count: int = dataclasses.field(init=False, default=0)

    def report(self, seq):
        start = self.count + 1
        self.count += len(seq)
        self.logger.log(self.loglevel, self.log_fmt, start, self.count)


def key_by_uuid(items):
    return ((item['uuid'], item) for item in items)


@dataclasses.dataclass
class ClusterActivityReport:
    since: datetime.datetime
    to: datetime.datetime
    arv_client: arv_types.ArvadosAPIClient = dataclasses.field(default_factory=arvados.api)
    prom_client: 'prometheus_api_client.PrometheusConnect | None' = None
    exclude: dataclasses.InitVar[str] = ''

    should_exclude: Callable[[str], bool] = dataclasses.field(init=False)
    # A WorkflowRunSummary for each project. The None key summarizes all runs.
    summaries: dict[str | None, WorkflowRunSummary] = dataclasses.field(init=False, default_factory=dict)
    # Statistics for each workflow. The outer key is an owner name. The inner
    # key is a workflow UUID.
    statistics: dict[str, dict[str, WorkflowRunStatistics]] = dataclasses.field(
        init=False,
        default_factory=lambda: collections.defaultdict(dict),
    )
    # Map container request UUIDs to workflow UUIDs
    workflow_map: dict[str, str] = dataclasses.field(init=False, default_factory=dict)
    # Map each type of object by UUID to the loaded API object
    groups: dict[str, arv_types.Group] = dataclasses.field(init=False, default_factory=dict)
    users: dict[str, arv_types.User] = dataclasses.field(init=False, default_factory=dict)
    workflows: dict[str, arv_types.Workflow] = dataclasses.field(init=False, default_factory=dict)
    # A chainmap of groups+users
    owners: Mapping[str, arv_types.Group | arv_types.User] = dataclasses.field(init=False)
    # UUIDs of containers that have been loaded. Used to find steps for csv_report.
    ctr_uuids: set[str] = dataclasses.field(init=False, default_factory=set)

    _request_select: ClassVar[list[str]] = [
        'container_uuid',
        'created_at',
        'cumulative_cost',
        'modified_by_user_uuid',
        'name',
        'owner_uuid',
        'requesting_container_uuid',
        'uuid',
    ]

    def __post_init__(self, exclude):
        self.owners = collections.ChainMap(self.groups, self.users)
        if exclude:
            self.should_exclude = re.compile(exclude, re.IGNORECASE).search
        else:
            self.should_exclude = lambda s: False

    def _since_filter(self):
        return ['created_at', '>=', self.since.strftime('%Y-%m-%dT00:00Z')]

    def _to_filter(self):
        return ['created_at', '<', self.to.strftime('%Y-%m-%dT00:00Z')]

    def _load_workflows(self, requests):
        new_map = {
            req['uuid']: wf_uuid
            for req in requests
            if (wf_uuid := req['properties'].get('template_uuid'))
        }
        self.workflow_map.update(new_map)
        if new_wfs := frozenset(new_map.values()).difference(self.workflows):
            self.workflows.update(key_by_uuid(arvados.util.keyset_list_all(
                self.arv_client.workflows().list,
                filters=[['uuid', 'in', list(new_wfs)]],
                select=['uuid', 'name'],
            )))

    def _load_groups_and_users(self, requests):
        user_uuids = {req['modified_by_user_uuid'] for req in requests}
        group_uuids = set()
        for req in requests:
            owner_uuid = req['owner_uuid']
            if arvados.util.user_uuid_pattern.fullmatch(owner_uuid):
                user_uuids.add(owner_uuid)
            elif arvados.util.group_uuid_pattern.fullmatch(owner_uuid):
                group_uuids.add(owner_uuid)

        if new_users := user_uuids.difference(self.users):
            self.users.update(key_by_uuid(arvados.util.keyset_list_all(
                self.arv_client.users().list,
                filters=[['uuid', 'in', list(new_users)]],
                select=['uuid', 'full_name', 'first_name', 'last_name'],
            )))
        if new_groups := group_uuids.difference(self.groups):
            self.groups.update(key_by_uuid(arvados.util.keyset_list_all(
                self.arv_client.groups().list,
                filters=[['uuid', 'in', list(new_groups)]],
                select=['uuid', 'name'],
            )))

    def _load_and_iter_runs(self, requests, containers, logger):
        """Iterate WorkflowRuns for the given sequence of requests

        `requests` is a sequence of Arvados container request records. This
        method yields a corresponding WorkflowRun for each, loading other
        resources from Arvados as needed to build them.
        """
        if not requests:
            return
        logger.report(requests)
        self._load_groups_and_users(requests)
        if new_cuuids := frozenset(req['container_uuid'] for req in requests).difference(containers):
            containers.update(key_by_uuid(arvados.util.keyset_list_all(
                self.arv_client.containers().list,
                filters=[['uuid', 'in', list(new_cuuids)]],
                select=['uuid', 'started_at', 'finished_at', 'cost'],
            )))
        for req in requests:
            container = containers[req['container_uuid']]
            # If the container doesn't have `finished_at` set, then it's some
            # aborted attempt to run, and there's not enough info to report it.
            if container['finished_at']:
                yield WorkflowRun(
                    req,
                    container,
                    owner=self.owners.get(req['owner_uuid']),
                    user=self.users.get(req['modified_by_user_uuid']),
                    workflow=self.workflows.get(self.workflow_map.get(req['uuid'])),
                )

    def _iter_runs(self):
        """Iterate all WorkflowRuns for this report"""
        containers = {}
        logger = ProgressLogger("Exporting workflow runs %s - %s")
        wf_reqs = arvados.util.keyset_list_all(
            self.arv_client.container_requests().list,
            filters=[
                ['command', 'like', '["arvados-cwl-runner"%'],
                ['container_uuid', '!=', None],
                self._since_filter(),
                self._to_filter(),
            ],
            select=self._request_select + ['properties'],
        )
        # Arvados will never return more than 1000 results to a single list
        # query. We go through requests in batches of 999 to maximize the
        # chances that queries for associated resources will fit in a single
        # page of results.
        while new_reqs := list(itertools.islice(wf_reqs, 999)):
            self._load_workflows(new_reqs)
            yield from self._load_and_iter_runs(new_reqs, containers, logger)

    def iter_and_tally_runs(self):
        """Iterate, filter, and build summaries for top-level WorkflowRuns"""
        if self.owners:
            return
        self.summaries[None] = WorkflowRunSummary(None)
        for run in self._iter_runs():
            run_key = run.name_key()
            if self.should_exclude(run_key):
                continue
            yield run
            self.ctr_uuids.add(run.request['container_uuid'])
            self.summaries[None].tally(run)

            owner_uuid = run.request['owner_uuid']
            owner_name = run.owner_name()
            try:
                summary = self.summaries[owner_name]
            except KeyError:
                summary = self.summaries[owner_name] = WorkflowRunSummary(owner_uuid)
            summary.tally(run)

            wf_uuid = run.template_uuid()
            try:
                stats = self.statistics[owner_name][wf_uuid]
            except KeyError:
                stats = self.statistics[owner_name][wf_uuid] = WorkflowRunStatistics(owner_uuid, wf_uuid)
            stats.tally(run)

    def iter_steps(self):
        logging.info("Getting workflow steps")
        containers = {}
        logger = ProgressLogger("Got workflow steps %s - %s")
        seen_cuuids = set()
        while new_cuuids := self.ctr_uuids.difference(seen_cuuids):
            # The ideal slice length here really depends on how many steps
            # the typical workflow starts. 50 is just a fuzzy attempt to
            # balance batching queries while keeping result sizes reasonable.
            # It could be adjusted if better data suggests that's a good idea.
            cuuid_batch = list(itertools.islice(new_cuuids, 50))
            new_reqs = list(arvados.util.keyset_list_all(
                self.arv_client.container_requests().list,
                filters=[
                    ['requesting_container_uuid', 'in', cuuid_batch],
                    ['container_uuid', '!=', None],
                ],
                select=self._request_select,
            ))
            self.workflow_map.update(
                (req['uuid'], wf_uuid)
                for req in new_reqs
                if (wf_uuid := self.workflow_map.get(req['requesting_container_uuid']))
            )
            yield from self._load_and_iter_runs(new_reqs, containers, logger)
            seen_cuuids.update(cuuid_batch)

    def html_report(self):
        """Get a cluster activity report for the desired time period,
        returning a string containing the report as an HTML document."""
        for _ in self.iter_and_tally_runs():
            # HTML doesn't report individual runs, just summaries.
            pass

        arv_config = self.arv_client.config()
        active = self.summaries.pop(None)
        cluster = ClusterSummary.query(self.arv_client, self.prom_client, self.since, self.to)
        summaries = sorted(
            self.summaries.items(),
            key=lambda kv: kv[1].total_cost(),
            reverse=True,
        )
        statistics = {key: sorted(
            workflows.items(),
            key=lambda kv: kv[1].total_runs(),
            reverse=True,
        ) for key, workflows in self.statistics.items()}

        jinja = jinja2.Environment(
            autoescape=jinja2.select_autoescape(),
            loader=jinja2.PackageLoader('arvados_cluster_activity'),
            trim_blocks=True,
        )
        jinja.filters.update(_JINJA_FILTERS)
        wb_url = arv_config['Services']['Workbench2']['ExternalURL']
        def workbench_url(value):
            if not isinstance(value, str):
                value = PurePath(*value).as_posix()
            return markupsafe.Markup(urllib.parse.urljoin(wb_url, value))
        jinja.filters['workbench_url'] = workbench_url

        return jinja.get_template('report.html.j2').render(
            active=active,
            cluster=cluster,
            cluster_id=arv_config['ClusterID'],
            report=self,
            report_span=datespan_fmt(self.since, self.to),
            statistics=statistics,
            summaries=summaries,
            today=datetime.date.today(),
        )

    def csv_report(self, out, columns, *, include_steps: bool):
        if not columns:
            if include_steps:
                columns = (
                    "Project", "Workflow", "Step",
                    "Sample", "User", "Submitted", "Runtime", "Cost"
                )
            else:
                columns = (
                    "Project", "Workflow",
                    "Sample", "User", "Submitted", "Runtime", "CumulativeCost"
                )

        csvwriter = csv.DictWriter(out, fieldnames=columns, extrasaction="ignore")
        csvwriter.writeheader()
        csvwriter.writerows(run.csv_row() for run in self.iter_and_tally_runs())
        if include_steps:
            csvwriter.writerows(run.csv_row() for run in self.iter_steps())
