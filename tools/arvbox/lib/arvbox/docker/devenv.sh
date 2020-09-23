#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox
flock $ARVADOS_CONTAINER_PATH/createusers.lock /usr/local/lib/arvbox/createusers.sh --no-chown

if [[ -n "$*" ]] ; then
    exec su --preserve-environment arvbox -c "$*"
else
    exec su --login arvbox
fi
