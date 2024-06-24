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
from datetime import datetime, timedelta
import pkg_resources

from dataclasses import dataclass

import crunchstat_summary.dygraphs
from crunchstat_summary.summarizer import Task

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
    name: str = ""
    cost: float = 0
    count: int = 0
    hours: float = 0


class Summarizer:
    label: str
    tasks: collections.defaultdict[str, Task]

    def long_label(self):
        return self.label


def date_export(item):
    if isinstance(item, datetime):
        return """@new Date("{}")@""".format(item.strftime("%Y-%m-%dT%H:%M:%SZ"))

class ReportChart(crunchstat_summary.dygraphs.DygraphsChart):
    def sections(self):
        return [
            {
                'label': s.long_label(),
                'charts': [
                    self.chartdata(s.label, s.tasks, stat)
                    for stat in (('Compute', ['hours']),
                                 ('Disk', ['GiB']),
                                 )
                    ],
            }
            for s in self.summarizers]

    def js(self):
        return 'var chartdata = {};\n{}'.format(
            json.dumps(self.sections(), default=date_export).replace('"@', '').replace('@"', '').replace('\\"', '"'),
            '\n'.join([pkg_resources.resource_string('crunchstat_summary', jsa).decode('utf-8') for jsa in self.JSASSETS]))

    def _collate_data(self, tasks, stats):
        data = []
        nulls = []
        # uuid is category for crunch2
        print(tasks, stats)
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


def csv_dateformat(date):
    dt = ciso8601.parse_datetime(date)
    return dt.strftime("%Y-%m-%d %H:%M%S")


class ClusterActivityReport(object):
    def __init__(self, label=None, threads=1, **kwargs):
        self.label = "Cluster report"
        self.threadcount = threads
        self.arv_client = arvados.api()

        self.active_users = set()
        self.project_summary = {}
        self.total_hours = 0
        self.total_cost = 0

    def run(self):
        pass

    def html_report(self, since, to, exclude):

        for row in self.report_from_api(since, to, True, exclude):
            pass

        tophtml = ""
        bottomhtml = ""
        label = self.label

        tophtml = """<h2>Summary</h2>
        <p>Total user accounts: {total_users}</p>
        <p>Active users: {active_users}</p>
        <p>Aggregate compute hours: {total_hours}</p>
        <p>Aggregate compute cost: ${total_cost}</p>
        """.format(active_users=len(self.active_users),
                   total_users=self.total_users,
                   total_hours=round(self.total_hours, 1),
                   total_cost=round(self.total_cost, 2))

        bottomhtml = []

        for k, prj in sorted(self.project_summary.items(), key=lambda x: x[1].cost, reverse=True):
            wfsum = []
            for k2, r in sorted(prj.runs.items(), key=lambda x: x[1].cost, reverse=True):
                wfsum.append( """
                {name} -- count {count} -- average runtime {runtime} -- average cost per run ${cost}
                """.format(name=r.name, count=r.count, runtime=hours_to_runtime_str(r.hours/r.count), cost=round(r.cost/r.count, 2)))

            bottomhtml.append(
                """<h2>{name}</h2>
                <p>Users: {users}</p>
                <p>{wfsum}</p>
                <p>Compute hours: {hours}</p>
                <p>Compute cost: ${cost}</p>
                """.format(name=prj.name,
                           users=", ".join(prj.users),
                           cost=round(prj.cost, 2),
                           hours=round(prj.hours, 1),
                           wfsum="</p><p>".join(wfsum))
            )


        summarizers = []
        s1 = Summarizer()
        s1.label = "Compute"
        s1.tasks = collections.defaultdict(Task)

        task = s1.tasks["Compute"]
        task.series["Compute", "hours"].append((datetime.now() + timedelta(minutes=0), 1))
        task.series["Compute", "hours"].append((datetime.today() + timedelta(minutes=2), 2))
        task.series["Compute", "hours"].append((datetime.today() + timedelta(minutes=4), 1))

        summarizers.append(s1)

        return WEBCHART_CLASS(label, summarizers).html(tophtml, bottomhtml)

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
            workflowname = container_request["name"] if template_uuid == "none" else workflows.get(template_uuid, "workflow missing")

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

        if row["Step"] == "workflow runner":
            prj.runs.setdefault(row["Workflow"], WorkflowRunSummary(name=row["Workflow"],
                                                                    uuid=row["WorkflowUUID"]))
            wfuuid = row["Workflow"]
            prj.runs[wfuuid].count += 1
            prj.runs[wfuuid].cost += row["CumulativeCost"]
            prj.runs[wfuuid].hours += hrs

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
