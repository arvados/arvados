# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# Build this with `build/build_docker_image.py`

FROM debian:bookworm-slim

RUN apt-get update -q \
 && DEBIAN_FRONTEND=noninteractive apt-get install -qy python3-venv

# The build script sets up our build context with all the Python source
# packages to install.
COPY . /usr/local/src/

RUN python3 -m venv /opt/arvados-py \
 && /opt/arvados-py/bin/pip install -qq --no-cache-dir --no-input /usr/local/src/*.whl

### Stage 2
FROM debian:bookworm-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

RUN apt-get update -q \
 && DEBIAN_FRONTEND=noninteractive apt-get install -qy python3 libcurl4 nodejs

# The symlink provides path compatibility with the old arvados/jobs image.
RUN adduser --disabled-password --gecos 'Crunch execution user' crunch \
 && install --directory --owner=crunch --group=crunch --mode=0700 \
    /keep /tmp/crunch-src /tmp/crunch-job \
 && ln -s /opt/arvados-py /usr/lib/python3-arvados-cwl-runner

USER crunch
COPY --from=0 /opt/arvados-py/ /opt/arvados-py/
ENV PATH=/opt/arvados-py/bin:/usr/local/bin:/usr/bin:/bin
