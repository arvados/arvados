# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM rockylinux:8.8-minimal
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

# Install dependencies.
RUN microdnf --assumeyes module enable ruby:3.1 \
 && microdnf --assumeyes --enablerepo=devel install \
    autoconf \
    automake \
    bison \
    bzip2 \
    cpio \
    diffutils \
    findutils \
    gcc-c++ \
    glibc-devel \
    glibc-headers \
    gzip \
    libffi-devel \
    libtool \
    make \
    openssl-devel \
    patch \
    pkgconf \
    procps-ng \
    python3 \
    readline-devel \
    redhat-rpm-config \
    ruby \
    ruby-devel \
    shadow-utils \
    sqlite-devel \
    tar \
    wget \
    which \
    zlib-devel

RUN touch /var/lib/rpm/* && \
    echo "gem: --no-document" >> ~/.gemrc && \
    gem install bundler --version 2.2.19

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
