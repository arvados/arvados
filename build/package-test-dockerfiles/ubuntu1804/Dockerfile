# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:bionic
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates gnupg2

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.7 -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.7 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb [trusted=yes] file:///arvados/packages/ubuntu1804/ /" >>/etc/apt/sources.list

# Add preferences file for the Arvados packages. This pins Arvados
# packages at priority 501, so that older python dependency versions
# are preferred in those cases where we need them
ADD etc-apt-preferences.d-arvados /etc/apt/preferences.d/arvados
