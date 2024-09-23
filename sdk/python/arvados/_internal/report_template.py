# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

try:
    from html import escape
except ImportError:
    from cgi import escape

import json
from typing import ItemsView

class ReportTemplate(object):
    """Base class for HTML reports produced by Arvados reporting tools.

    Used by crunchstat-summary and cluster-activity.

    """

    STYLE = '''
    <style>
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
    </style>
    '''

    def __init__(self, label):
        self.label = label
        self.cards = []

    def cardlist(self, items):
        if not isinstance(items, list):
            items = [items]

        return "\n".join(
                """
                <div class="card">
                  <div class="content">
{}
                  </div>
                </div>""".format(i) for i in items)

    def html(self):
        return '''<!doctype html>
<html>
  <head>
    <title>{label}</title>

{js}

{style}

{header}

  </head>

  <body>
  <div class="card">
    <div class="content">
      <h1>{label}</h1>
    </div>
  </div>

{cards}

  </body>
</html>
        '''.format(label=escape(self.label),
                   js=self.js(),
                   style=self.style(),
                   header=self.headHTML(),
                   cards=self.cardlist(self.cards))

    def js(self):
        return ''

    def style(self):
        return self.STYLE

    def headHTML(self):
        """Return extra HTML text to include in HEAD."""
        return ''
