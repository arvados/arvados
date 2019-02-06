#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

arvados-node-manager --version

exec /usr/share/python2.7/dist/arvados-node-manager/bin/python2.7 <<EOF
import libcloud.compute.types
import libcloud.compute.providers
libcloud.compute.providers.get_driver(libcloud.compute.types.Provider.AZURE_ARM)
print "Successfully imported compatible libcloud library"
EOF
