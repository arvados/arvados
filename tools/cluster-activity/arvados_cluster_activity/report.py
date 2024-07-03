#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import logging
import ciso8601
import arvados.util
import re
import csv
import math
import collections
import json
from datetime import date, datetime, timedelta
import pkg_resources

from dataclasses import dataclass

import crunchstat_summary.dygraphs
from crunchstat_summary.summarizer import Task

from arvados_cluster_activity.prometheus import get_metric_usage, get_data_usage

sortablecss = """
@charset "UTF-8";
.sortable thead th:not(.no-sort) {
  cursor: pointer;
}
.sortable thead th:not(.no-sort)::after, .sortable thead th:not(.no-sort)::before {
  transition: color 0.1s ease-in-out;
  font-size: 1.2em;
  color: transparent;
}
.sortable thead th:not(.no-sort)::after {
  margin-left: 3px;
  content: "▸";
}
.sortable thead th:not(.no-sort):hover::after {
  color: inherit;
}
.sortable thead th:not(.no-sort)[aria-sort=descending]::after {
  color: inherit;
  content: "▾";
}
.sortable thead th:not(.no-sort)[aria-sort=ascending]::after {
  color: inherit;
  content: "▴";
}
.sortable thead th:not(.no-sort).indicator-left::after {
  content: "";
}
.sortable thead th:not(.no-sort).indicator-left::before {
  margin-right: 3px;
  content: "▸";
}
.sortable thead th:not(.no-sort).indicator-left:hover::before {
  color: inherit;
}
.sortable thead th:not(.no-sort).indicator-left[aria-sort=descending]::before {
  color: inherit;
  content: "▾";
}
.sortable thead th:not(.no-sort).indicator-left[aria-sort=ascending]::before {
  color: inherit;
  content: "▴";
}

/*# sourceMappingURL=sortable.css.map */
"""

@dataclass
class WorkflowRunSummary:
    name: str
    uuid: str
    count: int = 0
    cost: float = 0
    hours: float = 0

@dataclass
class ProjectSummary:
    users: set
    uuid: str
    runs: dict[str, WorkflowRunSummary]
    earliest: datetime = datetime(year=9999, month=1, day=1)
    latest: datetime = datetime(year=1900, month=1, day=1)
    name: str = ""
    cost: float = 0
    count: int = 0
    hours: float = 0
    activityspan: str = ""
    tablerow: str = ""

@dataclass
class Summarizer:
    label: str
    tasks: collections.defaultdict[str, Task]

    def long_label(self):
        return self.label


def date_export(item):
    if isinstance(item, datetime):
        return """@new Date("{}")@""".format(item.strftime("%Y-%m-%dT%H:%M:%SZ"))

def aws_monthly_cost(value):
    value_gb = value / (1024*1024*1024)
    first_50tb = min(1024*50, value_gb)
    next_450tb = max(min(1024*450, value_gb-1024*50), 0)
    over_500tb = max(value_gb-1024*500, 0)

    monthly_cost = (first_50tb * 0.023) + (next_450tb * 0.022) + (over_500tb * 0.021)
    return monthly_cost


def format_with_suffix_base2(summary_value):
    for scale in ["KiB", "MiB", "GiB", "TiB", "PiB", "EiB"]:
        summary_value = summary_value / 1024
        if summary_value < 1024:
            return "%.3f %s" % (summary_value, scale)

def format_with_suffix_base10(summary_value):
    for scale in ["KB", "MB", "GB", "TB", "PB", "EB"]:
        summary_value = summary_value / 1000
        if summary_value < 1000:
            return "%.3f %s" % (summary_value, scale)

containers_category = 'Concurrent running containers'
storage_category = 'Storage Usage'

