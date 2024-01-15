#!/opt/arvados-py/bin/python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import yaml
import sys

fn = sys.argv[1]

try:
    with open(fn+".override") as f:
        b = yaml.safe_load(f)
except IOError:
    exit()

with open(fn) as f:
    a = yaml.safe_load(f)

def recursiveMerge(a, b):
    if isinstance(a, dict) and isinstance(b, dict):
        for k in b:
            print(k)
            a[k] = recursiveMerge(a.get(k), b[k])
        return a
    else:
        return b

with open(fn, "w") as f:
    yaml.safe_dump(recursiveMerge(a, b), f)
