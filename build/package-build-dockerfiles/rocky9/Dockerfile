# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ARG HOSTTYPE
ARG BRANCH
ARG GOVERSION

FROM rockylinux:9-minimal as build_x86_64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-amd64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v14.21.3-linux-x64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v14.21.3-linux-x64/bin/* /usr/local/bin/
ONBUILD RUN npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v14.21.3-linux-x64/bin/* /usr/local/bin/

FROM rockylinux:9-minimal as build_aarch64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-arm64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v14.21.3-linux-arm64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v14.21.3-linux-arm64/bin/* /usr/local/bin/
ONBUILD RUN npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v14.21.3-linux-arm64/bin/* /usr/local/bin/

FROM build_${HOSTTYPE}

MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

# Install dependencies.
RUN microdnf --assumeyes module enable ruby:3.1 \
 && microdnf --assumeyes --enablerepo=devel install \
    automake \
    bison \
    bzip2 \
    fuse-devel \
    gcc \
    gcc-c++ \
    git \
    libcurl-devel \
    libyaml-devel \
    make \
    nss-devel \
    openssl-devel \
    pam-devel \
    procps-ng \
    python3.11 \
    python3.11-devel \
    rpm-build \
    ruby \
    ruby-devel \
    tar \
    unzip \
    which \
    zlib-devel

RUN echo "gem: --no-document" >> ~/.gemrc && \
    gem install --conservative --version '>= 2.4.0' bundler && \
    gem install fpm --version 1.15.1 && \
    bundle config --global jobs $(let a=$(grep -c processor /proc/cpuinfo )-1; echo $a)

# Cf. https://build.betterup.com/one-weird-trick-that-will-speed-up-your-bundle-install/
ENV MAKE "make --jobs $(grep -c processor /proc/cpuinfo)"

# Preseed the go module cache.
RUN git clone git://git.arvados.org/arvados.git /tmp/arvados && \
    cd /tmp/arvados && \
    if [[ -n "${BRANCH}" ]]; then git checkout ${BRANCH}; fi && \
    go mod download

ENV WORKSPACE /arvados
CMD ["bash", "/jenkins/run-build-packages.sh", "--target", "rocky9"]
