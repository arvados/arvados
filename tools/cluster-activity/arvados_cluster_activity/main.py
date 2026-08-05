#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import base64
from datetime import timedelta, timezone, datetime
import logging
import os

prometheus_import_error = None
try:
    from prometheus_api_client import PrometheusConnect
except ImportError as e:
    prometheus_import_error = e
    PrometheusConnect = None

import arvados

from arvados_cluster_activity.report import ClusterActivityReport, aws_monthly_cost, bytes_base2_fmt, WorkflowRunCSVRow
from arvados_cluster_activity.prometheus import get_metric_usage, get_data_usage
from arvados_cluster_activity._version import __version__


class _ArgTypes:
    """Namespace class that holds types and utilities for argument parsing
    and validation.
    """
    @staticmethod
    def date_arg(text: str | None = None) -> datetime:
        """Converts the input string in the form of YYYY-MM-DD to a
        UTC-timezone datetime object for the specified date at 00h00m00s
        ("midnight"). When called without argument, returns the midnight
        datetime of the current UTC day.
        """
        if text is None:
            dt_obj = datetime.now(timezone.utc)
        else:
            try:
                dt_obj = datetime.strptime(text, "%Y-%m-%d")
            except ValueError:
                # strptime may raise ValueError with a cryptic message, e.g.
                # strptime("2026-01-32", "%Y-%m-%d") -> "unconverted data
                # remains: 2" (tested in Pythons 3.10 -- 3.13), so we provide a
                # clear and fiendly message.
                raise argparse.ArgumentTypeError(
                    f"invalid date: {text!r}"
                ) from None
        return dt_obj.replace(
            hour=0, minute=0, second=0, microsecond=0, tzinfo=timezone.utc
        )

    @staticmethod
    def positive_days(text: str) -> timedelta:
        """Returns a timedelta of N days with the input string parsed as an
        integer. Negative values are invalid.
        """
        n = int(text)  # Let ValueError propagate.
        if n < 0:
            raise ValueError(f"not a positive integer: {text!r}")
        return timedelta(days=n)

    @staticmethod
    def load_prometheus_auth(path: str) -> str:
        """Loads Prometheus environment variables from `path` into the current
        process's `os.environ`.

        Returns `path` unmodified if the operation succeeds.
        """
        prom_vars = {}
        try:
            with open(path, "rt") as f:
                for line in f:
                    if line.startswith("export "):
                        line = line[7:]
                    sp = line.strip().split("=")
                    if sp[0].startswith("PROMETHEUS_"):
                        prom_vars[sp[0]] = sp[1]
        except OSError as err:
            raise argparse.ArgumentTypeError(str(err)) from None
        os.environ.update(prom_vars)
        return path

    VALID_FIELDS = WorkflowRunCSVRow.field_names()

    @staticmethod
    def columns(text: str) -> tuple[str]:
        # We can be more lenient with space characters around commas, but still
        # expect the arg value as one string, for compatibility.
        valid_fields = frozenset(_ArgTypes.VALID_FIELDS)
        result = []
        for elem in map(str.strip, text.split(",")):
            if elem not in valid_fields:
                raise ValueError(f"invalid column name: {elem!r}")
            result.append(elem)
        return tuple(result)


def get_argument_parser(prog: str | None = None) -> argparse.ArgumentParser:
    arg_parser = argparse.ArgumentParser(prog=prog)

    start_date_group = arg_parser.add_mutually_exclusive_group(required=True)
    start_date_group.add_argument(
        '--start',
        type=_ArgTypes.date_arg,
        help='Start date for the report in YYYY-MM-DD format (UTC)',
        metavar="DATE"
    )
    start_date_group.add_argument(
        '--days',
        type=_ArgTypes.positive_days,
        help='Number of days before "end" to start the report',
        metavar="N"
    )
    arg_parser.add_argument(
        '--end',
        type=_ArgTypes.date_arg,
        help=(
            'End date for the report in YYYY-MM-DD format (UTC);'
            ' Default: today'
        ),
        metavar="DATE",
        default=_ArgTypes.date_arg()  # Today.
    )

    arg_parser.add_argument(
        '--cost-report-file',
        help='Export cost report to specified CSV file',
        metavar='PATH'
    )
    arg_parser.add_argument(
        '--include-workflow-steps', default=False, action="store_true",
        help='Include individual workflow steps in cost report (optional)'
    )
    arg_parser.add_argument(
        '--columns',
        type=_ArgTypes.columns,
        help=(
            "Cost report columns (optional), must be comma separated with no"
            " spaces between column names.  Available columns are:"
            f" {', '.join(_ArgTypes.VALID_FIELDS)}."
        ),
        metavar="COLUMN[,COLUMN ...]"
    )
    arg_parser.add_argument(
        '--exclude',
        help=(
            "Exclude workflows containing this substring"
            " (may be a regular expression)"
        ),
        metavar="PATTERN"
    )

    arg_parser.add_argument(
        '--html-report-file',
        help='Export HTML report to specified file',
        metavar='PATH'
    )
    arg_parser.add_argument(
        '--version', action='version', version=f"%(prog)s {__version__}",
        help='Print version and exit.'
    )

    arg_parser.add_argument(
        '--cluster',
        help='Cluster to query for prometheus stats',
        metavar='CLUSTER-ID'
    )
    arg_parser.add_argument(
        '--prometheus-auth',
        type=_ArgTypes.load_prometheus_auth,
        help='Authorization file with prometheus info',
        metavar='PATH'
    )

    return arg_parser


