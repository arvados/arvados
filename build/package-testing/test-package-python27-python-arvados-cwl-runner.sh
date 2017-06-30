#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec python <<EOF
import arvados_cwl
print "arvados-cwl-runner version", arvados_cwl.__version__
EOF
