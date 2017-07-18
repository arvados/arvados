# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:stretch
MAINTAINER Nico Cesar <nico@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

# Install RVM
COPY D39DC0E3.asc /tmp
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates gpg procps && \
    gpg --import /tmp/D39DC0E3.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/debian9/ /" >>/etc/apt/sources.list
