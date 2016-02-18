#!/bin/bash

exec 2>&1
set -eux -o pipefail

. /usr/local/lib/arvbox/common.sh

mkdir -p /var/lib/arvados/gostuff
cd /var/lib/arvados/gostuff

export GOPATH=$PWD
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "/usr/src/arvados" "$GOPATH/src/git.curoverse.com/arvados.git"
flock /var/lib/arvados/gostuff.lock go get -t "git.curoverse.com/arvados.git/services/crunchstat"
install bin/crunchstat /usr/local/bin

export ARVADOS_API_HOST=$localip:${services[api]}
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=$(cat /usr/src/arvados/services/api/superuser_token)
export CRUNCH_JOB_BIN=/usr/src/arvados/sdk/cli/bin/crunch-job
export PERLLIB=/usr/src/arvados/sdk/perl/lib
export CRUNCH_TMP=/tmp/$1
export CRUNCH_DISPATCH_LOCKFILE=/var/lock/$1-dispatch
export CRUNCH_JOB_DOCKER_BIN=docker
export HOME=/tmp/$1

cd /usr/src/arvados/services/api
exec bundle exec ./script/crunch-dispatch.rb development
