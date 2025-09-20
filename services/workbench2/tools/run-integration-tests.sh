#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

cleanup() {
    set -x
    set +e +o pipefail
    kill ${arvboot_PID} ${consume_stdout_PID} ${wb2_PID} ${consume_wb2_stdout_PID}
    wait ${arvboot_PID} ${consume_stdout_PID} ${wb2_PID} ${consume_wb2_stdout_PID} || true
    echo >&2 "done"
}

random_free_port() {
    while port=$(shuf -n1 -i $(cat /proc/sys/net/ipv4/ip_local_port_range | tr '\011' '-'))
    netstat -atun | grep -q ":$port\s" ; do
        continue
    done
    echo $port
}

usage() {
    cat <<EOF
Usage: $0 [options]
Options:
  -i            Run Cypress in interactive mode.
Environment:
  ARVADOS_DIRECTORY  Path to an Arvados Git checkout
  WORKSPACE          Path to the Workbench source in that checkout
EOF
    exit 0
}

CYPRESS_MODE="run"
while getopts "i" o; do
    case "${o}" in
        i)
            # Interactive mode
            CYPRESS_MODE="open --e2e"
            ;;
        *)
            echo "Invalid Option: -$OPTARG" 1>&2
            usage
            ;;
    esac
done
shift $((OPTIND-1))

echo "ARVADOS_DIRECTORY is ${ARVADOS_DIRECTORY}"
cd "${WORKSPACE:=$ARVADOS_DIRECTORY/services/workbench2}"

if [ ! -f src/index.tsx ]; then
    echo "ERROR: '${WORKSPACE}' isn't workbench2's directory" >&2
    usage
fi

if [ -f public/config.json ]; then
    echo "ERROR: Please move public/config.json file out of the way" >&2
    exit 1
fi

echo "Launching arvados in test mode..."
TESTTMP="$ARVADOS_DIRECTORY/tmp/workbench2-integration"
mkdir -p "$TESTTMP"
ARVADOS_LOG="${TESTTMP}/arvados-workbench2-tests.log"
TEST_CONFIG="$TESTTMP/arvados_config.yml"
yq -y ".Clusters.zzzzz.API.VocabularyPath = \"$WORKSPACE/tools/example-vocabulary.json\"" \
   <"$WORKSPACE/tools/arvados_config.yml" >"$TEST_CONFIG"
coproc arvboot ("$(go env GOPATH)/bin/arvados-server" boot \
    -type test \
    -source "$ARVADOS_DIRECTORY" \
    -config "$TEST_CONFIG" \
    -no-workbench1 \
    -no-workbench2 \
    -own-temporary-database \
    -timeout 20m 2>"$ARVADOS_LOG")
trap cleanup ERR EXIT

read controllerURL _ <&"${arvboot[0]}"
echo "Arvados up and running at ${controllerURL}"

# Copy coproc's stdout to stderr, to ensure `arvados-server boot`
# doesn't get blocked trying to write stdout.
exec 7<&"${arvboot[0]}"; coproc consume_stdout (cat <&7 >&2)

echo "Launching workbench2..."
export NODE_TLS_REJECT_UNAUTHORIZED=0  # Allow self-signed certs on 'wait-on'
WB2_PORT=`random_free_port`
coproc wb2 (PORT=${WB2_PORT} \
    REACT_APP_ARVADOS_API_HOST="$(echo "$controllerURL" | cut -d/ -f3)" \
    yarn start)
exec 8<&"${wb2[0]}"; coproc consume_wb2_stdout (cat <&8 >&2)

# Wait for workbench2 to be up.
# Using https-get to avoid false positive 'ready' detection.
yarn run wait-on --timeout 300000 https-get://127.0.0.1:${WB2_PORT}

echo "Running tests..."
CYPRESS_system_token="$(yq -r .Clusters.zzzzz.SystemRootToken "$TEST_CONFIG")" \
    CYPRESS_controller_url=${controllerURL} \
    CYPRESS_BASE_URL=https://127.0.0.1:${WB2_PORT} \
    yarn run cypress ${CYPRESS_MODE} "$@"
