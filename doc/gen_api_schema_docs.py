#! /usr/bin/env python

# gen_api_schema_docs.py
#
# Generate Textile documentation pages for Arvados schema resources.

import requests
import re
import os

r = requests.get('https://localhost:9900/arvados/v1/schema',
                 verify=False)
if r.status_code != 200:
    raise Exception('Bad status code %d: %s' % (r.status_code, r.text))

if 'application/json' not in r.headers.get('content-type', ''):
    raise Exception('Unexpected content type: %s: %s' %
                    (r.headers.get('content-type', ''), r.text))

schema = r.json()
navorder = 0
for resource in sorted(schema.keys()):
    navorder = navorder + 1
    properties = schema[resource]
    res_api_endpoint = re.sub(r'([a-z])([A-Z])', r'\1_\2', resource).lower()
    outfile = "{}.textile".format(resource)
    if os.path.exists(outfile):
        outfile = "{}_new.textile".format(resource)
    print outfile, "..."
    with open(outfile, "w") as f:
        f.write("""---
layout: default
navsection: api
navmenu: Schema
title: {resource}
---

h1. {resource}

A **{resource}** represents...

h2. Methods

        See "REST methods for working with Arvados resources":{{{{site.baseurl}}}}/api/methods.html

API endpoint base: @https://{{{{ site.arvados_api_host }}}}/arvados/v1/{res_api_endpoint}@

h2. Creation

h3. Prerequisites

Prerequisites for creating a {resource}.

h3. Side effects

Side effects of creating a {resource}.

h2. Resources

Each {resource} has, in addition to the usual "attributes of Arvados resources":resources.html:

table(table table-bordered table-condensed).
|_. Attribute|_. Type|_. Description|_. Example|
""".format(
    resource=resource,
    navorder=navorder,
    res_api_endpoint=res_api_endpoint))

        for prop in properties:
            if prop not in ['id', 'uuid', 'href', 'kind', 'etag', 'self_link',
                            'owner_uuid', 'created_at',
                            'modified_by_client_uuid',
                            'modified_by_user_uuid',
                            'modified_at']:
                f.write('|{name}|{type}|||\n'.format(**prop))