class ReportChart(crunchstat_summary.dygraphs.DygraphsChart):
    def __init__(self, label, summarizers):
        super(ReportChart, self).__init__(label, summarizers)
        self.label = label
        self.summarizers = summarizers
        self.STYLE = '\n'.join((self.STYLE, sortablecss))

    JSASSETS = ['dygraphs.js', 'sortable.js']

    def sections(self):
        return [
            {
                'label': s.long_label(),
                'charts': [
                    self.chartdata(s.label, s.tasks, stat)
                    for stat in ((containers_category, ['containers']),
                                 (storage_category, ['storage used']),
                                 )
                    ],
            }
            for s in self.summarizers]

    def js(self):
        return 'var chartdata = {};\n{}{}'.format(
            json.dumps(self.sections(), default=date_export).replace('"@', '').replace('@"', '').replace('\\"', '"'),
            '\n'.join([pkg_resources.resource_string('crunchstat_summary', 'synchronizer.js').decode('utf-8')]),
            '\n'.join([pkg_resources.resource_string('arvados_cluster_activity', jsa).decode('utf-8') for jsa in self.JSASSETS]))

    def _collate_data(self, tasks, stats):
        data = []
        nulls = []
        # uuid is category for crunch2
        for uuid, task in tasks.items():
            # All stats in a category are assumed to have the same time base and same number of samples
            category = stats[0]
            series_names = stats[1]
            sn0 = series_names[0]
            series = task.series[(category,sn0)]
            for i in range(len(series)):
                pt = series[i]
                vals = [task.series[(category,stat)][i][1] for stat in series_names[1:]]
                data.append([pt[0]] + nulls + [pt[1]] + vals)
            nulls.append(None)
        return sorted(data)


WEBCHART_CLASS = ReportChart


