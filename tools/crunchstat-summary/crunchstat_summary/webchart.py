# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import cgi
import json
import pkg_resources


class WebChart(object):
    """Base class for a web chart.

    Subclasses must assign JSLIB and JSASSETS, and override the
    chartdata() method.
    """
    JSLIB = None
    JSASSET = None

    def __init__(self, label, summarizers):
        self.label = label
        self.summarizers = summarizers

    def html(self):
        return '''<!doctype html><html><head>
        <title>{} stats</title>
        <script type="text/javascript" src="{}"></script>
        <script type="text/javascript">{}</script>
        {}
        </head><body></body></html>
        '''.format(cgi.escape(self.label),
                   self.JSLIB, self.js(), self.headHTML())

    def js(self):
        return 'var chartdata = {};\n{}'.format(
            json.dumps(self.sections()),
            '\n'.join([pkg_resources.resource_string('crunchstat_summary', jsa) for jsa in self.JSASSETS]))

    def sections(self):
        return [
            {
                'label': s.long_label(),
                'charts': [
                    self.chartdata(s.label, s.tasks, stat)
                    for stat in (('cpu', 'user+sys__rate'),
                                 ('mem', 'rss'),
                                 ('net:eth0', 'tx+rx__rate'),
                                 ('net:keep0', 'tx+rx__rate'))],
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
