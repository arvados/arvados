#!/usr/bin/env python

import re
import sys
import arvados.collection
from arvados.keep import KeepLocator

for collectionsWithMissing in sys.argv[1:]:

    g = re.match(r"\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d-\d\d:\d\d_(.....-.....-...............)_missing\.txt", collectionsWithMissing)

    collection = g.group(1)

    blocklist = open(collectionsWithMissing)

    missingblocks = set()
    for b in blocklist:
        missingblocks.add(b.strip())

    def scanfiles(name, cur):
        if isinstance(cur, arvados.collection.ArvadosFile):
            segs = cur.segments()
            for s in segs:
                st = KeepLocator(s.locator).stripped()
                if st in missingblocks:
                    print "\"%s\", \"%s\", \"%s\"" % (collection, name, st)
        else:
            for k, d in cur.items():
                scanfiles("%s/%s" % (name, k), d)

    scanfiles(".", arvados.collection.CollectionReader(collection))
