#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

arv-put --version

/usr/share/python2.7/dist/python-arvados-python-client/bin/python2.7 << EOF
import arvados
print "Successfully imported arvados"
EOF
