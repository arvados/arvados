#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

cleanup() {
    set -x
    kill ${arvboot_PID} ${consume_stdout_PID} ${wb2_PID} ${consume_wb2_stdout_PID}
    wait ${arvboot_PID} ${consume_stdout_PID} ${wb2_PID} ${consume_wb2_stdout_PID} || true
    rm -rf ${ARVADOS_DIR}
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

ARVADOS_DIR=`mktemp -d`
ARVADOS_LOG=${ARVADOS_DIR}/arvados.log
WB2_DIR=`pwd`
ARVADOS_CONF=${WB2_DIR}/tools/arvados_config.yml

if [ -f "${WB2_DIR}/public/config.json" ]; then
    echo "ERROR: Cannot run with Workbench2's public/config.json file"
    exit 1
fi

if [ ! -d "${ARVADOS_DIR}/lib" ]; then
    echo "Downloading arvados..."
    git clone https://git.arvados.org/arvados.git ${ARVADOS_DIR} || exit 1
fi

echo "Building & installing arvados-server..."
cd ${ARVADOS_DIR}
go mod download || exit 1
cd cmd/arvados-server
go install
cd -

echo "Installing dev dependencies..."
~/go/bin/arvados-server install -type test || exit 1

echo "Launching arvados in test mode..."
coproc arvboot (~/go/bin/arvados-server boot \
    -type test \
    -config ${ARVADOS_CONF} \
    -own-temporary-database \
    -timeout 20m 2> ${ARVADOS_LOG})
trap cleanup ERR EXIT

read controllerURL <&"${arvboot[0]}" || exit 1
echo "Arvados up and running at ${controllerURL}"
IFS='/' ; read -ra controllerHostPort <<< "${controllerURL}" ; unset IFS
controllerHostPort=${controllerHostPort[2]}

# Copy coproc's stdout to stderr, to ensure `arvados-server boot`
# doesn't get blocked trying to write stdout.
exec 7<&"${arvboot[0]}"; coproc consume_stdout (cat <&7 >&2)

cd ${WB2_DIR}
echo "Launching workbench2..."
WB2_PORT=`random_free_port`
coproc wb2 (PORT=${WB2_PORT} \
    REACT_APP_ARVADOS_API_HOST=${controllerHostPort} \
    yarn start)
exec 8<&"${wb2[0]}"; coproc consume_wb2_stdout (cat <&8 >&2)

# Wait for workbench2 to be up.
# Using https-get to avoid false positive 'ready' detection.
yarn run wait-on --timeout 300000 https-get://localhost:${WB2_PORT} || exit 1

echo "Running tests..."
CYPRESS_system_token=systemusertesttoken1234567890aoeuidhtnsqjkxbmwvzpy \
    CYPRESS_controller_url=${controllerURL} \
    CYPRESS_BASE_URL=https://localhost:${WB2_PORT} \
    yarn run cypress run
