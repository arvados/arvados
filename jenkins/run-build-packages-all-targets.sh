#!/bin/bash

read -rd "\000" helpmessage <<EOF
$(basename $0): Orchestrate run-build-packages.sh for every target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

set -e

FINAL_EXITCODE=0
JENKINS_DIR=$(dirname "$(readlink -e "$0")")

run_docker() {
    local tag=$1; shift
    if docker run -v "$JENKINS_DIR:/jenkins" -v "$WORKSPACE:/arvados" \
          --env ARVADOS_DEBUG=1 "arvados/build:$tag"; then
        # Success - nothing more to do.
        true
    else
        FINAL_EXITCODE=$?
        echo "ERROR: $tag build failed with exit status $FINAL_EXITCODE." >&2
    fi
}

# In case it's needed, build the containers. This costs just a few
# seconds when the containers already exist, so it's not a big deal to
# do it on each run.
cd "$JENKINS_DIR/dockerfiles"
time ./build-all-build-containers.sh

for dockerfile_path in $(find -name Dockerfile); do
    run_docker "$(basename $(dirname "$dockerfile_path"))"
done

exit $FINAL_EXITCODE
