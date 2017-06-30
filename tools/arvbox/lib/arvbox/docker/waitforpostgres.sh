#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

while ! psql postgres -c\\du >/dev/null 2>/dev/null ; do
    sleep 1
done
