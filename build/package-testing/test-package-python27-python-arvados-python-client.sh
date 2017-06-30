#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec python2.7 <<EOF
import arvados
print "Successfully imported arvados"
EOF
