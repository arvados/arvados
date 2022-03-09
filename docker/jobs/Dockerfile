# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Based on Debian
FROM debian:buster-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update -q
RUN apt-get install -yq --no-install-recommends gnupg

ARG repo_version
RUN echo repo_version $repo_version
ADD apt.arvados.org-$repo_version.list /etc/apt/sources.list.d/

ADD 1078ECD7.key /tmp/
RUN cat /tmp/1078ECD7.key | apt-key add -

ARG python_sdk_version
ARG cwl_runner_version
RUN echo cwl_runner_version $cwl_runner_version python_sdk_version $python_sdk_version

RUN apt-get update -q
RUN apt-get install -yq --no-install-recommends python3-arvados-cwl-runner=$cwl_runner_version

# use the Python executable from the python-arvados-cwl-runner package
RUN PYTHON=`ls /usr/share/python3*/dist/python3-arvados-cwl-runner/bin/python|head -n1` && rm -f /usr/bin/python && ln -s $PYTHON /usr/bin/python
RUN PYTHON3=`ls /usr/share/python3*/dist/python3-arvados-cwl-runner/bin/python3|head -n1` && rm -f /usr/bin/python3 && ln -s $PYTHON3 /usr/bin/python3

# Install dependencies and set up system.
RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

USER crunch
