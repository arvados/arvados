#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import dataclasses

from datetime import datetime, timedelta, timezone
from typing import Any, Iterable

try:
    from prometheus_api_client import PrometheusConnect, MetricsList, Metric
except ImportError:
    pandas = None
else:
    from prometheus_api_client.utils import parse_datetime
    import pandas

def get_metric_usage(prom, start_time, end_time, metric, resampleTo='1min'):
    if not prom:
        return
    delta = timedelta(days=1)
    while start_time < end_time:
        this_end = min(start_time + delta, end_time)
        metric_data = prom.custom_query_range(
            metric,
            start_time=start_time,
            end_time=this_end,
            step=15,
        )
        start_time = this_end
        try:
            metric_data[0]['metric'].setdefault('__name__', metric)
        except IndexError:
            continue
        metric_list = MetricsList(metric_data)
        mv0 = metric_list[0].metric_values
        series = mv0.set_index(pandas.DatetimeIndex(mv0['ds']))
        yield series.resample(resampleTo).max(1).ffill()


@dataclasses.dataclass(slots=True)
class PrometheusDataPoint:
    datetime: datetime
    value: Any

    def to_json(self):
        # Convert timestamp to milliseconds for JavaScript.
        return [int(self.datetime.timestamp() * 1000), self.value]


@dataclasses.dataclass
class PrometheusUsageChart:
    title: str
    line_label: str
    series_data: dataclasses.InitVar[Iterable]
    data: list[PrometheusDataPoint] = dataclasses.field(init=False)

    def __post_init__(self, series_data):
        self.data = [
            PrometheusDataPoint(dt, value)
            for series in series_data
            for dt, value, *_ in series.itertuples()
        ]

    def iter_values(self):
        return (data.value for data in self.data)

    def last_value(self, default=None):
        try:
            return self.data[-1].value
        except IndexError:
            return default

    def json_object(self):
        return {
            'label': self.title,
            'charts': [{
                'options': {
                    'legend': 'always',
                    'connectSeparatedPoints': True,
                    'labels': ['date', self.line_label],
                    'includeZero': True,
                    'title': self.title,
                },
                'data': [data.to_json() for data in self.data],
            }],
        }


def get_data_usage(prom, timestamp, cluster):
    if not prom:
        return (None, None)

    from prometheus_api_client import PrometheusConnect, MetricsList, Metric

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_total_bytes',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})
    metric_object_list = MetricsList(metric_data)
    if not metric_data:
        return (None, None)

    my_metric_object = metric_object_list[0] # one of the metrics from the list
    value = my_metric_object.metric_values.iloc[0]["y"]

    metric_data = prom.get_current_metric_value(metric_name='arvados_keep_dedup_byte_ratio',
                                                label_config={"cluster": cluster},
                                                params={"time": timestamp.timestamp()})
    if not metric_data:
        return (None, None)

    my_metric_object = MetricsList(metric_data)[0]
    dedup_ratio = my_metric_object.metric_values.iloc[0]["y"]
    return value, dedup_ratio
