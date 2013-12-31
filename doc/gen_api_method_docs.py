#! /usr/bin/env python

# gen_api_method_docs.py
#
# Generate docs for Arvados methods.
#
# This script will retrieve the discovery document at
# https://localhost:9900/discovery/v1/apis/arvados/v1/rest
# and will generate Textile documentation files in the current
# directory.

import requests
import re
import os
import pprint

r = requests.get('https://localhost:9900/discovery/v1/apis/arvados/v1/rest',
                 verify=False)
if r.status_code != 200:
    raise Exception('Bad status code %d: %s' % (r.status_code, r.text))

if 'application/json' not in r.headers.get('content-type', ''):
    raise Exception('Unexpected content type: %s: %s' %
                    (r.headers.get('content-type', ''), r.text))

api = r.json

resource_num = 0
for resource in sorted(api[u'resources']):
    resource_num = resource_num + 1
    out_fname = resource + '.textile'
    if os.path.exists(out_fname):
        print "PATH EXISTS ", out_fname
        next
    outf = open(out_fname, 'w')
    outf.write(
"""---
layout: default
navsection: api
navmenu: API Methods
title: "{resource}"
navorder: {resource_num}
---

h1. {resource}

Required arguments are displayed in %{{background:#ccffcc}}green%.

""".format(resource_num=resource_num, resource=resource))

    methods = api['resources'][resource]['methods']
    for method in sorted(methods.keys()):
        methodinfo = methods[method]
        outf.write(
"""
h2. {method}

{description}

Arguments:

table(table table-bordered table-condensed).
|_. Argument |_. Type |_. Description |_. Location |_. Example |
""".format(
    method=method, description=methodinfo['description']))

        required = []
        notrequired = []
        for param, paraminfo in methodinfo['parameters'].iteritems():
            paraminfo.setdefault(u'description', '')
            paraminfo.setdefault(u'location', '')
            limit = ''
            if paraminfo.get('minimum', '') or paraminfo.get('maximum', ''):
                limit = "range {0}-{1}".format(
                    paraminfo.get('minimum', ''),
                    paraminfo.get('maximum', 'unlimited'))
            if paraminfo.get('default', ''):
                if limit:
                    limit = limit + '; '
                limit = limit + 'default %d' % paraminfo['default']
            if limit:
                paraminfo['type'] = '{0} ({1})'.format(
                    paraminfo['type'], limit)

            row = "|{param}|{type}|{description}|{location}||\n".format(
                param=param, **paraminfo)
            if paraminfo.get('required', False):
                required.append(row)
            else:
                notrequired.append(row)

        for row in sorted(required):
            outf.write("{background:#ccffcc}." + row)
        for row in sorted(notrequired):
            outf.write(row)

        # pprint.pprint(methodinfo)

    outf.close()
    print "wrote ", out_fname


