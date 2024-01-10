# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Dockerfile for building an arvados/jobs Docker image from local git tree.
#
# Intended for use by developers working on arvados-python-client or
# arvados-cwl-runner and need to run a crunch job with a custom package
# version.
#
# Use arvados/build/build-dev-docker-jobs-image.sh to build.

FROM debian:bullseye-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

RUN DEBIAN_FRONTEND=noninteractive apt-get update -q && apt-get install -qy --no-install-recommends \
    git python3-dev python3-venv libcurl4-gnutls-dev libgnutls28-dev nodejs build-essential

RUN python3 -m venv /opt/arvados-py
ENV PATH=/opt/arvados-py/bin:/usr/local/bin:/usr/bin:/bin
RUN python3 -m pip install --no-cache-dir setuptools wheel

# The build script sets up our build context with all the Python source
# packages to install.
COPY . /usr/local/src/
# Run a-c-r afterward to check for a successful install.
RUN python3 -m pip install --no-cache-dir /usr/local/src/* && arvados-cwl-runner --version

RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

USER crunch
