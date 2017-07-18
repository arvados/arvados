# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import math
import crunchstat_summary.webchart


class ChartJS(crunchstat_summary.webchart.WebChart):
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/canvasjs/1.7.0/canvasjs.min.js'
    JSASSET = 'chartjs.js'

    def chartdata(self, label, tasks, stat):
        return {
            'axisY': self._axisY(tasks=tasks, stat=stat),
            'data': [
                {
                    'type': 'line',
                    'markerType': 'none',
                    'dataPoints': self._datapoints(
                        label=uuid, task=task, series=task.series[stat]),
                }
                for uuid, task in tasks.iteritems()
            ],
            'title': {
                'text': '{}: {} {}'.format(label, stat[0], stat[1]),
            },
            'zoomEnabled': True,
        }

    def _axisY(self, tasks, stat):
        ymax = 1
        for task in tasks.itervalues():
            for pt in task.series[stat]:
                ymax = max(ymax, pt[1])
        ytick = math.exp((1+math.floor(math.log(ymax, 2)))*math.log(2))/4
        return {
            'gridColor': '#cccccc',
            'gridThickness': 1,
            'interval': ytick,
            'minimum': 0,
            'maximum': ymax,
            'valueFormatString': "''",
        }

    def _datapoints(self, label, task, series):
        points = [
            {'x': pt[0].total_seconds(), 'y': pt[1]}
            for pt in series]
        if len(points) > 0:
            points[-1]['markerType'] = 'cross'
            points[-1]['markerSize'] = 12
        return points
