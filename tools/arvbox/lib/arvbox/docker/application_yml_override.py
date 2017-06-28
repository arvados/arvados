#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import yaml

try:
    with open("application.yml.override") as f:
        b = yaml.load(f)
except IOError:
    exit()

with open("application.yml") as f:
    a = yaml.load(f)

def recursiveMerge(a, b):
    if isinstance(a, dict) and isinstance(b, dict):
        for k in b:
            print k
            a[k] = recursiveMerge(a.get(k), b[k])
        return a
    else:
        return b

with open("application.yml", "w") as f:
    yaml.dump(recursiveMerge(a, b), f)
