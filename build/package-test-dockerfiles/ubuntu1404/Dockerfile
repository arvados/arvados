# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:trusty
MAINTAINER Ward Vandewege <wvandewege@veritasgenetics.com>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies
RUN apt-get update && \
    apt-get -y install --no-install-recommends curl ca-certificates python2.7-dev python3 python-setuptools python3-setuptools libcurl4-gnutls-dev curl git libattr1-dev libfuse-dev libpq-dev python-pip unzip binutils build-essential ca-certificates

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# udev daemon can't start in a container, so don't try.
RUN mkdir -p /etc/udev/disabled

RUN echo "deb file:///arvados/packages/ubuntu1404/ /" >>/etc/apt/sources.list
