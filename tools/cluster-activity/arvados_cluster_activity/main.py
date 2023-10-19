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

from prometheus_api_client import PrometheusConnect, MetricsList, Metric

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--start', help='Start date for the report in YYYY-MM-DD format (UTC)')
    arg_parser.add_argument('--end', help='End date for the report in YYYY-MM-DD format (UTC)')
    arg_parser.add_argument('--days', type=int, help='Number of days before now() to start the report')
    args = arg_parser.parse_args(arguments)

    if args.days and (args.start or args.end):
        arg_parser.print_help()
        print("Error: either specify --days or both --start and --end")
        exit(1)

    if not args.days and (not args.start or not args.end):
        arg_parser.print_help()
        print("\nError: either specify --days or both --start and --end")
        exit(1)

    if (args.start and not args.end) or (args.end and not args.start):
        arg_parser.print_help()
        print("\nError: no start or end date found, either specify --days or both --start and --end")
        exit(1)

    if args.days:
        to = datetime.datetime.utcnow()
        since = to - datetime.timedelta(days=args.days)

    if args.start:
        try:
            since = datetime.datetime.strptime(args.start,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: start date must be in YYYY-MM-DD format")
            exit(1)

    if args.end:
        try:
            to = datetime.datetime.strptime(args.end,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: end date must be in YYYY-MM-DD format")
            exit(1)

    return args, since, to


def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    #args, since, to = parse_arguments(arguments)

    #arv = arvados.api()

    prom_host = os.environ["PROMETHEUS_HOST"]
    prom_token = os.environ["PROMETHEUS_APIKEY"]

    prom = PrometheusConnect(url=prom_host, headers={"Authorization": "Bearer "+prom_token})

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_total_bytes', label_config={"cluster": "pirca"})

    metric_object_list = MetricsList(metric_data)
    #for item in metric_object_list:
    #    print(item.metric_name, item.label_config, "\n")

    my_metric_object = metric_object_list[0] # one of the metrics from the list

    print(my_metric_object.metric_values)
    value = my_metric_object.metric_values.iloc[0]["y"]

    for scale in ["KiB", "MiB", "GiB", "TiB", "PiB"]:
        value = value / 1024
        if value < 1024:
            print(value, scale)
            break

if __name__ == "__main__":
    main()
