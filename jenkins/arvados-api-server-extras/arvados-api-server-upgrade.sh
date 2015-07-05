#!/bin/bash

RELEASE_PATH=/var/www/arvados-api/current
SHARED_PATH=/var/www/arvados-api/shared
CONFIG_PATH=/etc/arvados/api/

echo "Assumption: nginx is configured to serve `hostname` from /var/www/`hostname`/current"
echo "Assumption: /var/www/`hostname` is symlinked to /var/www/arvados-api"
echo "Assumption: configuration files are in /etc/arvados/api/"
echo "Assumption: nginx and passenger run as the www-data user"
echo

echo "Copying files from $CONFIG_PATH"
cp -f $CONFIG_PATH/database.yml $RELEASE_PATH/config/database.yml
cp -f $RELEASE_PATH/config/environments/production.rb.example $RELEASE_PATH/config/environments/production.rb
cp -f $CONFIG_PATH/application.yml $RELEASE_PATH/config/application.yml
cp -f $CONFIG_PATH/omniauth.rb $RELEASE_PATH/config/initializers/omniauth.rb
echo "Done."

# Before we do anything else, make sure some directories and files are in place
if [[ ! -e $SHARED_PATH/log ]]; then mkdir -p $SHARED_PATH/log; fi
if [[ ! -e $RELEASE_PATH/tmp ]]; then mkdir -p $RELEASE_PATH/tmp; fi
if [[ ! -e $RELEASE_PATH/log ]]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
if [[ ! -e $SHARED_PATH/log/production.log ]]; then touch $SHARED_PATH/log/production.log; fi

echo "Running bundle install"
(cd $RELEASE_PATH && RAILS_ENV=production bundle install --path $SHARED_PATH/vendor_bundle)
echo "Done."

echo "Precompiling assets"
# precompile assets; thankfully this does not take long
(cd $RELEASE_PATH; RAILS_ENV=production bundle exec rake assets:precompile)
echo "Done."

echo "Ensuring directory and file permissions"
# Ensure correct ownership of a few files
chown www-data:www-data $RELEASE_PATH/config/environment.rb
chown www-data:www-data $RELEASE_PATH/config.ru
chown www-data:www-data $RELEASE_PATH/config/database.yml
chown www-data:www-data $RELEASE_PATH/Gemfile.lock
chown -R www-data:www-data $RELEASE_PATH/tmp
chown -R www-data:www-data $SHARED_PATH/log
chown www-data:www-data $RELEASE_PATH/db/structure.sql
chmod 644 $SHARED_PATH/log/*
echo "Done."

echo "Running sanity check"
(cd $RELEASE_PATH && RAILS_ENV=production bundle exec rake config:check)
SANITY_CHECK_EXIT_CODE=$?
echo "Done."

if [[ "$SANITY_CHECK_EXIT_CODE" != "0" ]]; then
  echo "Sanity check failed, aborting. Please roll back to the previous version of the package."
  echo "The database has not been migrated yet, so reinstalling the previous version is safe."
  exit $SANITY_CHECK_EXIT_CODE
fi

echo "Starting db:migrate"
(cd $RELEASE_PATH && bundle exec rake RAILS_ENV=production  db:migrate)
echo "Done."

echo "Restarting nginx"
service nginx restart
echo "Done."
