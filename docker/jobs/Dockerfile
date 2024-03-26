# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Based on Debian
FROM debian:bullseye-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ARG repo_version
ARG cwl_runner_version

ADD apt.arvados.org-$repo_version.list /etc/apt/sources.list.d/
ADD 1078ECD7.key /etc/apt/trusted.gpg.d/arvados.asc
RUN apt-get update -q
RUN DEBIAN_FRONTEND=noninteractive apt-get install -yq --no-install-recommends python3-arvados-cwl-runner=$cwl_runner_version

# Install dependencies and set up system.
RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

USER crunch
ENV PATH=/usr/lib/python3-arvados-cwl-runner/bin:/usr/local/bin:/usr/bin:/bin
