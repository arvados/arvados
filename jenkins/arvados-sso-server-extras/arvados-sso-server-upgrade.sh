#!/bin/bash

set -e

if [ -e /etc/redhat-release ]; then
    WWW_OWNER=nginx:nginx
else
    # Assume we're on a Debian-based system for now.
    WWW_OWNER=www-data:www-data
fi

NGINX_SERVICE=${NGINX_SERVICE:-$(service --status-all 2>/dev/null \
    | grep -Eo '\bnginx[^[:space:]]*' || true)}
if [ -z "$NGINX_SERVICE" ]; then
    cat >&2 <<EOF
Error: nginx service not found. Aborting.
Set NGINX_SERVICE to the name of the service hosting the Rails server.
EOF
    exit 1
elif [ "$NGINX_SERVICE" != "$(echo "$NGINX_SERVICE" | head -n 1)" ]; then
    cat >&2 <<EOF
Error: multiple nginx services found. Aborting.
Set NGINX_SERVICE to the name of the service hosting the Rails server.
EOF
    exit 1
fi

RELEASE_PATH=/var/www/arvados-sso/current
SHARED_PATH=/var/www/arvados-sso/shared
CONFIG_PATH=/etc/arvados/sso/

echo
echo "Assumption: $NGINX_SERVICE is configured to serve your SSO server URL from"
echo "            /var/www/arvados-sso/current"
echo "Assumption: configuration files are in /etc/arvados/sso/"
echo "Assumption: $NGINX_SERVICE and passenger run as $WWW_OWNER"
echo

echo "Copying files from $CONFIG_PATH ..."
cp -f $CONFIG_PATH/database.yml $RELEASE_PATH/config/database.yml
cp -f $RELEASE_PATH/config/environments/production.rb.example $RELEASE_PATH/config/environments/production.rb
cp -f $CONFIG_PATH/application.yml $RELEASE_PATH/config/application.yml
echo "... done."

# Before we do anything else, make sure some directories and files are in place
if [[ ! -e $SHARED_PATH/log ]]; then mkdir -p $SHARED_PATH/log; fi
if [[ ! -e $RELEASE_PATH/tmp ]]; then mkdir -p $RELEASE_PATH/tmp; fi
if [[ ! -e $RELEASE_PATH/log ]]; then ln -s $SHARED_PATH/log $RELEASE_PATH/log; fi
if [[ ! -e $SHARED_PATH/log/production.log ]]; then touch $SHARED_PATH/log/production.log; fi

cd "$RELEASE_PATH"
export RAILS_ENV=production

echo "Running bundle install ..."
bundle install --path $SHARED_PATH/vendor_bundle --quiet
echo "... done."

echo "Ensuring directory and file permissions ..."
# Ensure correct ownership of a few files
chown "$WWW_OWNER" $RELEASE_PATH/config/environment.rb
chown "$WWW_OWNER" $RELEASE_PATH/config.ru
chown "$WWW_OWNER" $RELEASE_PATH/config/database.yml
chown "$WWW_OWNER" $RELEASE_PATH/Gemfile.lock
chown -R "$WWW_OWNER" $RELEASE_PATH/tmp
chown -R "$WWW_OWNER" $SHARED_PATH/log
chown "$WWW_OWNER" $RELEASE_PATH/db/schema.rb
chmod 644 $SHARED_PATH/log/*
echo "... done."

# If we use `grep -q`, rake will write a backtrace on EPIPE.
if bundle exec rake db:migrate:status | grep '^database: ' >/dev/null; then
    echo "Starting db:migrate ..."
    bundle exec rake db:migrate
elif [ 0 -eq ${PIPESTATUS[0]} ]; then
    # The database exists, but the migrations table doesn't.
    echo "Setting up database ..."
    bundle exec rake db:schema:load db:seed
else
    echo "Error: Database is not ready to set up. Aborting." >&2
    exit 1
fi
echo "... done."

echo "Precompiling assets ..."
# precompile assets; thankfully this does not take long
bundle exec rake assets:precompile -q -s
echo "... done."

echo "Restarting nginx ..."
service "$NGINX_SERVICE" restart
echo "... done."
