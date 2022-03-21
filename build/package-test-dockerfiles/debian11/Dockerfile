# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:bullseye
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates gpg procps gpg-agent

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.7 -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.7 && \
    echo "gem: --no-document" >> /etc/gemrc && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/debian11/ /" >>/etc/apt/sources.list
