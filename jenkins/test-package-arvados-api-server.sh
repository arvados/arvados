#!/bin/sh
set -e
cd /var/www/arvados-api/current/
/usr/local/rvm/bin/rvm-exec default bundle install
/usr/local/rvm/bin/rvm-exec default bundle list >$ARV_PACKAGES_DIR/arados-api-server.gems
