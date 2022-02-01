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
    if [ ${CLEANUP_ARVADOS_DIR} -eq 1 ]; then
        rm -rf ${ARVADOS_DIR}
    fi
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
    echo "Usage: ${0} [options]"
    echo "Options:"
    echo "  -i            Run Cypress in interactive mode."
    echo "  -a PATH       Arvados dir. If PATH doesn't exist, a repo clone is downloaded there."
    echo "  -w PATH       Workbench2 dir. Default: Current working directory"
    exit 0
}

# Allow self-signed certs on 'wait-on'
export NODE_TLS_REJECT_UNAUTHORIZED=0

ARVADOS_DIR="unset"
CLEANUP_ARVADOS_DIR=0
CYPRESS_MODE="run"
WB2_DIR=`pwd`

while getopts "ia:w:" o; do
    case "${o}" in
        i)
            # Interactive mode
            CYPRESS_MODE="open"
            ;;
        a)
            ARVADOS_DIR=${OPTARG}
            ;;
        w)
            WB2_DIR=${OPTARG}
            ;;
        *)
            echo "Invalid Option: -$OPTARG" 1>&2
            usage
            ;;
    esac
done
shift $((OPTIND-1))

if [ "${ARVADOS_DIR}" = "unset" ]; then
  echo "ARVADOS_DIR is unset, creating a temporary directory for new checkout"
  ARVADOS_DIR=`mktemp -d`
fi

echo "ARVADOS_DIR is ${ARVADOS_DIR}"

ARVADOS_LOG=${ARVADOS_DIR}/arvados.log
ARVADOS_CONF=${WB2_DIR}/tools/arvados_config.yml
VOCABULARY_CONF=${WB2_DIR}/tools/example-vocabulary.json

if [ ! -f "${WB2_DIR}/src/index.tsx" ]; then
    echo "ERROR: '${WB2_DIR}' isn't workbench2's directory"
    usage
fi

if [ ! -f ${ARVADOS_CONF} ]; then
    echo "ERROR: Arvados config file ${ARVADOS_CONF} not found"
    exit 1
fi

if [ -f "${WB2_DIR}/public/config.json" ]; then
    echo "ERROR: Please move public/config.json file out of the way"
    exit 1
fi

if [ ! -d "${ARVADOS_DIR}/.git" ]; then
    mkdir -p ${ARVADOS_DIR} || exit 1
    CLEANUP_ARVADOS_DIR=1
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
TMPSUBDIR=$(mktemp -d -p /tmp | cut -d \/ -f3) # Removes the /tmp/ part for the regex below
TMPDIR=/tmp/${TMPSUBDIR}
cp ${VOCABULARY_CONF} ${TMPDIR}/voc.json
cp ${ARVADOS_CONF} ${TMPDIR}/arvados.yml
sed -i "s/VocabularyPath: \".*\"/VocabularyPath: \"\/tmp\/${TMPSUBDIR}\/voc.json\"/" ${TMPDIR}/arvados.yml
coproc arvboot (~/go/bin/arvados-server boot \
    -type test \
    -config ${TMPDIR}/arvados.yml \
    -no-workbench1 \
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
    yarn run cypress ${CYPRESS_MODE}
