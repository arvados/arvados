#!/bin/sh
# This file declares variables common to all scripts for one Rails package.

PACKAGE_NAME=arvados-api-server
INSTALL_PATH=/var/www/arvados-api
CONFIG_PATH=/etc/arvados/api
DOC_URL="http://doc.arvados.org/install/install-api-server.html#configure"

RAILSPKG_DATABASE_LOAD_TASK=db:structure:load
setup_extra_conffiles() {
    setup_conffile initializers/omniauth.rb
}
