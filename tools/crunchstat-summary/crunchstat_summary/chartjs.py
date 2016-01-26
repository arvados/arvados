from __future__ import print_function

import cgi
import json
import pkg_resources

from crunchstat_summary import logger


class ChartJS(object):
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/canvasjs/1.7.0/canvasjs.min.js'

    def __init__(self, label, summarizers):
        self.label = label
        self.summarizers = summarizers

    def html(self):
        return '''<!doctype html><html><head>
        <title>{} stats</title>
        <script type="text/javascript" src="{}"></script>
        <script type="text/javascript">{}</script>
        </head><body></body></html>
        '''.format(cgi.escape(self.label), self.JSLIB, self.js())

    def js(self):
        return 'var sections = {};\n{}'.format(
            json.dumps(self.sections()),
            pkg_resources.resource_string('crunchstat_summary', 'chartjs.js'))

    def sections(self):
        return [
            {
                'label': s.long_label(),
                'charts': self.charts(s.label, s.tasks),
            }
            for s in self.summarizers]

    def charts(self, label, tasks):
        return [
            {
                'axisY': {
                    'minimum': 0,
                },
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
