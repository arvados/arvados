#!/bin/bash

# Example of using `arvados-server boot` in a script. Bring up a test
# cluster, wait for it to come up, fetch something from its discovery
# doc, and shut down.

set -e -o pipefail

cleanup() {
    set -x
    kill ${boot_PID} ${consume_stdout_PID}
    wait ${boot_PID} ${consume_stdout_PID} || true
    echo >&2 "done"
}

coproc boot (arvados-server boot -type test -config doc/examples/config/zzzzz.yml -own-temporary-database -timeout 20m)
trap cleanup ERR EXIT

read controllerURL <&"${boot[0]}"

# Copy coproc's stdout to stderr, to ensure `arvados-server boot`
# doesn't get blocked trying to write stdout.
exec 7<&"${boot[0]}"; coproc consume_stdout (cat <&7 >&2)

keepwebURL=$(curl --silent --fail --insecure "${controllerURL}/discovery/v1/apis/arvados/v1/rest" | jq -r .keepWebServiceUrl)
echo >&2 "controller is at $controllerURL"
echo >&2 "keep-web is at $keepwebURL"
