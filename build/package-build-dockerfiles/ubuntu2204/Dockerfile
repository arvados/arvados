# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ARG HOSTTYPE
ARG BRANCH
ARG GOVERSION

FROM ubuntu:jammy as build_x86_64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-amd64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v12.22.12-linux-x64.tar.xz /usr/local/
ONBUILD RUN env -C /usr/local/node-v12.22.12-linux-x64/bin PATH="$PATH:." ./npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v12.22.12-linux-x64/bin/* /usr/local/bin/
# On x86, we want some cross-compilation support for arm64
# Add gcc-aarch64-linux-gnu to compile go binaries for arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y gcc-aarch64-linux-gnu
# We also need libpam compiled for arm64, and that requires some sources.list mangling
ONBUILD RUN /bin/sed -i 's/deb http/deb [ arch=amd64 ] http/' /etc/apt/sources.list
ONBUILD ADD ports.list /etc/apt/sources.list.d/
ONBUILD RUN /usr/bin/dpkg --add-architecture arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -o APT::Immediate-Configure=0 -q -y libpam0g-dev:arm64 libfuse-dev:arm64

FROM ubuntu:jammy as build_aarch64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-arm64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v12.22.12-linux-arm64.tar.xz /usr/local/
ONBUILD RUN env -C /usr/local/node-v12.22.12-linux-arm64/bin PATH="$PATH:." ./npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v12.22.12-linux-arm64/bin/* /usr/local/bin/

FROM build_${HOSTTYPE}

LABEL org.opencontainers.image.authors="Arvados Package Maintainers <packaging@arvados.org>"

ENV DEBIAN_FRONTEND noninteractive

SHELL ["/bin/bash", "-c"]
# Install dependencies.
RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y \
    build-essential \
    curl \
    equivs \
    git \
    libattr1-dev \
    libcurl4-gnutls-dev \
    libfuse-dev \
    libgnutls28-dev \
    libpam-dev \
    libpq-dev \
    pkgconf \
    python3 \
    python3-dev \
    python3-venv \
    ruby \
    ruby-dev \
    shared-mime-info \
    tzdata \
    unzip

RUN echo "gem: --no-document" >> ~/.gemrc && \
    gem install --conservative --version '>= 2.4.0' bundler && \
    gem install fpm --version 1.15.1 && \
    bundle config --global jobs $(let a=$(grep -c processor /proc/cpuinfo )-1; echo $a)

# Cf. https://build.betterup.com/one-weird-trick-that-will-speed-up-your-bundle-install/
ENV MAKE "make --jobs 8"

# Preseed the go module cache.
RUN git clone git://git.arvados.org/arvados.git /tmp/arvados && \
    cd /tmp/arvados && \
    if [[ -n "${BRANCH}" ]]; then git checkout ${BRANCH}; fi && \
    go mod download

ENV WORKSPACE /arvados
CMD ["bash", "/jenkins/run-build-packages.sh", "--target", "ubuntu2204"]
