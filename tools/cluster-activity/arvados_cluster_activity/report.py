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
from typing import Dict, List
import statistics

from dataclasses import dataclass
from arvados_cluster_activity.prometheus import get_metric_usage, get_data_usage
from arvados_cluster_activity.reportchart import ReportChart


@dataclass
class WorkflowRunSummary:
    name: str
    uuid: str
    cost: List[float]
    hours: List[float]
    count: int = 0


@dataclass
class ProjectSummary:
    users: set
    uuid: str
    runs: Dict[str, WorkflowRunSummary]
    earliest: datetime = datetime(year=9999, month=1, day=1)
    latest: datetime = datetime(year=1900, month=1, day=1)
    name: str = ""
    cost: float = 0
    count: int = 0
    hours: float = 0
    activityspan: str = ""
    tablerow: str = ""


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

containers_graph = ('Concurrent running containers', 'containers')
storage_graph = ('Storage usage', 'used')
managed_graph = ('Data under management', 'managed')


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
    def __init__(self, prom_client):
        self.arv_client = arvados.api()
        self.prom_client = prom_client
        self.cluster = self.arv_client.config()["ClusterID"]

        self.active_users = set()
        self.project_summary = {}
        self.total_hours = 0
        self.total_cost = 0
        self.total_workflows = 0
        self.storage_cost = 0
        self.summary_fetched = False
        self.graphs = {}

    def collect_graph(self, since, to, metric, resample_to, extra=None):
        if not self.prom_client:
            return

        flatdata = []

        for series in get_metric_usage(self.prom_client, since, to, metric % self.cluster, resampleTo=resample_to):
            for t in series.itertuples():
                flatdata.append([t[0], t[1]])
                if extra:
                    extra(t[0], t[1])

        return flatdata

    def collect_storage_cost(self, timestamp, value):
        self.storage_cost += aws_monthly_cost(value) / (30*24)

    def html_report(self, since, to, exclude, include_workflow_steps):
        """Get a cluster activity report for the desired time period,
        returning a string containing the report as an HTML document."""

        self.label = "Cluster report for %s from %s to %s" % (self.cluster, since.date(), to.date())

        if not self.summary_fetched:
            # If we haven't done it already, need to fetch everything
            # from the API to collect summary stats (report_from_api
            # calls collect_summary_stats on each row).
            #
            # Because it is a Python generator, we need call it in a
            # loop to process all the rows.  This method also yields
            # each row which is used by a different function to create
            # the CSV report, but for the HTML report we just discard
            # them.
            for row in self.report_from_api(since, to, include_workflow_steps, exclude):
                pass

        container_cumulative_hours = 0
        def collect_container_hours(timestamp, value):
            nonlocal container_cumulative_hours
            # resampled to 5 minute increments but we want
            # a sum of hours
            container_cumulative_hours += value / 12

        logging.info("Getting container hours time series")

        self.graphs[containers_graph] = self.collect_graph(since, to,
                           "arvados_dispatchcloud_containers_running{cluster='%s'}",
                           resample_to="5min",
                           extra=collect_container_hours
                           )

        logging.info("Getting data usage time series")
        self.graphs[managed_graph] = self.collect_graph(since, to,
                           "arvados_keep_collection_bytes{cluster='%s'}", resample_to="60min")

        self.graphs[storage_graph] = self.collect_graph(since, to,
                           "arvados_keep_total_bytes{cluster='%s'}", resample_to="60min",
                                                        extra=self.collect_storage_cost)

        label = self.label

        cards = []

        workbench = self.arv_client.config()["Services"]["Workbench2"]["ExternalURL"]
        if workbench.endswith("/"):
            workbench = workbench[:-1]

        if to.date() == self.today():
            # The deduplication ratio overstates things a bit, you can
            # have collections which reference a small slice of a large
            # block, and this messes up the intuitive value of this ratio
            # and exagerates the effect.
            #
            # So for now, as much fun as this is, I'm excluding it from
            # the report.
            #
            # dedup_savings = aws_monthly_cost(managed_data_now) - storage_cost
            # <tr><th>Monthly savings from storage deduplication</th> <td>${dedup_savings:,.2f}</td></tr>

            data_rows = ""
            if self.graphs[managed_graph] and self.graphs[storage_graph]:
                managed_data_now = self.graphs[managed_graph][-1][1]
                storage_used_now = self.graphs[storage_graph][-1][1]
                data_rows = """
            <tr><th>Total data under management</th> <td>{managed_data_now}</td></tr>
            <tr><th>Total storage usage</th> <td>{storage_used_now}</td></tr>
            <tr><th>Deduplication ratio</th> <td>{dedup_ratio:.1f}</td></tr>
            <tr><th>Approximate monthly storage cost</th> <td>${storage_cost:,.2f}</td></tr>
                """.format(
                       managed_data_now=format_with_suffix_base10(managed_data_now),
                       storage_used_now=format_with_suffix_base10(storage_used_now),
                       storage_cost=aws_monthly_cost(storage_used_now),
                       dedup_ratio=managed_data_now / storage_used_now,
                )

            cards.append("""<h2>Cluster status as of {now}</h2>
            <table class='aggtable'><tbody>
            <tr><th><a href="{workbench}/users">Total users</a></th><td>{total_users}</td></tr>
            <tr><th>Total projects</th><td>{total_projects}</td></tr>
            {data_rows}
            </tbody></table>
            <p>See <a href="#prices">note on usage and cost calculations</a> for details on how costs are calculated.</p>
            """.format(now=self.today(),
                       total_users=self.total_users,
                       total_projects=self.total_projects,
                       workbench=workbench,
                       data_rows=data_rows))

        # We have a couple of options for getting total container hours
        #
        # total_hours=container_cumulative_hours
        #
        # calculates the sum from prometheus metrics
        #
        # total_hours=self.total_hours
        #
        # calculates the sum of the containers that were fetched
        #
        # The problem is these numbers tend not to match, especially
        # if the report generation was not called with "include
        # workflow steps".
        #
        # I decided to use the sum from containers fetched, because it
        # will match the sum of compute time for each project listed
        # in the report.

        cards.append("""<h2>Activity and cost over the {reporting_days} day period {since} to {to}</h2>
        <table class='aggtable'><tbody>
        <tr><th>Active users</th> <td>{active_users}</td></tr>
        <tr><th><a href="#Active_Projects">Active projects</a></th> <td>{active_projects}</td></tr>
        <tr><th>Workflow runs</th> <td>{total_workflows:,}</td></tr>
        <tr><th>Compute used</th> <td>{total_hours:,.1f} hours</td></tr>
        <tr><th>Compute cost</th> <td>${total_cost:,.2f}</td></tr>
        <tr><th>Storage cost</th> <td>${storage_cost:,.2f}</td></tr>
        </tbody></table>
        <p>See <a href="#prices">note on usage and cost calculations</a> for details on how costs are calculated.</p>
        """.format(active_users=len(self.active_users),
                   total_users=self.total_users,
                   total_hours=self.total_hours,
                   total_cost=self.total_cost,
                   total_workflows=self.total_workflows,
                   active_projects=len(self.project_summary),
                   since=since.date(), to=to.date(),
                   reporting_days=(to - since).days,
                   storage_cost=self.storage_cost))

        projectlist = sorted(self.project_summary.items(), key=lambda x: x[1].cost, reverse=True)

        for k, prj in projectlist:
            if prj.earliest.date() == prj.latest.date():
                prj.activityspan = "{}".format(prj.earliest.date())
            else:
                prj.activityspan = "{} to {}".format(prj.earliest.date(), prj.latest.date())

            prj.tablerow = """<td>{users}</td> <td>{active}</td> <td>{hours:,.1f}</td> <td>${cost:,.2f}</td>""".format(
                active=prj.activityspan,
                cost=prj.cost,
                hours=prj.hours,
                users=", ".join(prj.users),
            )

        if any(self.graphs.values()):
            cards.append("""
                <div id="chart"></div>
            """)

        cards.append(
            """
            <a id="Active_Projects"><h2>Active Projects</h2></a>
            <table class='sortable active-projects'>
            <thead><tr><th>Project</th> <th>Users</th> <th>Active</th> <th>Compute usage (hours)</th> <th>Compute cost</th> </tr></thead>
            <tbody><tr>{projects}</tr></tbody>
            </table>
            <p>See <a href="#prices">note on usage and cost calculations</a> for details on how costs are calculated.</p>
            """.format(projects="</tr>\n<tr>".join("""<td><a href="#{name}">{name}</a></td>{rest}""".format(name=prj.name, rest=prj.tablerow) for k, prj in projectlist)))

        for k, prj in projectlist:
            wfsum = []
            for k2, r in sorted(prj.runs.items(), key=lambda x: x[1].count, reverse=True):
                wfsum.append("""
                <tr><td>{count}</td> <td>{workflowlink}</td> <td>{median_runtime}</td> <td>{mean_runtime}</td> <td>${median_cost:,.2f}</td> <td>${mean_cost:,.2f}</td> <td>${totalcost:,.2f}</td></tr>
                """.format(
                    count=r.count,
                    mean_runtime=hours_to_runtime_str(statistics.mean(r.hours)),
                    median_runtime=hours_to_runtime_str(statistics.median(r.hours)),
                    mean_cost=statistics.mean(r.cost),
                    median_cost=statistics.median(r.cost),
                    totalcost=sum(r.cost),
                    workflowlink="""<a href="{workbench}/workflows/{uuid}">{name}</a>""".format(workbench=workbench,uuid=r.uuid,name=r.name)
                    if r.uuid != "none" else r.name))

            cards.append(
                """<a id="{name}"></a><a href="{workbench}/projects/{uuid}"><h2>{name}</h2></a>

                <table class='sortable single-project'>
                <thead><tr> <th>Users</th> <th>Active</th> <th>Compute usage (hours)</th> <th>Compute cost</th> </tr></thead>
                <tbody><tr>{projectrow}</tr></tbody>
                </table>

                <table class='sortable project'>
                <thead><tr><th>Workflow run count</th> <th>Workflow name</th> <th>Median runtime</th> <th>Mean runtime</th> <th>Median cost per run</th> <th>Mean cost per run</th> <th>Sum cost over runs</th></tr></thead>
                <tbody>
                {wfsum}
                </tbody></table>
                """.format(name=prj.name,
                           wfsum=" ".join(wfsum),
                           projectrow=prj.tablerow,
                           workbench=workbench,
                           uuid=prj.uuid)
            )

        # The deduplication ratio overstates things a bit, you can
        # have collections which reference a small slice of a large
        # block, and this messes up the intuitive value of this ratio
        # and exagerates the effect.
        #
        # So for now, as much fun as this is, I'm excluding it from
        # the report.
        #
        # <p>"Monthly savings from storage deduplication" is the
        # estimated cost difference between "storage usage" and "data
        # under management" as a way of comparing with other
        # technologies that do not support data deduplication.</p>


        cards.append("""
        <h2 id="prices">Note on usage and cost calculations</h2>

        <div style="max-width: 60em">

        <p>The numbers presented in this report are estimates and will
        not perfectly match your cloud bill.  Nevertheless this report
        should be useful for identifying your main cost drivers.</p>

        <h3>Storage</h3>

        <p>"Total data under management" is what you get if you add up
        all blocks referenced by all collections in Workbench, without
        considering deduplication.</p>

        <p>"Total storage usage" is the actual underlying storage
        usage, accounting for data deduplication.</p>

        <p>Storage costs are based on AWS "S3 Standard"
        described on the <a href="https://aws.amazon.com/s3/pricing/">Amazon S3 pricing</a> page:</p>

        <ul>
        <li>$0.023 per GB / Month for the first 50 TB</li>
        <li>$0.022 per GB / Month for the next 450 TB</li>
        <li>$0.021 per GB / Month over 500 TB</li>
        </ul>

        <p>Finally, this only the base storage cost, and does not
        include any fees associated with S3 API usage.  However, there
        are generally no ingress/egress fees if your Arvados instance
        and S3 bucket are in the same region, which is the normal
        recommended configuration.</p>

        <h3>Compute</h3>

        <p>"Compute usage" are instance-hours used in running
        workflows.  Because multiple steps may run in parallel on
        multiple instances, a workflow that completes in four hours
        but runs parallel steps on five instances, would be reported
        as using 20 instance hours.</p>

        <p>"Runtime" is the actual wall clock time that it took to
        complete a workflow.  This does not include time spent in the
        queue for the workflow itself, but does include queuing time
        of individual workflow steps.</p>

        <p>Computational costs are derived from Arvados cost
        calculations of container runs.  For on-demand instances, this
        uses the prices from the InstanceTypes section of the Arvado
        config file, set by the system administrator.  For spot
        instances, this uses current spot prices retrieved on the fly
        the AWS API.</p>

        <p>Be aware that the cost calculations are only for the time
        the container is running and only do not take into account the
        overhead of launching instances or idle time between scheduled
        tasks or prior to automatic shutdown.</p>

        </div>
        """)

        return ReportChart(label, cards, self.graphs).html()

    def iter_container_info(self, pending, include_steps, exclude):
        # "pending" is a list of arvados-cwl-runner container requests
        # returned by the API.  This method fetches detailed
        # information about the runs and yields report rows.

        # 1. Get container records corresponding to container requests.
        containers = {}

        for container in arvados.util.keyset_list_all(
            self.arv_client.containers().list,
            filters=[
                ["uuid", "in", [c["container_uuid"] for c in pending if c["container_uuid"]]],
            ],
            select=["uuid", "started_at", "finished_at", "cost"]):

            containers[container["uuid"]] = container

        # 2. Look for the template_uuid property and fetch the
        # corresponding workflow record.
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

        # 3. Look at owner_uuid and fetch owning projects and users
        projects = {}

        for pr in arvados.util.keyset_list_all(
                self.arv_client.groups().list,
                filters=[
                    ["uuid", "in", list(set(c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'j7d0g'))],
                ],
                select=["uuid", "name"]):
            projects[pr["uuid"]] = pr["name"]

        # 4. Look at owner_uuid and modified_by_user_uuid and get user records
        for pr in arvados.util.keyset_list_all(
                self.arv_client.users().list,
                filters=[
                    ["uuid", "in", list(set(c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'tpzed')|set(c["modified_by_user_uuid"] for c in pending))],
                ],
                select=["uuid", "full_name", "first_name", "last_name"]):
            projects[pr["uuid"]] = pr["full_name"]

        # 5. Optionally iterate over individual workflow steps.
        if include_steps:
            name_regex = re.compile(r"(.+)_[0-9]+")
            child_crs = {}
            child_cr_containers = set()
            stepcount = 0

            # 5.1. Go through the container requests owned by the toplevel workflow container
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

                # 5.2. Get the containers corresponding to the
                # container requests.  This has the same logic as
                # report_from_api where we batch it into 1000 items at
                # a time.
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

            # Get any remaining containers
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

        # 6. Now go through the list of workflow runs, yield a row
        # with all the information we have collected, as well as the
        # details for each workflow step (if enabled)
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
                                                                    uuid=row["WorkflowUUID"],
                                                                    cost=[], hours=[]))
            wfuuid = row["Workflow"]
            prj.runs[wfuuid].count += 1
            prj.runs[wfuuid].cost.append(row["CumulativeCost"])
            prj.runs[wfuuid].hours.append(hrs)
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
                    ["created_at", "<=", to.strftime("%Y%m%dT%H%M%SZ")],
                ],
                select=["uuid", "owner_uuid", "container_uuid", "name", "cumulative_cost", "properties", "modified_by_user_uuid", "created_at"]):

            if container_request["cumulative_cost"] == 0:
                continue

            # Every 1000 container requests, we fetch the
            # corresponding container records.
            #
            # What's so special about 1000?  Because that's the
            # maximum Arvados page size, so when we use ['uuid', 'in',
            # [...]] to fetch associated records it doesn't make sense
            # to provide more than 1000 uuids.
            #
            # TODO: use the ?include=container_uuid feature so a
            # separate request to the containers table isn't necessary.
            if len(pending) < 1000:
                pending.append(container_request)
            else:
                count += len(pending)
                logging.info("Exporting workflow runs %s - %s", count-len(pending), count)
                for row in self.iter_container_info(pending, include_steps, exclude):
                    self.collect_summary_stats(row)
                    yield row
                pending.clear()

        count += len(pending)
        logging.info("Exporting workflow runs %s - %s", count-len(pending), count)
        for row in self.iter_container_info(pending, include_steps, exclude):
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

    def today(self):
        return date.today()
