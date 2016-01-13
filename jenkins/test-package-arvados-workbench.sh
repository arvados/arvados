#!/bin/sh
set -e
cd /var/www/arvados-workbench/current/
/usr/local/rvm/bin/rvm-exec default bundle install
/usr/local/rvm/bin/rvm-exec default bundle list >$ARV_PACKAGES_DIR/arvados-workbench.gems
