from __future__ import print_function

import cgi
import json
import math
import pkg_resources

from crunchstat_summary import logger


class ChartJS(object):
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/canvasjs/1.7.0/canvasjs.min.js'

    def html_header(self, label):
        return '''<!doctype html><html><head>
        <title>{} stats</title>
        <script type="text/javascript" src="{}"></script>
        <script type="text/javascript">var sections = [
        '''.format(cgi.escape(label), self.JSLIB)

    def html_trailer(self):
        return '''];\n{}</script>
        </head><body></body></html>
        '''.format(pkg_resources.resource_string('crunchstat_summary', 'chartjs.js'))

    def section(self, summarizer):
        return json.dumps({
                'label': summarizer.long_label(),
                'charts': self.charts(summarizer.label, summarizer.tasks),
            })

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

    def charts(self, label, tasks):
        return [
            {
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
            for stat in (('cpu', 'user+sys__rate'),
                         ('mem', 'rss'),
                         ('net:eth0', 'tx+rx__rate'),
                         ('net:keep0', 'tx+rx__rate'))]

    def _datapoints(self, label, task, series):
        points = [
            {'x': pt[0].total_seconds(), 'y': pt[1]}
            for pt in series]
        if len(points) > 0:
            points[-1]['markerType'] = 'cross'
            points[-1]['markerSize'] = 12
        return points