def print_data_usage(prom, timestamp, cluster, label):
    value, dedup_ratio = get_data_usage(prom, timestamp, cluster)

    if value is None:
        return

    monthly_cost = aws_monthly_cost(value)
    print(label,
          "%s apparent," % (bytes_base2_fmt(value*dedup_ratio)),
          "%s actually stored," % (bytes_base2_fmt(value)),
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
    if PrometheusConnect is None:
        logging.warn("Failed to import prometheus_api_client client.  Did you include the [prometheus] option when installing the package?  Error was: %s" % prometheus_import_error)
        return None

    headers = {}
    if not (prom_host := os.environ.get("PROMETHEUS_HOST")):
        logging.warn("PROMETHEUS_HOST not found, not collecting activity from Prometheus")
        return None
    elif prom_token := os.environ.get("PROMETHEUS_APIKEY"):
        headers["Authorization"] = f"Bearer {prom_token}"
    elif prom_user := os.environ.get("PROMETHEUS_USER"):
        basic_auth = base64.b64encode(
            f"{prom_user}:{os.environ.get('PROMETHEUS_PASSWORD', '')}".encode('utf-8'),
        ).decode('ascii')
        headers["Authorization"] = f"Basic {basic_auth}"
    else:
        logging.warn("Prometheus credentials not found, not collecting activity from Prometheus")
        return None

    try:
        return PrometheusConnect(url=prom_host, headers=headers)
    except Exception as e:
        logging.warn("Connecting to Prometheus failed, will not collect activity from Prometheus.  Error was: %s" % e)
        return None

def report_from_prometheus(prom, cluster, start_time, end_time):

    if not cluster:
        arv_client = arvados.api()
        cluster = arv_client.config()["ClusterID"]

    print(cluster, "between", start_time, "and", end_time, "timespan", (end_time - start_time))

    try:
        print_data_usage(prom, start_time, cluster, "at start:")
    except:
        logging.exception("Failed to get start value")

    try:
        print_data_usage(prom, end_time - timedelta(minutes=240), cluster, "current :")
    except:
        logging.exception("Failed to get end value")

    print_container_usage(prom, start_time, end_time, "arvados_dispatchcloud_containers_running{cluster='%s'}" % cluster, '%.1f container hours', lambda x: x/60)
    print_container_usage(prom, start_time, end_time, "sum(arvados_dispatchcloud_instances_price{cluster='%s'})" % cluster, '$%.2f spent on compute', lambda x: x/60)
    print()


def main(arguments=None):
    parser = get_argument_parser()
    args = parser.parse_args(arguments)
    if args.start is None:
        args.start = args.end - args.days

    logging.basicConfig(
        level=logging.INFO,
    )

    prom_client = get_prometheus_client()
    reporter = ClusterActivityReport(
        args.start,
        args.end,
        prom_client=prom_client,
        exclude=args.exclude,
    )
    if args.cost_report_file:
        with open(args.cost_report_file, "wt") as f:
            reporter.csv_report(f, args.columns, include_steps=args.include_workflow_steps)
    else:
        logging.info("Use --cost-report-file to get a CSV file of workflow runs")

    if args.html_report_file:
        with open(args.html_report_file, "wt") as f:
            f.write(reporter.html_report())
    else:
        logging.info("Use --html-report-file to get HTML report of cluster usage")

    if not args.cost_report_file and not args.html_report_file:
        report_from_prometheus(prom_client, args.cluster, args.start, args.end)

if __name__ == "__main__":
    main()
