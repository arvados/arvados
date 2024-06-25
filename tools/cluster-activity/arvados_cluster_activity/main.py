#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import sys

import arvados
import arvados.util
import ciso8601
import csv
import os
import logging
import re

from arvados_cluster_activity.report import ClusterActivityReport, aws_monthly_cost
from arvados_cluster_activity.prometheus import get_metric_usage

from datetime import timedelta, timezone, datetime
import base64

prometheus_support = True

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--start', help='Start date for the report in YYYY-MM-DD format (UTC) (or use --days)')
    arg_parser.add_argument('--end', help='End date for the report in YYYY-MM-DD format (UTC), default "now"')
    arg_parser.add_argument('--days', type=int, help='Number of days before "end" to start the report (or use --start)')
    arg_parser.add_argument('--cost-report-file', type=str, help='Export cost report to specified CSV file')
    arg_parser.add_argument('--include-workflow-steps', default=False,
                            action="store_true", help='Include individual workflow steps (optional)')
    arg_parser.add_argument('--columns', type=str, help="""Cost report columns (optional), must be comma separated with no spaces between column names.
    Available columns are: Project, ProjectUUID, Workflow, WorkflowUUID, Step, StepUUID, Sample, SampleUUID, User, UserUUID, Submitted, Started, Runtime, Cost""")
    arg_parser.add_argument('--exclude', type=str, help="Exclude workflows containing this substring (may be a regular expression)")

    arg_parser.add_argument('--html-report-file', type=str, help='Export HTML report to specified file')

    if prometheus_support:
        arg_parser.add_argument('--cluster', type=str, help='Cluster to query for prometheus stats')
        arg_parser.add_argument('--prometheus-auth', type=str, help='Authorization file with prometheus info')

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
            to = datetime.strptime(args.end,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: end date must be in YYYY-MM-DD format")
            exit(1)
    else:
        to = datetime.now(timezone.utc)

    if args.days:
        since = to - timedelta(days=args.days)

    if args.start:
        try:
            since = datetime.strptime(args.start,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: start date must be in YYYY-MM-DD format")
            exit(1)


    if prometheus_support and args.prometheus_auth:
        with open(args.prometheus_auth, "rt") as f:
            for line in f:
                sp = line.strip().split("=")
                if sp[0].startswith("PROMETHEUS_"):
                    os.environ[sp[0]] = sp[1]

    return args, since, to

def data_usage(prom, timestamp, cluster, label):
    from prometheus_api_client import PrometheusConnect, MetricsList, Metric

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

    monthly_cost = aws_monthly_cost(value)

    for scale in ["KiB", "MiB", "GiB", "TiB", "PiB"]:
        summary_value = summary_value / 1024
        if summary_value < 1024:
            print(label,
                  "%.3f %s apparent," % (summary_value*dedup_ratio, scale),
                  "%.3f %s actually stored," % (summary_value, scale),
                  "$%.2f monthly S3 storage cost" % monthly_cost)
            break


def print_container_usage(prom, start_time, end_time, metric, label, fn=None):
    cumulative = 0

    for rs in get_metric_usage(prom, start_time, end_time, metric):
        # Calculate the sum of values
        #print(rs.sum()["y"])
        cumulative += rs.sum()["y"]

    if fn is not None:
        cumulative = fn(cumulative)

    print(label % cumulative)


def get_prometheus_client():
    from prometheus_api_client import PrometheusConnect

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

    return prom

def report_from_prometheus(prom, cluster, since, to):

    print(cluster, "between", since, "and", to, "timespan", (to-since))

    try:
        data_usage(prom, since, cluster, "at start:")
    except:
        logging.exception("Failed to get start value")

    try:
        data_usage(prom, to - timedelta(minutes=240), cluster, "current :")
    except:
        logging.exception("Failed to get end value")

    print_container_usage(prom, since, to, "arvados_dispatchcloud_containers_running{cluster='%s'}" % cluster, '%.1f container hours', lambda x: x/60)
    print_container_usage(prom, since, to, "sum(arvados_dispatchcloud_instances_price{cluster='%s'})" % cluster, '$%.2f spent on compute', lambda x: x/60)
    print()


def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    args, since, to = parse_arguments(arguments)

    logging.getLogger().setLevel(logging.INFO)

    prom = None
    if prometheus_support:
        if "PROMETHEUS_HOST" in os.environ:
            prom = get_prometheus_client()
            if args.cluster:
                report_from_prometheus(prom, args.cluster, since, to)
            else:
                logging.warn("--cluster not provided, not collecting activity from Prometheus")
        else:
            logging.warn("PROMETHEUS_HOST not found, not collecting activity from Prometheus")

    reporter = ClusterActivityReport(prom)

    if args.cost_report_file:
        with open(args.cost_report_file, "wt") as f:
            reporter.csv_report(since, to, f, args.include_workflow_steps, args.columns, args.exclude)
    else:
        logging.warn("--cost-report-file not provided, not writing cost report")

    if args.html_report_file:
        with open(args.html_report_file, "wt") as f:
            f.write(reporter.html_report(since, to, args.exclude))

if __name__ == "__main__":
    main()
