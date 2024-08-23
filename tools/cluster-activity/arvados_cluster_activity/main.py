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

from arvados_cluster_activity.report import ClusterActivityReport, aws_monthly_cost, format_with_suffix_base2
from arvados_cluster_activity.prometheus import get_metric_usage, get_data_usage

from arvados_cluster_activity._version import __version__

from datetime import timedelta, timezone, datetime
import base64

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
    arg_parser.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')

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


    if args.prometheus_auth:
        with open(args.prometheus_auth, "rt") as f:
            for line in f:
                if line.startswith("export "):
                   line = line[7:]
                sp = line.strip().split("=")
                if sp[0].startswith("PROMETHEUS_"):
                    os.environ[sp[0]] = sp[1]

    return args, since, to

def print_data_usage(prom, timestamp, cluster, label):
    value, dedup_ratio = get_data_usage(prom, timestamp, cluster)

    if value is None:
        return

    monthly_cost = aws_monthly_cost(value)
    print(label,
          "%s apparent," % (format_with_suffix_base2(value*dedup_ratio)),
          "%s actually stored," % (format_with_suffix_base2(value)),
          "$%.2f monthly S3 storage cost" % monthly_cost)

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
    try:
        from prometheus_api_client import PrometheusConnect
    except ImportError as e:
        logging.warn("Failed to import prometheus_api_client client.  Did you include the [prometheus] option when installing the package?  Error was: %s" % e)
        return None

    prom_host = os.environ.get("PROMETHEUS_HOST")
    prom_token = os.environ.get("PROMETHEUS_APIKEY")
    prom_user = os.environ.get("PROMETHEUS_USER")
    prom_pw = os.environ.get("PROMETHEUS_PASSWORD")

    headers = {}
    if prom_token:
        headers["Authorization"] = "Bearer %s" % prom_token

    if prom_user:
        headers["Authorization"] = "Basic %s" % str(base64.b64encode(bytes("%s:%s" % (prom_user, prom_pw), 'utf-8')), 'utf-8')

    try:
        return PrometheusConnect(url=prom_host, headers=headers)
    except Exception as e:
        logging.warn("Connecting to Prometheus failed, will not collect activity from Prometheus.  Error was: %s" % e)
        return None

def report_from_prometheus(prom, cluster, since, to):

    if not cluster:
        arv_client = arvados.api()
        cluster = arv_client.config()["ClusterID"]

    print(cluster, "between", since, "and", to, "timespan", (to-since))

    try:
        print_data_usage(prom, since, cluster, "at start:")
    except:
        logging.exception("Failed to get start value")

    try:
        print_data_usage(prom, to - timedelta(minutes=240), cluster, "current :")
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
    if "PROMETHEUS_HOST" in os.environ:
        prom = get_prometheus_client()
    else:
        logging.warn("PROMETHEUS_HOST not found, not collecting activity from Prometheus")

    reporter = ClusterActivityReport(prom)

    if args.cost_report_file:
        with open(args.cost_report_file, "wt") as f:
            reporter.csv_report(since, to, f, args.include_workflow_steps, args.columns, args.exclude)
    else:
        logging.info("Use --cost-report-file to get a CSV file of workflow runs")

    if args.html_report_file:
        with open(args.html_report_file, "wt") as f:
            f.write(reporter.html_report(since, to, args.exclude, args.include_workflow_steps))
    else:
        logging.info("Use --html-report-file to get HTML report of cluster usage")

    if not args.cost_report_file and not args.html_report_file:
        report_from_prometheus(prom, args.cluster, since, to)

if __name__ == "__main__":
    main()
