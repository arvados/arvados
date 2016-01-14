#!/bin/sh
set -e
cd /var/www/arvados-workbench/current/
yum install --assumeyes httpd
yum reinstall --assumeyes arvados-workbench
/usr/local/rvm/bin/rvm-exec default bundle list >$ARV_PACKAGES_DIR/arvados-workbench.gems
