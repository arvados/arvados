# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import dataclasses
import datetime

from typing import Sequence

@dataclasses.dataclass
class FakePrometheusClient:
    values: Sequence[str]
    start_time: datetime.datetime = datetime.datetime(2026, 3, 5, 11, tzinfo=datetime.timezone.utc)
    delta: datetime.timedelta = datetime.timedelta(minutes=5)

    def _iter_values(self):
        time = self.start_time
        for value in self.values:
            yield time, value
            time += self.delta

    def custom_query_range(self, metric, start_time, end_time, step):
        metric_name, _, _ = metric.partition('{')
        metric_info = {'__name__': metric_name}
        values = [
            [dt.timestamp(), value]
            for dt, value in self._iter_values()
            if start_time <= dt < end_time
        ]
        return [{'metric': metric_info, 'values': values}]
