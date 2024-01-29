#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -ex -o pipefail

SRC=$(realpath $(dirname ${BASH_SOURCE[0]})/../..)

ctrname=arvadostest
ctrbase=${ctrname}
if [[ "${1}" != "--update" ]] || ! docker images --format={{.Repository}} | grep -x ${ctrbase}; then
    ctrbase=debian:10
fi

if docker ps -a --format={{.Names}} | grep -x ${ctrname}; then
    echo >&2 "container name already in use -- another builder running?"
    exit 1
fi

(cd ${SRC}/cmd/arvados-server && go install)
trap "docker rm --volumes ${ctrname}" ERR
docker run -it --name ${ctrname} \
       -v ${GOPATH:-${HOME}/go}/bin/arvados-server:/bin/arvados-server:ro \
       -v ${SRC}:/src/arvados:ro \
       -v /tmp \
       --env http_proxy \
       --env https_proxy \
       ${ctrbase} \
       bash -c "
set -ex -o pipefail
arvados-server install -type test
pg_ctlcluster 11 main start
cp -a /src/arvados /tmp/
cd /tmp/arvados
rm -rf tmp config.yml database.yml services/api/config/database.yml
mkdir tmp
build/run-tests.sh WORKSPACE=\$PWD --temp /tmp/arvados/tmp --only x"
docker commit ${ctrname} ${ctrname}
trap - ERR
docker rm --volumes ${ctrname}
