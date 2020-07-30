#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Bring up a docker container with some locally-built commands (e.g.,
# cmd/arvados-server) replacing the ones that came with
# arvados-server-easy when the arvados-installpackage-* image was
# built.
#
# Assumes docker-build-install.sh has already succeeded.
#
# Example:
#
#    docker-boot.sh cmd/arvados-server services/keep-balance

set -e -o pipefail

declare -A opts=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        --shell)
            shift
            opts[shell]=1
            ;;
        *)
            break
            ;;
    esac
done

cleanup() {
    if [[ -n "${tmpdir}" ]]; then
        rm -rf "${tmpdir}"
    fi
}
trap cleanup ERR EXIT

tmpdir=$(mktemp -d)
version=$(git describe --tag --dirty)

declare -a volargs=()
for inject in "$@"; do
    case "$inject" in
        nginx.conf)
            volargs+=(-v "$(pwd)/sdk/python/tests/$inject:/var/lib/arvados/share/$inject:ro")
            ;;
        *)
            echo >&2 "building $inject..."
            (cd $inject && GOBIN=$tmpdir go install -ldflags "-X git.arvados.org/arvados.git/lib/cmd.version=${version} -X main.version=${version}")
            cmd="$(basename "$inject")"
            volargs+=(-v "$tmpdir/$cmd:/var/lib/arvados/bin/$cmd:ro")
            ;;
    esac
done

osbase=debian:10
installimage=arvados-installpackage-${osbase}
command="/var/lib/arvados/bin/arvados-server boot -listen-host 0.0.0.0"
if [[ "${opts[shell]}" ]]; then
    command="bash -login"
fi
docker run -it --rm \
       "${volargs[@]}" \
       "${installimage}" \
       bash -c "/etc/init.d/postgresql start && /var/lib/arvados/bin/arvados-server init -cluster-id x1234 && $command"
