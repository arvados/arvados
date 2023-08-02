# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM rockylinux:8.6-minimal
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

# Install dependencies.
RUN microdnf --assumeyes --enablerepo=devel install \
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
    procps-ng \
    python3 \
    readline-devel \
    ruby \
    shadow-utils \
    sqlite-devel \
    tar \
    wget \
    which \
    zlib-devel

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN touch /var/lib/rpm/* && \
    gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.7 -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.7 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.3.26

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
