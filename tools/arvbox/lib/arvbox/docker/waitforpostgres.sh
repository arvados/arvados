#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

. /usr/local/lib/arvbox/common.sh

while ! psql postgres -c\\du >/dev/null 2>/dev/null ; do
    sleep 1
done

while ! test -s $ARVADOS_CONTAINER_PATH/server-cert-${localip}.pem ; do
    sleep 1
done
