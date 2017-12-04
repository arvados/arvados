# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import crunchstat_summary.webchart


class DygraphsChart(crunchstat_summary.webchart.WebChart):
    CSS = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.css'
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.js'
    JSASSETS = ['synchronizer.js','dygraphs.js']

    def headHTML(self):
        return '<link rel="stylesheet" href="{}">\n'.format(self.CSS)

    def chartdata(self, label, tasks, stat):
        return {
            'data': self._collate_data(tasks, stat),
            'options': {
                'connectSeparatedPoints': True,
                'labels': ['elapsed']+[uuid for uuid, _ in tasks.iteritems()],
                'title': '{}: {} {}'.format(label, stat[0], stat[1]),
            },
        }

    def _collate_data(self, tasks, stat):
        data = []
        nulls = []
        for uuid, task in tasks.iteritems():
            for pt in task.series[stat]:
                data.append([pt[0].total_seconds()] + nulls + [pt[1]])
            nulls.append(None)
        return sorted(data)
