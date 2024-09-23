#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from datetime import timedelta, timezone

def get_metric_usage(prom, start_time, end_time, metric, resampleTo="min"):
    from prometheus_api_client.utils import parse_datetime
    from prometheus_api_client import PrometheusConnect, MetricsList, Metric
    import pandas

    start = start_time
    chunk_size = timedelta(days=1)

    while start < end_time:
        if start + chunk_size > end_time:
            chunk_size = end_time - start

        metric_data = prom.custom_query_range(metric,
                                              start_time=start,
                                              end_time=(start + chunk_size),
                                              step=15
                                              )

        start += chunk_size

        if len(metric_data) == 0:
            continue

        if "__name__" not in metric_data[0]["metric"]:
            metric_data[0]["metric"]["__name__"] = metric

        metric_object_list = MetricsList(metric_data)
        my_metric_object = metric_object_list[0] # one of the metrics from the list

        series = my_metric_object.metric_values.set_index(pandas.DatetimeIndex(my_metric_object.metric_values['ds']))

        # Resample to 1 minute increments, fill in missing values
        rs = series.resample(resampleTo).max(1).ffill()

        yield rs

def get_data_usage(prom, timestamp, cluster):
    from prometheus_api_client import PrometheusConnect, MetricsList, Metric

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_total_bytes',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})

    metric_object_list = MetricsList(metric_data)

    if len(metric_data) == 0:
        return

    my_metric_object = metric_object_list[0] # one of the metrics from the list
    value = my_metric_object.metric_values.iloc[0]["y"]

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_dedup_byte_ratio',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})

    if len(metric_data) == 0:
        return (None, None)

    my_metric_object = MetricsList(metric_data)[0]
    dedup_ratio = my_metric_object.metric_values.iloc[0]["y"]

    return value, dedup_ratio
