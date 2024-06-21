# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:bullseye
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates gpg procps gpg-agent ruby ruby-dev && \
    echo "gem: --no-document" >> /etc/gemrc && \
    gem install --conservative --version '~> 2.4.0' bundler && \
    mkdir -p /etc/udev/disabled && \
    echo "deb file:///arvados/packages/debian11/ /" >>/etc/apt/sources.list
