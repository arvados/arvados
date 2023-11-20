# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:jammy
LABEL org.opencontainers.image.authors="Arvados Package Maintainers <packaging@arvados.org>"

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
    # Use custom 1.1 openssl -- see: https://github.com/rvm/rvm/issues/5209#issuecomment-1114159447
    /usr/local/rvm/bin/rvm pkg install openssl && \
    /usr/local/rvm/bin/rvm install 2.7 --with-openssl-dir=/usr/local/rvm/usr -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.7 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb [trusted=yes] file:///arvados/packages/ubuntu2204/ /" >>/etc/apt/sources.list
