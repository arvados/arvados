#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

cleanup_arvboot() {
    set -x
    kill ${arvboot_PID} ${consume_stdout_PID}
    wait ${arvboot_PID} ${consume_stdout_PID} || true
    echo >&2 "done"
}

random_free_port() {
    while port=$(shuf -n1 -i $(cat /proc/sys/net/ipv4/ip_local_port_range | tr '\011' '-'))
    netstat -atun | grep -q ":$port\s" ; do
        continue
    done
    echo $port
}

# Allow self-signed certs on 'wait-on'
export NODE_TLS_REJECT_UNAUTHORIZED=0

WORKDIR=`mktemp -d`
WORKDIR=/tmp/arvboot # For script testing purposes...
ARVADOS_LOG=${WORKDIR}/arvados.log
ARVADOS_CONF=`pwd`/tools/arvados_config.yml

if [ ! -e "${WORKDIR}/lib" ]; then
    echo "Downloading arvados..."
    git clone https://git.arvados.org/arvados.git ${WORKDIR} || exit 1
fi

echo "Building & installing arvados-server..."
cd ${WORKDIR}
go mod download || exit 1
cd cmd/arvados-server
go install
cd -

echo "Installing dev dependencies..."
~/go/bin/arvados-server install -type test || exit 1

echo "Running arvados in test mode..."
# ARVADOS_PORT=`random_free_port`
# go run ./cmd/arvados-server boot \
#     -config ${ARVADOS_CONF} \
#     -type test \
#     -own-temporary-database \
#     -controller-address :${ARVADOS_PORT} \
#     -listen-host localhost > ${ARVADOS_LOG} 2>&1 &
coproc arvboot (~/go/bin/arvados-server boot \
    -type test                      \
    -config ${ARVADOS_CONF}         \
    -own-temporary-database         \
    -timeout 20m)
trap cleanup_arvboot ERR EXIT

read controllerURL <&"${arvboot[0]}"

# Copy coproc's stdout to stderr, to ensure `arvados-server boot`
# doesn't get blocked trying to write stdout.
exec 7<&"${arvbboot[0]}"; coproc consume_stdout (cat <&7 >&2)

cd -
echo "Running workbench2..."
WB2_PORT=`random_free_port`
PORT=${WB2_PORT} REACT_APP_ARVADOS_API_HOST=${controllerURL} \
    yarn start &

# Wait for arvados & workbench2 to be up.
# Using https-get to avoid false positive 'ready' detection.
# yarn run wait-on --httpTimeout 300000 https-get://localhost:${ARVADOS_PORT}/discovery/v1/apis/arvados/v1/rest ||
yarn run wait-on --httpTimeout 300000 https-get://localhost:${WB2_PORT}

echo "Running tests..."
CYPRESS_system_token=systemusertesttoken1234567890aoeuidhtnsqjkxbmwvzpy \
    CYPRESS_controller_url=${controllerURL} \
    CYPRESS_BASE_URL=https://localhost:${WB2_PORT} \
    yarn run cypress run
TEST_EXIT_CODE=$?

# Cleanup
rm -rf ${WORKDIR}

exit ${TEST_EXIT_CODE}