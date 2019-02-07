#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

flock /var/lib/arvados/createusers.lock /usr/local/lib/arvbox/createusers.sh

if [[ -n "$*" ]] ; then
    exec su --preserve-environment arvbox -c "$*"
else
    exec su --login arvbox
fi
