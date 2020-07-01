#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

version="${PACKAGE_VERSION:-0.9.99}"

# mkdir -p /tmp/pkg
# (
#     cd cmd/arvados-dev
#     go install
# )
# docker run --rm \
#        -v /tmp/pkg:/pkg \
#        -v "${GOPATH:-${HOME}/go}"/bin/arvados-dev:/arvados-dev:ro \
#        -v "$(pwd)":/arvados:ro "${BUILDIMAGE:-debian:10}" \
#        /arvados-dev buildpackage \
#        -source /arvados \
#        -package-version "${version}" \
#        -output-directory /pkg
pkgfile=/tmp/pkg/arvados-server-easy_${version}_amd64.deb
# ls -l ${pkgfile}
# (
#     cd /tmp/pkg
#     dpkg-scanpackages . | gzip > Packages.gz
# )
sourcesfile=/tmp/sources.conf.d-arvados
echo >$sourcesfile "deb [trusted=yes] file:///pkg ./"
docker run -it --rm \
       -v /tmp/pkg:/pkg:ro \
       -v ${sourcesfile}:/etc/apt/sources.list.d/arvados-local.list:ro \
       ${INSTALLIMAGE:-debian:10} \
       bash -c 'apt update && DEBIAN_FRONTEND=noninteractive apt install -y arvados-server-easy && bash -login'
