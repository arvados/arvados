#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
set -eux -o pipefail

. /usr/local/lib/arvbox/common.sh
. /usr/local/lib/arvbox/go-setup.sh

flock /var/lib/gopath/gopath.lock go get -t "git.curoverse.com/arvados.git/services/crunchstat"
flock /var/lib/gopath/gopath.lock go get -t "git.curoverse.com/arvados.git/sdk/go/crunchrunner"
install $GOPATH/bin/crunchstat $GOPATH/bin/crunchrunner /usr/local/bin

if test -s /var/lib/arvados/api_rails_env ; then
  RAILS_ENV=$(cat /var/lib/arvados/api_rails_env)
else
  RAILS_ENV=development
fi

export ARVADOS_API_HOST=$localip:${services[api]}
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=$(cat /usr/src/arvados/services/api/superuser_token)
export CRUNCH_JOB_BIN=/usr/src/arvados/sdk/cli/bin/crunch-job
export PERLLIB=/usr/src/arvados/sdk/perl/lib
export CRUNCH_TMP=/tmp/$1
export CRUNCH_DISPATCH_LOCKFILE=/var/lock/$1-dispatch
export CRUNCH_JOB_DOCKER_BIN=docker
export HOME=/tmp/$1
export CRUNCH_JOB_DOCKER_RUN_ARGS=--net=host
# Stop excessive stat of /etc/localtime
export TZ='America/New_York'

cd /usr/src/arvados/services/api
if test "$1" = "crunch0" ; then
    exec bundle exec ./script/crunch-dispatch.rb $RAILS_ENV --jobs --pipelines
else
    exec bundle exec ./script/crunch-dispatch.rb $RAILS_ENV --jobs
fi
