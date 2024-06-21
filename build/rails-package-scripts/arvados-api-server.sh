#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This file declares variables common to all scripts for one Rails package.

PACKAGE_NAME=arvados-api-server
INSTALL_PATH=/var/www/arvados-api
CONFIG_PATH=/etc/arvados/api
DOC_URL="http://doc.arvados.org/install/install-api-server.html#configure"

RAILSPKG_DATABASE_LOAD_TASK=db:schema:load
setup_extra_conffiles() {
  # Rails 5.2 does not tolerate dangling symlinks in the initializers directory, and this one
  # can still be there, left over from a previous version of the API server package.
  rm -f $RELEASE_PATH/config/initializers/omniauth.rb
}
