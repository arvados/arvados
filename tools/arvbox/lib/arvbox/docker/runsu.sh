#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

HOSTUID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f4)
HOSTGID=$(ls -nd /usr/src/arvados | sed 's/ */ /' | cut -d' ' -f5)

export ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox

flock $ARVADOS_CONTAINER_PATH/createusers.lock /usr/local/lib/arvbox/createusers.sh

export HOME=$ARVADOS_CONTAINER_PATH

chown arvbox /dev/stderr

if test -z "$1" ; then
    exec chpst -u arvbox:arvbox:docker $0-service
else
    exec chpst -u arvbox:arvbox:docker $@
fi
