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

    def chartdata(self, label, tasks, stats):
        '''For Crunch2, label is the name of container request,
        tasks is the top level container and
        stats is index by a tuple of (category, metric).
        '''
        return {
            'data': self._collate_data(tasks, stats),
            'options': {
                'legend': 'always',
                'connectSeparatedPoints': True,
                'labels': ['elapsed'] +  stats[1],
                'title': '{}: {}'.format(label, stats[0]),
            },
        }

    def _collate_data(self, tasks, stats):
        data = []
        nulls = []
        # uuid is category for crunch2
        for uuid, task in tasks.items():
            # All stats in a category are assumed to have the same time base and same number of samples
            category = stats[0]
            series_names = stats[1]
            sn0 = series_names[0]
            series = task.series[(category,sn0)]
            for i in range(len(series)):
                pt = series[i]
                vals = [task.series[(category,stat)][i][1] for stat in series_names[1:]]
                data.append([pt[0].total_seconds()] + nulls + [pt[1]] + vals)
            nulls.append(None)
        return sorted(data)
