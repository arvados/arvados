#!/bin/sh
set -e
cd /var/www/arvados-api/current/
yum install --assumeyes httpd
yum reinstall --assumeyes arvados-api-server
/usr/local/rvm/bin/rvm-exec default bundle list >$ARV_PACKAGES_DIR/arados-api-server.gems
