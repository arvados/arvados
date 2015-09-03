#!/bin/sh

RELEASE_PATH=/var/www/arvados-sso/current
rm -f $RELEASE_PATH/config/database.yml
rm -f $RELEASE_PATH/config/environments/production.rb
rm -f $RELEASE_PATH/config/application.yml
rm -rf $RELEASE_PATH/public/assets/
rm -rf $RELEASE_PATH/tmp
rm $RELEASE_PATH/log

