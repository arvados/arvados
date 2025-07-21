# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# Build this with `build/build_docker_image.py`

FROM debian:bookworm-slim

RUN apt-get update -q \
 && DEBIAN_FRONTEND=noninteractive apt-get install -qy python3-venv \
 && python3 -m venv /opt/arvados-py

# The build script sets up our build context with all the Python source
# packages to install.
COPY . /usr/local/src/

RUN /opt/arvados-py/bin/pip install -qq --no-cache-dir --no-input \
    -r /usr/local/src/requirements.txt

### Stage 2
FROM debian:bookworm-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>
ARG APT_PKGLIST
ARG OLD_PKGNAME=python3-arvados-python-client

RUN apt-get update -q \
 && DEBIAN_FRONTEND=noninteractive apt-get install -qy python3 $APT_PKGLIST

# The symlinks provide path compatibility with old package-based images.
RUN adduser --disabled-password --gecos 'Crunch execution user' crunch \
 && install --directory --owner=crunch --group=crunch --mode=0700 \
    /keep /tmp/crunch-src /tmp/crunch-job \
 && ln -s /opt/arvados-py "/usr/lib/$OLD_PKGNAME"

USER crunch
ENV PATH=/opt/arvados-py/bin:/usr/local/bin:/usr/bin:/bin

COPY --from=0 /opt/arvados-py/ /opt/arvados-py/
