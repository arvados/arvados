# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Based on Debian Jessie
FROM debian:jessie
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

ADD apt.arvados.org.list /etc/apt/sources.list.d/
RUN apt-key adv --keyserver pool.sks-keyservers.net --recv 1078ECD7
RUN gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3

ARG python_sdk_version
ARG cwl_runner_version
RUN echo cwl_runner_version $cwl_runner_version python_sdk_version $python_sdk_version

RUN apt-get update -q
RUN apt-get install -yq --no-install-recommends \
    git python-pip python-virtualenv \
    python-dev libgnutls28-dev libcurl4-gnutls-dev nodejs \
    python-arvados-python-client=$python_sdk_version \
    python-arvados-cwl-runner=$cwl_runner_version

# Install dependencies and set up system.
RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

USER crunch
