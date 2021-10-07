#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
sleep 2
set -eux -o pipefail

. /usr/local/lib/arvbox/common.sh
. /usr/local/lib/arvbox/go-setup.sh

(cd /usr/local/bin && ln -sf arvados-server keepstore)

if test "$1" = "--only-deps" ; then
    exit
fi

mkdir -p $ARVADOS_CONTAINER_PATH/$1

exec /usr/local/bin/keepstore
