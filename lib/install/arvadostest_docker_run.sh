#!/bin/bash

# Example:
#
# ./arvadostest_docker_build.sh             # build the base image ("arvadostest")
# ./arvadostest_docker_build.sh --update    # update the base image with current version of `arvados-server install`
# ./arvadostest_docker_run.sh --interactive # start a container using the previously built base image, copy this source tree into it, and invoke run-tests.sh with the given args

set -ex -o pipefail

declare -a qargs
for arg in "$@"; do
    qargs+=("${arg@Q}")
done

SRC=$(realpath $(dirname ${BASH_SOURCE[0]})/../..)

docker run --rm -it \
       --privileged \
       -v /dev/fuse:/dev/fuse \
       -v ${SRC}:/src/arvados:ro \
       -v /tmp \
       --env http_proxy \
       --env https_proxy \
       arvadostest \
       bash -c "
set -ex -o pipefail
pg_ctlcluster 11 main start
cp -a /src/arvados /tmp/
cd /tmp/arvados
rm -rf tmp config.yml database.yml services/api/config/database.yml
mkdir tmp
go run ./cmd/arvados-server install -type test
build/run-tests.sh WORKSPACE=\$PWD --temp /tmp/arvados/tmp ${qargs[@]}"
