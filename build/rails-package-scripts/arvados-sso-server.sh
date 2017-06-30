#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This file declares variables common to all scripts for one Rails package.

PACKAGE_NAME=arvados-sso-server
INSTALL_PATH=/var/www/arvados-sso
CONFIG_PATH=/etc/arvados/sso
DOC_URL="http://doc.arvados.org/install/install-sso.html#configure"
RAILSPKG_DATABASE_LOAD_TASK=db:schema:load
RAILSPKG_SUPPORTS_CONFIG_CHECK=0
