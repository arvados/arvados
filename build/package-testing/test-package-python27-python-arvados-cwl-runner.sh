#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e
arvados-cwl-runner --version
set +e

PYTHON=`ls /usr/share/python2.7/dist/python-arvados-cwl-runner/bin/python2.7`

if [ "$PYTHON" != "" ]; then
  set -e
  exec $PYTHON <<EOF
import arvados_cwl
print "arvados-cwl-runner version", arvados_cwl.__version__
EOF
else
	exit 1
fi
