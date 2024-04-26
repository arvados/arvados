# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ARG HOSTTYPE
ARG BRANCH
ARG GOVERSION

FROM rockylinux:8.8-minimal as build_x86_64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-amd64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v12.22.12-linux-x64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v12.22.12-linux-x64/bin/* /usr/local/bin/
ONBUILD RUN npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v12.22.12-linux-x64/bin/* /usr/local/bin/

FROM rockylinux:8.8-minimal as build_aarch64
ONBUILD ARG BRANCH
# Install go
ONBUILD ARG GOVERSION
ONBUILD ADD generated/go${GOVERSION}.linux-arm64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v12.22.12-linux-arm64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v12.22.12-linux-arm64/bin/* /usr/local/bin/
ONBUILD RUN npm install -g yarn
ONBUILD RUN ln -sf /usr/local/node-v12.22.12-linux-arm64/bin/* /usr/local/bin/

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
    libattr-devel \
    libcurl-devel \
    libffi-devel \
    libtool \
    libyaml-devel \
    make \
    nss-devel \
    openssl-devel \
    pam-devel \
    patch \
    pkgconf \
    postgresql-devel \
    procps-ng \
    python39 \
    python39-devel \
    readline-devel \
    rpm-build \
    ruby \
    ruby-devel \
    sqlite-devel \
    tar \
    unzip \
    wget \
    which \
    xz-libs \
    zlib-devel

# fpm depends on dotenv, but version 3.0 of that gem dropped support for
# Ruby 2.7, so we need to specifically install an older version.
RUN echo "gem: --no-document" >> ~/.gemrc && \
    gem install --conservative --version '>= 2.4.0' bundler && \
    gem install dotenv --version '~> 2.8' && \
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
CMD ["bash", "/jenkins/run-build-packages.sh", "--target", "rocky8"]
