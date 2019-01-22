#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

PYTHON=`ls /usr/share/python2.7/dist/python-arvados-fuse/bin/python2.7`

if [ "$PYTHON" != "" ]; then
  set -e
  exec $PYTHON <<EOF
import arvados_fuse
print "Successfully imported arvados_fuse"
EOF
else
	exit 1
fi
