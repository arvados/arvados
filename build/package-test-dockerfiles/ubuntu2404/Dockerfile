# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:noble
LABEL org.opencontainers.image.authors="Arvados Package Maintainers <packaging@arvados.org>"

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates gnupg2 ruby ruby-dev && \
    gem install --conservative --version '>= 2.4.0' bundler && \
    ln -s /usr/bin/bundler3.2 /usr/local/bin/bundle && \
    mkdir -p /etc/udev/disabled && \
    echo "deb [trusted=yes] file:///arvados/packages/ubuntu2404/ /" >>/etc/apt/sources.list
