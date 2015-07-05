#!/bin/bash

RELEASE_PATH=/var/www/arvados-workbench/current
SHARED_PATH=/var/www/arvados-workbench/shared
CONFIG_PATH=/etc/arvados/workbench/

echo "Assumption: nginx is configured to serve workbench.`hostname` from /var/www/workbench.`hostname`/current"
echo "Assumption: /var/www/`hostname` is symlinked to /var/www/arvados-workbench"
echo "Assumption: configuration files are in /etc/arvados/workbench/"
echo "Assumption: nginx and passenger run as the www-data user"
echo

echo "Copying files from $CONFIG_PATH"
cp -f $CONFIG_PATH/application.yml $RELEASE_PATH/config/application.yml
cp -f $RELEASE_PATH/config/environments/production.rb.example $RELEASE_PATH/config/environments/production.rb
echo "Done."

# Before we do anything else, make sure some directories and files are in place
if [[ ! -e $SHARED_PATH/log ]]; then mkdir -p $SHARED_PATH/log; fi
if [[ ! -e $RELEASE_PATH/tmp ]]; then mkdir -p $RELEASE_PATH/tmp; fi
if [[ ! -e $RELEASE_PATH/log ]]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
if [[ ! -e $SHARED_PATH/log/production.log ]]; then touch $SHARED_PATH/log/production.log; fi

echo "Running bundle install"
(cd $RELEASE_PATH && RAILS_ENV=production bundle install --path $SHARED_PATH/vendor_bundle)
echo "Done."

# We do not need to precompile assets, they are already part of the package.

echo "Ensuring directory and file permissions"
chown www-data:www-data $RELEASE_PATH/config/environment.rb
chown www-data:www-data $RELEASE_PATH/config.ru
chown www-data:www-data $RELEASE_PATH/config/database.yml
chown www-data:www-data $RELEASE_PATH/Gemfile.lock
chown -R www-data:www-data $RELEASE_PATH/tmp
chown -R www-data:www-data $SHARED_PATH/log
chown www-data:www-data $RELEASE_PATH/db/schema.rb
chmod 644 $SHARED_PATH/log/*
echo "Done."

echo "Running sanity check"
(cd $RELEASE_PATH && RAILS_ENV=production bundle exec rake config:check)
SANITY_CHECK_EXIT_CODE=$?
echo "Done."

if [[ "$SANITY_CHECK_EXIT_CODE" != "0" ]]; then
  echo "Sanity check failed, aborting. Please roll back to the previous version of the package."
  exit $SANITY_CHECK_EXIT_CODE
fi

# We do not need to run db:migrate because Workbench is stateless

echo "Restarting nginx"
service nginx restart
echo "Done."