def runtime_str(container_request, containers):
    length = ciso8601.parse_datetime(containers[container_request["container_uuid"]]["finished_at"]) - ciso8601.parse_datetime(containers[container_request["container_uuid"]]["started_at"])

    hours = length.days * 24 + (length.seconds // 3600)
    minutes = (length.seconds // 60) % 60
    seconds = length.seconds % 60

    return "%i:%02i:%02i" % (hours, minutes, seconds)

def runtime_in_hours(runtime):
    sp = runtime.split(":")
    hours = float(sp[0])
    hours += float(sp[1]) / 60
    hours += float(sp[2]) / 3600
    return hours

def hours_to_runtime_str(frac_hours):
    hours = math.floor(frac_hours)
    minutes = (frac_hours - math.floor(frac_hours)) * 60.0
    seconds = (minutes - math.floor(minutes)) * 60.0

    return "%i:%02i:%02i" % (hours, minutes, seconds)


def csv_dateformat(datestr):
    dt = ciso8601.parse_datetime(datestr)
    return dt.strftime("%Y-%m-%d %H:%M:%S")


class ClusterActivityReport(object):
    def __init__(self, prom_client, label=None, threads=1, **kwargs):
        self.threadcount = threads
        self.arv_client = arvados.api()
        self.prom_client = prom_client
        self.cluster = self.arv_client.config()["ClusterID"]

        self.active_users = set()
        self.project_summary = {}
        self.total_hours = 0
        self.total_cost = 0
        self.total_workflows = 0
        self.summarizers = []
        self.storage_cost = 0
        self.summary_fetched = False

    def run(self):
        pass

    def collect_graph(self, s1, since, to, taskname, legend, metric, resampleTo, extra=None):
        if not self.prom_client:
            return

        task = s1.tasks[taskname]

        for series in get_metric_usage(self.prom_client, since, to, metric % self.cluster, resampleTo=resampleTo):
            for t in series.itertuples():
                task.series[taskname, legend].append(t)
                if extra:
                    extra(t)

    def collect_storage_cost(self, t):
        self.storage_cost += aws_monthly_cost(t.y) / (30*24)

    def html_report(self, since, to, exclude, include_workflow_steps):

        self.label = "Cluster report for %s from %s to %s" % (self.cluster, since.date(), to.date())

        # If we already produced a CSV report we have summary stats
        # and don't need to fetch everything all over again.
        if not self.summary_fetched:
            for row in self.report_from_api(since, to, include_workflow_steps, exclude):
                pass

        container_cumulative_hours = 0
        def collect_container_hours(t):
            nonlocal container_cumulative_hours
            # resampled to 5 minute increments but we want
            # a sum of hours
            container_cumulative_hours += t.y / 12

        logging.info("Getting container hours time series")
        s1 = Summarizer(label="", tasks=collections.defaultdict(Task))
        self.collect_graph(s1, since, to, containers_category, "containers",
                           "arvados_dispatchcloud_containers_running{cluster='%s'}",
                           resampleTo="5min",
                           extra=collect_container_hours
                           )

        logging.info("Getting data usage time series")
        s2 = Summarizer(label="", tasks=collections.defaultdict(Task))
        self.collect_graph(s2, since, to, storage_category, "managed",
                           "arvados_keep_collection_bytes{cluster='%s'}", resampleTo="60min")

        self.collect_graph(s2, since, to, storage_category, "storage used",
                           "arvados_keep_total_bytes{cluster='%s'}", resampleTo="60min", extra=self.collect_storage_cost)

        managed_data_now = None
        storage_used_now = None

        if len(s2.tasks[storage_category].series[storage_category,"managed"]) > 0:
            managed_data_now = s2.tasks[storage_category].series[storage_category,"managed"][-1]
        if len(s2.tasks[storage_category].series[storage_category,"storage used"]) > 0:
            storage_used_now = s2.tasks[storage_category].series[storage_category,"storage used"][-1]

        if managed_data_now and storage_used_now:
            storage_cost = aws_monthly_cost(storage_used_now.y)
            dedup_ratio = managed_data_now.y/storage_used_now.y
            dedup_savings = aws_monthly_cost(managed_data_now.y) - storage_cost

        if self.prom_client:
            self.summarizers = [s1, s2]
        else:
            self.summarizers = []

        tophtml = ""
        bottomhtml = ""
        label = self.label

        tophtml = []

        workbench = self.arv_client.config()["Services"]["Workbench2"]["ExternalURL"]
        if workbench.endswith("/"):
            workbench = workbench[:-1]

        if to.date() == date.today():

            data_rows = ""
            if managed_data_now and storage_used_now:
                data_rows = """
            <tr><td>Total data under management</td><td>{managed_data_now}</td></tr>
            <tr><td>Total storage usage</td><td>{storage_used_now}</td></tr>
            <tr><td>Deduplication ratio</td><td>{dedup_ratio:.1f}</td></tr>
            <tr><td>Approximate monthly storage cost</td><td>${storage_cost:,.2f}</td></tr>
            <tr><td>Monthly savings from storage deduplication</td><td>${dedup_savings:,.2f}</td></tr>
                """.format(
                       managed_data_now=format_with_suffix_base10(managed_data_now.y),
                       storage_used_now=format_with_suffix_base10(storage_used_now.y),
                       dedup_savings=dedup_savings,
                       storage_cost=storage_cost,
                       dedup_ratio=dedup_ratio,
                )

            tophtml.append("""<h2>Cluster status as of {now}</h2>
            <table class='aggtable'><tbody>
            <tr><td><a href="{workbench}/users">Total users</a></td><td>{total_users}</td></tr>
            <tr><td>Total projects</td><td>{total_projects}</td></tr>
            {data_rows}
            </tbody></table>
            """.format(now=date.today(),
                       total_users=self.total_users,
                       total_projects=self.total_projects,
                       workbench=workbench,
                       data_rows=data_rows))

        tophtml.append("""<h2>Activity and cost over the {reporting_days} day period {since} to {to}</h2>
        <table class='aggtable'><tbody>
        <tr><td>Active users</td><td>{active_users}</td></tr>
        <tr><td><a href="#Active_Projects">Active projects</a></td><td>{active_projects}</td></tr>
        <tr><td>Workflow runs</td><td>{total_workflows:,}</td></tr>
        <tr><td>Compute used</td><td>{total_hours:,.1f} hours</td></tr>
        <tr><td>Compute cost</td><td>${total_cost:,.2f}</td></tr>
        <tr><td>Storage cost</td><td>${storage_cost:,.2f}</td></tr>
        </tbody></table>
        """.format(active_users=len(self.active_users),
                   total_users=self.total_users,
                   #total_hours=container_cumulative_hours,
                   total_hours=self.total_hours,
                   total_cost=self.total_cost,
                   total_workflows=self.total_workflows,
                   active_projects=len(self.project_summary),
                   since=since.date(), to=to.date(),
                   reporting_days=(to - since).days,
                   storage_cost=self.storage_cost))

        bottomhtml = []

        projectlist = sorted(self.project_summary.items(), key=lambda x: x[1].cost, reverse=True)

        for k, prj in projectlist:
            if prj.earliest.date() == prj.latest.date():
                prj.activityspan = "{}".format(prj.earliest.date())
            else:
                prj.activityspan = "{} to {}".format(prj.earliest.date(), prj.latest.date())

            prj.tablerow = """<td>{users}</td> <td>{active}</td> <td>{hours:,.1f}</td> <td>${cost:,.2f}</td>""".format(
                name=prj.name,
                active=prj.activityspan,
                cost=prj.cost,
                hours=prj.hours,
                users=", ".join(prj.users),
            )

        bottomhtml.append(
            """
            <a id="Active_Projects"><h2>Active Projects</h2></a>
            <table class='sortable'>
            <thead><tr><th>Project</th> <th>Users</th> <th>Active</th> <th>Compute usage (hours)</th> <th>Compute cost</th> </tr></thead>
            <tbody><tr>{projects}</tr></tbody>
            </table>
            """.format(projects="</tr>\n<tr>".join("""<td><a href="#{name}">{name}</a></td>{rest}""".format(name=prj.name, rest=prj.tablerow) for k, prj in projectlist)))

        for k, prj in projectlist:
            wfsum = []
            for k2, r in sorted(prj.runs.items(), key=lambda x: x[1].count, reverse=True):
                wfsum.append("""
                <tr><td>{count}</td> <td>{workflowlink}</td>  <td>{runtime}</td> <td>${cost:,.2f}</td> <td>${totalcost:,.2f}</td></tr>
                """.format(
                    count=r.count,
                    runtime=hours_to_runtime_str(r.hours/r.count),
                    cost=r.cost/r.count,
                    totalcost=r.cost,
                    workflowlink="""<a href="{workbench}/workflows/{uuid}">{name}</a>""".format(workbench=workbench,uuid=r.uuid,name=r.name)
                    if r.uuid != "none" else r.name))

                # <table>
                # <thead><tr><th>Users</th> <th>Active</th> <th>Compute usage</th> <th>Compute cost</th> </tr></thead>
                # <tbody><tr>{projectrow}</tr></tbody>
                # </table>


            bottomhtml.append(
                """<a id="{name}"></a><a href="{workbench}/projects/{uuid}"><h2>{name}</h2></a>

                <table>
                <tbody><tr>{projectrow}</tr></tbody>
                </table>

                <table class='sortable'>
                <thead><tr><th>Workflow run count</th> <th>Workflow name</th> <th>Mean runtime</th> <th>Mean cost per run</th> <th>Sum cost over runs</th></tr></thead>
                <tbody>
                {wfsum}
                </tbody></table>
                """.format(name=prj.name,
                           users=", ".join(prj.users),
                           cost=prj.cost,
                           hours=prj.hours,
                           wfsum=" ".join(wfsum),
                           earliest=prj.earliest.date(),
                           latest=prj.latest.date(),
                           activity=prj.activityspan,
                           userplural='s' if len(prj.users) > 1 else '',
                           projectrow=prj.tablerow,
                           workbench=workbench,
                           uuid=prj.uuid)
            )

        return WEBCHART_CLASS(label, self.summarizers).html(tophtml, bottomhtml)

    def flush_containers(self, pending, include_steps, exclude):
        containers = {}

        for container in arvados.util.keyset_list_all(
            self.arv_client.containers().list,
            filters=[
                ["uuid", "in", [c["container_uuid"] for c in pending if c["container_uuid"]]],
            ],
            select=["uuid", "started_at", "finished_at", "cost"]):

            containers[container["uuid"]] = container

        workflows = {}
        workflows["none"] = "workflow run from command line"

        for wf in arvados.util.keyset_list_all(
                self.arv_client.workflows().list,
                filters=[
                    ["uuid", "in", list(set(c["properties"]["template_uuid"]
                                            for c in pending
                                            if "template_uuid" in c["properties"] and c["properties"]["template_uuid"].startswith(self.arv_client.config()["ClusterID"])))],
                ],
                select=["uuid", "name"]):
            workflows[wf["uuid"]] = wf["name"]

        projects = {}

        for pr in arvados.util.keyset_list_all(
                self.arv_client.groups().list,
                filters=[
                    ["uuid", "in", list(set(c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'j7d0g'))],
                ],
                select=["uuid", "name"]):
            projects[pr["uuid"]] = pr["name"]

        for pr in arvados.util.keyset_list_all(
                self.arv_client.users().list,
                filters=[
                    ["uuid", "in", list(set(c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'tpzed')|set(c["modified_by_user_uuid"] for c in pending))],
                ],
                select=["uuid", "full_name", "first_name", "last_name"]):
            projects[pr["uuid"]] = pr["full_name"]

        if include_steps:
            name_regex = re.compile(r"(.+)_[0-9]+")
            child_crs = {}
            child_cr_containers = set()
            stepcount = 0

            logging.info("Getting workflow steps")
            for cr in arvados.util.keyset_list_all(
                self.arv_client.container_requests().list,
                filters=[
                    ["requesting_container_uuid", "in", list(containers.keys())],
                ],
                select=["uuid", "name", "cumulative_cost", "requesting_container_uuid", "container_uuid"]):

                if cr["cumulative_cost"] == 0:
                    continue

                g = name_regex.fullmatch(cr["name"])
                if g:
                    cr["name"] = g[1]

                child_crs.setdefault(cr["requesting_container_uuid"], []).append(cr)
                child_cr_containers.add(cr["container_uuid"])
                if len(child_cr_containers) == 1000:
                    stepcount += len(child_cr_containers)
                    for container in arvados.util.keyset_list_all(
                            self.arv_client.containers().list,
                            filters=[
                                ["uuid", "in", list(child_cr_containers)],
                            ],
                            select=["uuid", "started_at", "finished_at", "cost"]):

                        containers[container["uuid"]] = container

                    logging.info("Got workflow steps %s - %s", stepcount-len(child_cr_containers), stepcount)
                    child_cr_containers.clear()

            if child_cr_containers:
                stepcount += len(child_cr_containers)
                for container in arvados.util.keyset_list_all(
                        self.arv_client.containers().list,
                        filters=[
                            ["uuid", "in", list(child_cr_containers)],
                        ],
                        select=["uuid", "started_at", "finished_at", "cost"]):

                    containers[container["uuid"]] = container
                logging.info("Got workflow steps %s - %s", stepcount-len(child_cr_containers), stepcount)

        for container_request in pending:
            if not container_request["container_uuid"] or not containers[container_request["container_uuid"]]["started_at"] or not containers[container_request["container_uuid"]]["finished_at"]:
                continue

            template_uuid = container_request["properties"].get("template_uuid", "none")
            workflowname = container_request["name"] if template_uuid == "none" else workflows.get(template_uuid, template_uuid)

            if exclude and re.search(exclude, workflowname, flags=re.IGNORECASE):
                continue

            yield {
                "Project": projects.get(container_request["owner_uuid"], "unknown owner"),
                "ProjectUUID": container_request["owner_uuid"],
                "Workflow": workflowname,
                "WorkflowUUID": container_request["properties"].get("template_uuid", "none"),
                "Step": "workflow runner",
                "StepUUID": container_request["uuid"],
                "Sample": container_request["name"],
                "SampleUUID": container_request["uuid"],
                "User": projects.get(container_request["modified_by_user_uuid"], "unknown user"),
                "UserUUID": container_request["modified_by_user_uuid"],
                "Submitted": csv_dateformat(container_request["created_at"]),
                "Started": csv_dateformat(containers[container_request["container_uuid"]]["started_at"]),
                "Finished": csv_dateformat(containers[container_request["container_uuid"]]["finished_at"]),
                "Runtime": runtime_str(container_request, containers),
                "Cost": round(containers[container_request["container_uuid"]]["cost"] if include_steps else container_request["cumulative_cost"], 3),
                "CumulativeCost": round(container_request["cumulative_cost"], 3)
                }

            if include_steps:
                for child_cr in child_crs.get(container_request["container_uuid"], []):
                    if not child_cr["container_uuid"] or not containers[child_cr["container_uuid"]]["started_at"] or not containers[child_cr["container_uuid"]]["finished_at"]:
                        continue
                    yield {
                        "Project": projects.get(container_request["owner_uuid"], "unknown owner"),
                        "ProjectUUID": container_request["owner_uuid"],
                        "Workflow": workflows.get(container_request["properties"].get("template_uuid", "none"), "workflow missing"),
                        "WorkflowUUID": container_request["properties"].get("template_uuid", "none"),
                        "Step": child_cr["name"],
                        "StepUUID": child_cr["uuid"],
                        "Sample": container_request["name"],
                        "SampleUUID": container_request["name"],
                        "User": projects.get(container_request["modified_by_user_uuid"], "unknown user"),
                        "UserUUID": container_request["modified_by_user_uuid"],
                        "Submitted": csv_dateformat(child_cr["created_at"]),
                        "Started": csv_dateformat(containers[child_cr["container_uuid"]]["started_at"]),
                        "Finished": csv_dateformat(containers[child_cr["container_uuid"]]["finished_at"]),
                        "Runtime": runtime_str(child_cr, containers),
                        "Cost": round(containers[child_cr["container_uuid"]]["cost"], 3),
                        "CumulativeCost": round(containers[child_cr["container_uuid"]]["cost"], 3),
                        }


    def collect_summary_stats(self, row):
        self.active_users.add(row["User"])
        self.project_summary.setdefault(row["ProjectUUID"],
                                        ProjectSummary(users=set(),
                                                       runs={},
                                                       uuid=row["ProjectUUID"],
                                                       name=row["Project"]))
        prj = self.project_summary[row["ProjectUUID"]]
        cost = row["Cost"]
        prj.cost += cost
        prj.count += 1
        prj.users.add(row["User"])
        hrs = runtime_in_hours(row["Runtime"])
        prj.hours += hrs

        started = datetime.strptime(row["Started"], "%Y-%m-%d %H:%M:%S")
        finished = datetime.strptime(row["Finished"], "%Y-%m-%d %H:%M:%S")

        if started < prj.earliest:
            prj.earliest = started

        if finished > prj.latest:
            prj.latest = finished

        if row["Step"] == "workflow runner":
            prj.runs.setdefault(row["Workflow"], WorkflowRunSummary(name=row["Workflow"],
                                                                    uuid=row["WorkflowUUID"]))
            wfuuid = row["Workflow"]
            prj.runs[wfuuid].count += 1
            prj.runs[wfuuid].cost += row["CumulativeCost"]
            prj.runs[wfuuid].hours += hrs
            self.total_workflows += 1

        self.total_hours += hrs
        self.total_cost += cost

    def report_from_api(self, since, to, include_steps, exclude):
        pending = []

        count = 0
        for container_request in arvados.util.keyset_list_all(
                self.arv_client.container_requests().list,
                filters=[
                    ["command", "like", "[\"arvados-cwl-runner%"],
                    ["created_at", ">=", since.strftime("%Y%m%dT%H%M%SZ")],
                ],
                select=["uuid", "owner_uuid", "container_uuid", "name", "cumulative_cost", "properties", "modified_by_user_uuid", "created_at"]):

            if container_request["cumulative_cost"] == 0:
                continue

            if len(pending) < 1000:
                pending.append(container_request)
            else:
                count += len(pending)
                logging.info("Exporting workflow runs %s - %s", count-len(pending), count)
                for row in self.flush_containers(pending, include_steps, exclude):
                    self.collect_summary_stats(row)
                    yield row
                pending.clear()

        count += len(pending)
        logging.info("Exporting workflow runs %s - %s", count-len(pending), count)
        for row in self.flush_containers(pending, include_steps, exclude):
            self.collect_summary_stats(row)
            yield row

        userinfo = self.arv_client.users().list(filters=[["is_active", "=", True]], limit=0).execute()
        self.total_users = userinfo["items_available"]

        groupinfo = self.arv_client.groups().list(filters=[["group_class", "=", "project"]], limit=0).execute()
        self.total_projects = groupinfo["items_available"]

    def csv_report(self, since, to, out, include_steps, columns, exclude):
        if columns:
            columns = columns.split(",")
        else:
            if include_steps:
                columns = ("Project", "Workflow", "Step", "Sample", "User", "Submitted", "Runtime", "Cost")
            else:
                columns = ("Project", "Workflow", "Sample", "User", "Submitted", "Runtime", "Cost")

        csvwriter = csv.DictWriter(out, fieldnames=columns, extrasaction="ignore")
        csvwriter.writeheader()

        for row in self.report_from_api(since, to, include_steps, exclude):
            csvwriter.writerow(row)

        self.summary_fetched = True
