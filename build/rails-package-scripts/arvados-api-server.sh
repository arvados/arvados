#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This file declares variables common to all scripts for one Rails package.

PACKAGE_NAME=arvados-api-server
INSTALL_PATH=/var/www/arvados-api
CONFIG_PATH=/etc/arvados/api
DOC_URL="http://doc.arvados.org/install/install-api-server.html#configure"
RELEASE_PATH=$INSTALL_PATH/current
RELEASE_CONFIG_PATH=$RELEASE_PATH/config
SHARED_PATH=$INSTALL_PATH/shared
