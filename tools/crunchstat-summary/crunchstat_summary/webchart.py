# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

try:
    from html import escape
except ImportError:
    from cgi import escape

import json
from typing import ItemsView
import pkg_resources


class WebChart(object):
    """Base class for a web chart.

    Subclasses must assign JSLIB and JSASSETS, and override the
    chartdata() method.
    """
    JSLIB = None
    JSASSET = None

    STYLE = '''
        body {
          background: #fafafa;
          font-family: "Roboto", "Helvetica", "Arial", sans-serif;
          font-size: 0.875rem;
          color: rgba(0, 0, 0, 0.87);
          font-weight: 400;
        }
        .card {
          background: #ffffff;
          box-shadow: 0px 1px 5px 0px rgba(0,0,0,0.2),0px 2px 2px 0px rgba(0,0,0,0.14),0px 3px 1px -2px rgba(0,0,0,0.12);
          border-radius: 4px;
          margin: 20px;
        }
        .content {
          padding: 2px 16px 8px 16px;
        }
        table {
          border-spacing: 0px;
        }
        tr {
          height: 36px;
          text-align: left;
        }
        th {
          padding-right: 4em;
          border-top: 1px solid rgba(224, 224, 224, 1);
        }
        td {
          padding-right: 2em;
          border-top: 1px solid rgba(224, 224, 224, 1);
        }
        #chart {
          margin-left: -20px;
        }
    '''

    def __init__(self, label, summarizers):
        self.label = label
        self.summarizers = summarizers

    def cardlist(self, items):
        if not isinstance(items, list):
            items = [items]

        return "\n".join(
                """<div class="card">
          <div class="content">
          {}
          </div>
        </div>""".format(i) for i in items)

    def html(self, beforechart='', afterchart=''):
        return '''<!doctype html><html><head>
        <title>{} stats</title>
        <script type="text/javascript" src="{}"></script>
        <script type="text/javascript">{}</script>
        <style>
        {}
        </style>
        {}
        </head>
        <body>
        <div class="card">
          <div class="content">
            <h1>{}</h1>
          </div>
        </div>
        {}
        <div class="card">
          <div class="content">
            <h2>Graph</h2>
            <div id="chart"></div>
          </div>
        </div>
        {}
        </body>
        </html>
        '''.format(escape(self.label),
                   self.JSLIB,
                   self.js(),
                   self.STYLE,
                   self.headHTML(),
                   escape(self.label),
                   self.cardlist(beforechart),
                   self.cardlist(afterchart))

    def js(self):
        return 'var chartdata = {};\n{}'.format(
            json.dumps(self.sections()),
            '\n'.join([pkg_resources.resource_string('crunchstat_summary', jsa).decode('utf-8') for jsa in self.JSASSETS]))

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

    def chartdata(self, label, tasks, stat):
        """Return chart data for the given tasks.

        The returned value will be available on the client side as an
        element of the "chartdata" array.
        """
        raise NotImplementedError()

    def headHTML(self):
        """Return extra HTML text to include in HEAD."""
        return ''
