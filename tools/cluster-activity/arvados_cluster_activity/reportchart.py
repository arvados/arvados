#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import json
import importlib.resources
from datetime import datetime
from arvados._internal.report_template import ReportTemplate

sortablecss = """
<style>
@charset "UTF-8";
.sortable thead th:not(.no-sort) {
  cursor: pointer;
}
.sortable thead th:not(.no-sort)::after, .sortable thead th:not(.no-sort)::before {
  transition: color 0.1s ease-in-out;
  font-size: 1.2em;
  color: transparent;
}
.sortable thead th:not(.no-sort)::after {
  margin-left: 3px;
  content: "▸";
}
.sortable thead th:not(.no-sort):hover::after {
  color: inherit;
}
.sortable thead th:not(.no-sort)[aria-sort=descending]::after {
  color: inherit;
  content: "▾";
}
.sortable thead th:not(.no-sort)[aria-sort=ascending]::after {
  color: inherit;
  content: "▴";
}
.sortable thead th:not(.no-sort).indicator-left::after {
  content: "";
}
.sortable thead th:not(.no-sort).indicator-left::before {
  margin-right: 3px;
  content: "▸";
}
.sortable thead th:not(.no-sort).indicator-left:hover::before {
  color: inherit;
}
.sortable thead th:not(.no-sort).indicator-left[aria-sort=descending]::before {
  color: inherit;
  content: "▾";
}
.sortable thead th:not(.no-sort).indicator-left[aria-sort=ascending]::before {
  color: inherit;
  content: "▴";
}

table.aggtable td:nth-child(2) {
  text-align: right;
}

table.active-projects td:nth-child(4),
table.active-projects td:nth-child(5) {
  text-align: right;
  padding-right: 6em;
}

table.single-project td:nth-child(3),
table.single-project td:nth-child(4) {
  text-align: right;
  padding-right: 6em;
}

table.active-projects th:nth-child(4),
table.active-projects th:nth-child(5) {
  text-align: left;
}

table.project td:nth-child(3),
table.project td:nth-child(4),
table.project td:nth-child(5),
table.project td:nth-child(6),
table.project td:nth-child(7) {
  text-align: right;
  padding-right: 6em;
}

table.project th:nth-child(3),
table.project th:nth-child(4),
table.project th:nth-child(5),
table.project th:nth-child(6),
table.project th:nth-child(7) {
  text-align: left;
}
</style>
"""

def date_export(item):
    if isinstance(item, datetime):
        return """@new Date("{}")@""".format(item.strftime("%Y-%m-%dT%H:%M:%SZ"))

class ReportChart(ReportTemplate):
    CSS = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.css'
    JSLIB = 'https://cdnjs.cloudflare.com/ajax/libs/dygraph/2.0.0/dygraph.min.js'
    JSASSETS = ['synchronizer.js', 'dygraphs.js', 'sortable.js']

    def __init__(self, label, cards, graphs):
        super(ReportChart, self).__init__(label)
        self.cards = cards
        self.graphs = graphs

    def sections(self):
        return [
            {
                'label': k[0],
                'charts': [self.chartdata(k, v)]
            }
            for k,v in self.graphs.items()]

    def chartdata(self, label, stats):
        return {
            'data': stats,
            'options': {
                'legend': 'always',
                'connectSeparatedPoints': True,
                'labels': ['date', label[1]],
                'includeZero': True,
                'title': label[0]
            },
        }

    def js(self):


        return '''
        <script type="text/javascript" src="{jslib}"></script>
        <script type="text/javascript">
        var chartdata = {chartdata};\n{jsassets}
        </script>'''.format(
            jslib=self.JSLIB,
            chartdata=json.dumps(self.sections(), default=date_export).replace('"@', '').replace('@"', '').replace('\\"', '"'),
            jsassets='\n'.join(
                importlib.resources.read_text('arvados_cluster_activity', jsa)
                for jsa in self.JSASSETS
            ),
        )

    def style(self):
        return '\n'.join((super().style(),
                         sortablecss,
                         '<link rel="stylesheet" href="{}">\n'.format(self.CSS)))
