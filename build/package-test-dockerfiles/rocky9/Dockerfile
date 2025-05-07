# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM rockylinux:9-minimal
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

# Install dependencies.
RUN sed -ri '/^\[devel\]$/,/^\[/ { s/^enabled=[01]$/enabled=1/ }' /etc/yum.repos.d/rocky-devel.repo \
 && microdnf --assumeyes module enable ruby:3.1 \
 && microdnf --assumeyes install \
    cpio \
    diffutils \
    findutils

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
