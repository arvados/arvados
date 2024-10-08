# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import importlib.resources
import json
from arvados._internal.report_template import ReportTemplate

class DygraphsChart(ReportTemplate):
    """Crunchstat report using dygraphs for charting.
    """

    CSS = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.css'
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.js'
    JSASSETS = ['synchronizer.js','dygraphs.js']

    def __init__(self, label, summarizers, beforechart, afterchart):
        super().__init__(label)
        self.summarizers = summarizers
        self.beforechart = beforechart
        self.afterchart = afterchart

    def html(self):
        self.cards.extend(self.beforechart)
        self.cards.append("""
                <h2>Graph</h2>
                <div id="chart"></div>
            """)
        self.cards.extend(self.afterchart)

        return super().html()

    def js(self):
        return '''
        <script type="text/javascript" src="{jslib}"></script>
        <script type="text/javascript">
        var chartdata = {chartdata};\n{jsassets}
        </script>'''.format(
            jslib=self.JSLIB,
            chartdata=json.dumps(self.sections()),
            jsassets='\n'.join(
                importlib.resources.read_text('crunchstat_summary', jsa, encoding='utf-8')
                for jsa in self.JSASSETS
            ),
        )

    def sections(self):
        return [
            {
                'label': s.long_label(),
                'charts': [
                    self.chartdata(s.label, s.tasks, stat)
                    for stat in (('cpu', ['user+sys__rate', 'user__rate', 'sys__rate']),
                                 ('mem', ['rss']),
                                 ('net:eth0', ['tx+rx__rate','rx__rate','tx__rate']),
                                 ('net:keep0', ['tx+rx__rate','rx__rate','tx__rate']),
                                 ('statfs', ['used', 'total']),
                                 )
                    ],
            }
            for s in self.summarizers]

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
                'includeZero': True,
                'title': '{}: {}'.format(label, stats[0]) if label else stats[0],
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

    def style(self):
        return '\n'.join((super().style(),
                         '<link rel="stylesheet" href="{}">\n'.format(self.CSS)))
