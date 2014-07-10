#!/bin/bash
set -e
export PATH="$PATH":/usr/local/arvados/src/services/crunch
export PERLLIB=/usr/local/arvados/src/sdk/perl/lib
export ARVADOS_API_HOST=qr1hi.arvadosapi.com
export CRUNCH_DISPATCH_LOCKFILE=/var/lock/crunch-dispatch

if [[ ! -e $CRUNCH_DISPATCH_LOCKFILE ]]; then
  touch $CRUNCH_DISPATCH_LOCKFILE
fi

export CRUNCH_JOB_BIN=/usr/local/arvados/src/services/crunch/crunch-job
export HOME=`pwd`
fuser -TERM -k $CRUNCH_DISPATCH_LOCKFILE || true

cd /usr/src/arvados/services/api
export RAILS_ENV=production
/usr/local/rvm/bin/rvm-exec default bundle install
exec /usr/local/rvm/bin/rvm-exec default bundle exec ./script/crunch-dispatch.rb 2>&1

