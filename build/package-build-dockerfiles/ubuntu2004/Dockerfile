# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ARG HOSTTYPE

FROM ubuntu:focal as build_x86_64
# Install go
ONBUILD ADD generated/go1.17.1.linux-amd64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v10.23.1-linux-x64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v10.23.1-linux-x64/bin/* /usr/local/bin/
# On x86, we want some cross-compilation support for arm64
# Add gcc-aarch64-linux-gnu to compile go binaries for arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y gcc-aarch64-linux-gnu
# We also need libpam compiled for arm64, and that requires some sources.list mangling
ONBUILD RUN /bin/sed -i 's/deb http/deb [ arch=amd64 ] http/' /etc/apt/sources.list
ONBUILD ADD ports.list /etc/apt/sources.list.d/
ONBUILD RUN /usr/bin/dpkg --add-architecture arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -o APT::Immediate-Configure=0 -q -y libpam0g-dev:arm64
# Note that cross compilation support for ubuntu2004 is disabled in
# run-library.sh because of
# https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=983477. Hopefully
# ubuntu2204 will have the fix introduced in debian11.

FROM ubuntu:focal as build_aarch64
# Install go
ONBUILD ADD generated/go1.17.1.linux-arm64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v10.23.1-linux-arm64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v10.23.1-linux-arm64/bin/* /usr/local/bin/

FROM build_${HOSTTYPE}

MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

SHELL ["/bin/bash", "-c"]
# Install dependencies.
RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y python3 python3-pip libcurl4-gnutls-dev libgnutls28-dev curl git libattr1-dev libfuse-dev libpq-dev unzip tzdata python3-venv python3-dev libpam-dev shared-mime-info equivs

# Install virtualenv
RUN /usr/bin/pip3 install 'virtualenv<20'

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.7 -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.7 && \
    echo "gem: --no-document" >> ~/.gemrc && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm --version 1.10.2

RUN /usr/local/rvm/bin/rvm-exec default bundle config --global jobs $(let a=$(grep -c processor /proc/cpuinfo )-1; echo $a)
# Cf. https://build.betterup.com/one-weird-trick-that-will-speed-up-your-bundle-install/
ENV MAKE "make --jobs $(grep -c processor /proc/cpuinfo)"

RUN git clone --depth 1 git://git.arvados.org/arvados.git /tmp/arvados && \
    cd /tmp/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle install && \
    cd /tmp/arvados/apps/workbench && \
    /usr/local/rvm/bin/rvm-exec default bundle install && \
    cd /tmp/arvados && \
    go mod download


ENV WORKSPACE /arvados
CMD ["/usr/local/rvm/bin/rvm-exec", "default", "bash", "/jenkins/run-build-packages.sh", "--target", "ubuntu2004"]
