from __future__ import print_function

import json
import pkg_resources


class ChartJS(object):
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/canvasjs/1.7.0/canvasjs.js'

    def __init__(self, label, tasks):
        self.label = label
        self.tasks = tasks

    def html(self):
        return '''<!doctype html><html><head>
        <title>{} stats</title>
        <script type="text/javascript" src="{}"></script>
        <script type="text/javascript">{}</script>
        </head><body></body></html>
        '''.format(self.label, self.JSLIB, self.js())

    def js(self):
        return 'var chartData = {};\n{}'.format(
            json.dumps(self.chartData()),
            pkg_resources.resource_string('crunchstat_summary', 'chartjs.js'))

    def chartData(self):
        maxpts = 0
        for task in self.tasks.itervalues():
            for series in task.series.itervalues():
                maxpts = max(maxpts, len(series))
        return [
            {
                'title': {
                    'text': '{}: {} {}'.format(self.label, stat[0], stat[1]),
                },
                'data': [
                    {
                        'type': 'line',
                        'dataPoints': [
                            {'x': pt[0].total_seconds(), 'y': pt[1]}
                            for pt in task.series[stat]]
                    }
                    for label, task in self.tasks.iteritems()
                ],
            }
            for stat in (('cpu', 'user+sys__rate'),
                         ('net:eth0', 'tx+rx__rate'),
                         ('mem', 'rss'))
        ]
