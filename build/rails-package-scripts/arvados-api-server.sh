#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This file declares variables common to all scripts for one Rails package.

PACKAGE_NAME=arvados-api-server
INSTALL_PATH=/var/www/arvados-api
CONFIG_PATH=/etc/arvados/api
DOC_URL="http://doc.arvados.org/install/install-api-server.html#configure"

RAILSPKG_DATABASE_LOAD_TASK=db:structure:load
setup_extra_conffiles() {
  # Rails 5.2 does not tolerate dangling symlinks in the initializers directory, and this one
  # can still be there, left over from a previous version of the API server package.
  rm -f $RELEASE_PATH/config/initializers/omniauth.rb
}

setup_before_nginx_restart() {
  # initialize git_internal_dir
  # usually /var/lib/arvados/internal.git (set in application.default.yml )
  if [ "$APPLICATION_READY" = "1" ]; then
      GIT_INTERNAL_DIR=$($COMMAND_PREFIX bin/rake config:dump 2>&1 | grep GitInternalDir | awk '{ print $2 }' |tr -d '"')
      if [ ! -e "$GIT_INTERNAL_DIR" ]; then
        run_and_report "Creating git_internal_dir '$GIT_INTERNAL_DIR'" \
          mkdir -p "$GIT_INTERNAL_DIR"
        run_and_report "Initializing git_internal_dir '$GIT_INTERNAL_DIR'" \
          git init --quiet --bare $GIT_INTERNAL_DIR
      else
        echo "Initializing git_internal_dir $GIT_INTERNAL_DIR: directory exists, skipped."
      fi
      run_and_report "Making sure '$GIT_INTERNAL_DIR' has the right permission" \
         chown -R "$WWW_OWNER:" "$GIT_INTERNAL_DIR"
  else
      echo "Initializing git_internal_dir... skipped."
  fi
}
