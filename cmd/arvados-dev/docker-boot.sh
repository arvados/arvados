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

cleanup() {
    if [[ -n "${tmpdir}" ]]; then
        rm -rf "${tmpdir}"
    fi
}
trap cleanup ERR EXIT

tmpdir=$(mktemp -d)
version=$(git describe --tag --dirty)

declare -a volargs=()
for srcdir in "$@"; do
    echo >&2 "building $srcdir..."
    (cd $srcdir && GOBIN=$tmpdir go install -ldflags "-X git.arvados.org/arvados.git/lib/cmd.version=${version} -X main.version=${version}")
    cmd="$(basename "$srcdir")"
    volargs+=(-v "$tmpdir/$cmd:/var/lib/arvados/bin/$cmd:ro")
done

osbase=debian:10
installimage=arvados-installpackage-${osbase}
docker run -it --rm \
       "${volargs[@]}" \
       "${installimage}" \
       bash -c '/etc/init.d/postgresql start && /var/lib/arvados/bin/arvados-server init -cluster-id x1234 && /var/lib/arvados/bin/arvados-server boot'
