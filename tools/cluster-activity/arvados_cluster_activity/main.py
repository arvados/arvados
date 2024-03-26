#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import sys

import arvados
import arvados.util
import datetime
import ciso8601
import csv
import os
from prometheus_api_client.utils import parse_datetime
from datetime import timedelta
import pandas
import base64

from prometheus_api_client import PrometheusConnect, MetricsList, Metric

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--start', help='Start date for the report in YYYY-MM-DD format (UTC)')
    arg_parser.add_argument('--end', help='End date for the report in YYYY-MM-DD format (UTC), default "now"')
    arg_parser.add_argument('--days', type=int, help='Number of days before "end" to start the report')
    arg_parser.add_argument('--cluster', type=str, help='Cluster to query')
    arg_parser.add_argument('--cost-report-file', type=str, help='Export cost report to specified CSV file')
    args = arg_parser.parse_args(arguments)

    if args.days and args.start:
        arg_parser.print_help()
        print("Error: either specify --days or both --start and --end")
        exit(1)

    if not args.days and not args.start:
        arg_parser.print_help()
        print("\nError: either specify --days or both --start and --end")
        exit(1)

    if (args.start and not args.end):
        arg_parser.print_help()
        print("\nError: no start or end date found, either specify --days or both --start and --end")
        exit(1)

    if args.end:
        try:
            to = datetime.datetime.strptime(args.end,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: end date must be in YYYY-MM-DD format")
            exit(1)
    else:
        to = datetime.datetime.utcnow()

    if args.days:
        since = to - datetime.timedelta(days=args.days)

    if args.start:
        try:
            since = datetime.datetime.strptime(args.start,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: start date must be in YYYY-MM-DD format")
            exit(1)


    return args, since, to

def data_usage(prom, timestamp, cluster, label):
    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_total_bytes',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})

    metric_object_list = MetricsList(metric_data)

    if len(metric_data) == 0:
        return

    my_metric_object = metric_object_list[0] # one of the metrics from the list
    value = my_metric_object.metric_values.iloc[0]["y"]
    summary_value = value

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_dedup_byte_ratio',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})

    if len(metric_data) == 0:
        return

    my_metric_object = MetricsList(metric_data)[0]
    dedup_ratio = my_metric_object.metric_values.iloc[0]["y"]

    value_gb = value / (1024*1024*1024)
    first_50tb = min(1024*50, value_gb)
    next_450tb = max(min(1024*450, value_gb-1024*50), 0)
    over_500tb = max(value_gb-1024*500, 0)

    monthly_cost = (first_50tb * 0.023) + (next_450tb * 0.022) + (over_500tb * 0.021)

    for scale in ["KiB", "MiB", "GiB", "TiB", "PiB"]:
        summary_value = summary_value / 1024
        if summary_value < 1024:
            print(label,
                  "%.3f %s apparent," % (summary_value*dedup_ratio, scale),
                  "%.3f %s actually stored," % (summary_value, scale),
                  "$%.2f monthly S3 storage cost" % monthly_cost)
            break




def container_usage(prom, start_time, end_time, metric, label, fn=None):
    start = start_time
    chunk_size = timedelta(days=1)
    cumulative = 0

    while start < end_time:
        if start + chunk_size > end_time:
            chunk_size = end_time - start

        metric_data = prom.custom_query_range(metric,
                                              start_time=start,
                                              end_time=(start + chunk_size),
                                              step=15
                                              )

        if len(metric_data) == 0:
            break

        if "__name__" not in metric_data[0]["metric"]:
            metric_data[0]["metric"]["__name__"] = metric

        metric_object_list = MetricsList(metric_data)
        my_metric_object = metric_object_list[0] # one of the metrics from the list

        series = my_metric_object.metric_values.set_index(pandas.DatetimeIndex(my_metric_object.metric_values['ds']))

        # Resample to 1 minute increments, fill in missing values
        rs = series.resample("min").mean(1).ffill()

        # Calculate the sum of values
        #print(rs.sum()["y"])
        cumulative += rs.sum()["y"]

        start += chunk_size

    if fn is not None:
        cumulative = fn(cumulative)

    print(label % cumulative)

def report_from_prometheus(cluster, since, to):
    prom_host = os.environ.get("PROMETHEUS_HOST")
    prom_token = os.environ.get("PROMETHEUS_APIKEY")
    prom_user = os.environ.get("PROMETHEUS_USER")
    prom_pw = os.environ.get("PROMETHEUS_PASSWORD")

    headers = {}
    if prom_token:
        headers["Authorization"] = "Bearer %s" % prom_token

    if prom_user:
        headers["Authorization"] = "Basic %s" % str(base64.b64encode(bytes("%s:%s" % (prom_user, prom_pw), 'utf-8')), 'utf-8')

    prom = PrometheusConnect(url=prom_host, headers=headers)

    print(cluster, "between", since, "and", to, "timespan", (to-since))

    try:
        data_usage(prom, since, cluster, "at start:")
    except:
        pass
    try:
        data_usage(prom, to - timedelta(minutes=240), cluster, "current :")
    except:
        pass

    container_usage(prom, since, to, "arvados_dispatchcloud_containers_running{cluster='%s'}" % cluster, '%.1f container hours', lambda x: x/60)
    container_usage(prom, since, to, "sum(arvados_dispatchcloud_instances_price{cluster='%s'})" % cluster, '$%.2f spent on compute', lambda x: x/60)
    print()

def flush_containers(arv_client, csvwriter, pending):
    containers = {}

    for container in arvados.util.keyset_list_all(
        arv_client.containers().list,
        filters=[
            ["uuid", "in", [c["container_uuid"] for c in pending]],
        ],
        select=["uuid", "started_at", "finished_at"]):

        containers[container["uuid"]] = container

    workflows = {}
    workflows["none"] = "workflow run from command line"

    for wf in arvados.util.keyset_list_all(
            arv_client.workflows().list,
            filters=[
                ["uuid", "in", [c["properties"]["template_uuid"] for c in pending if "template_uuid" in c["properties"]]],
            ],
            select=["uuid", "name"]):
        workflows[wf["uuid"]] = wf["name"]

    projects = {}

    for pr in arvados.util.keyset_list_all(
            arv_client.groups().list,
            filters=[
                ["uuid", "in", [c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'j7d0g']],
            ],
            select=["uuid", "name"]):
        projects[pr["uuid"]] = pr["name"]

    for pr in arvados.util.keyset_list_all(
            arv_client.users().list,
            filters=[
                ["uuid", "in", [c["owner_uuid"] for c in pending if c["owner_uuid"][6:11] == 'tpzed']],
            ],
            select=["uuid", "full_name", "first_name", "last_name"]):
        projects[pr["uuid"]] = pr["full_name"]

    for container_request in pending:
        length = ciso8601.parse_datetime(containers[container_request["container_uuid"]]["finished_at"]) - ciso8601.parse_datetime(containers[container_request["container_uuid"]]["started_at"])

        hours = length.seconds // 3600
        minutes = (length.seconds // 60) % 60
        seconds = length.seconds % 60

        csvwriter.writerow((
            projects.get(container_request["owner_uuid"], "unknown owner"),
            workflows.get(container_request["properties"].get("template_uuid", "none"), "workflow missing"),
            container_request["name"],
            containers[container_request["container_uuid"]]["started_at"],
            "%i:%02i:%02i:%02i" % (length.days, hours, minutes, seconds),
            container_request["cumulative_cost"],
            ))


def report_from_api(since, to, out):
    arv_client = arvados.api()

    csvwriter = csv.writer(out)
    csvwriter.writerow(("Project", "Workflow", "Sample", "Started", "Runtime", "Cost"))

    pending = []

    print(since.isoformat())
    for container_request in arvados.util.keyset_list_all(
            arv_client.container_requests().list,
            filters=[
                ["command", "like", "[\"arvados-cwl-runner%"],
                ["created_at", ">=", since.strftime("%Y%m%dT%H%M%SZ")],
            ],
            select=["uuid", "owner_uuid", "container_uuid", "name", "cumulative_cost", "properties"]):

        if len(pending) < 1000:
            pending.append(container_request)
        else:
            flush_containers(arv_client, csvwriter, pending)
            pending.clear()

    flush_containers(arv_client, csvwriter, pending)

def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    args, since, to = parse_arguments(arguments)

    if "PROMETHEUS_HOST" in os.environ:
        report_from_prometheus(args.cluster, since, to)

    if args.cost_report_file and "ARVADOS_API_HOST" in os.environ:
        with open(args.cost_report_file, "wt") as f:
            report_from_api(since, to, f)

if __name__ == "__main__":
    main()
