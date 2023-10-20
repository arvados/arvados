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


def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    args, since, to = parse_arguments(arguments)

    #arv = arvados.api()

    prom_host = os.environ["PROMETHEUS_HOST"]
    prom_token = os.environ.get("PROMETHEUS_APIKEY")
    prom_user = os.environ.get("PROMETHEUS_USER")
    prom_pw = os.environ.get("PROMETHEUS_PASSWORD")

    headers = {}
    if prom_token:
        headers["Authorization"] = "Bearer %s" % prom_token

    if prom_user:
        headers["Authorization"] = "Basic %s" % str(base64.b64encode(bytes("%s:%s" % (prom_user, prom_pw), 'utf-8')), 'utf-8')

    print(headers)
    prom = PrometheusConnect(url=prom_host, headers=headers)

    cluster = args.cluster

    print(cluster, "between", since, "and", to, "timespan", (to-since))

    data_usage(prom, since, cluster, "at start:")
    data_usage(prom, to - timedelta(minutes=240), cluster, "current :")

    container_usage(prom, since, to, "arvados_dispatchcloud_containers_running{cluster='%s'}" % cluster, '%.1f container hours', lambda x: x/60)
    container_usage(prom, since, to, "sum(arvados_dispatchcloud_instances_price{cluster='%s'})" % cluster, '$%.2f spent on compute', lambda x: x/60)
    print()

if __name__ == "__main__":
    main()
