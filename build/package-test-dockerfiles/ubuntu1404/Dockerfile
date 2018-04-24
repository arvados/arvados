# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:trusty
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies and RVM
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates python2.7-dev python3 python-setuptools python3-setuptools libcurl4-gnutls-dev curl git libattr1-dev libfuse-dev libpq-dev python-pip unzip binutils build-essential ca-certificates  && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/ubuntu1404/ /" >>/etc/apt/sources.list
