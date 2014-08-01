#!/bin/bash
set -e
export PATH="$PATH":/usr/src/arvados/services/crunch
export PERLLIB=/usr/src/arvados/sdk/perl/lib
export ARVADOS_API_HOST=api
export ARVADOS_API_HOST_INSECURE=yes
export CRUNCH_DISPATCH_LOCKFILE=/var/lock/crunch-dispatch

if [[ ! -e $CRUNCH_DISPATCH_LOCKFILE ]]; then
  touch $CRUNCH_DISPATCH_LOCKFILE
fi

export CRUNCH_JOB_BIN=/usr/src/arvados/services/crunch/crunch-job
export HOME=`pwd`
fuser -TERM -k $CRUNCH_DISPATCH_LOCKFILE || true

# Give the compute nodes some time to start up
sleep 5

cd /usr/src/arvados/services/api
export RAILS_ENV=production
/usr/local/rvm/bin/rvm-exec default bundle install
exec /usr/local/rvm/bin/rvm-exec default bundle exec ./script/crunch-dispatch.rb 2>&1

