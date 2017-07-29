# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:xenial
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

# Install RVM
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/ubuntu1604/ /" >>/etc/apt/sources.list

# Add preferences file for the Arvados packages. This pins Arvados
# packages at priority 501, so that older python dependency versions
# are preferred in those cases where we need them
ADD etc-apt-preferences.d-arvados /etc/apt/preferences.d/arvados
