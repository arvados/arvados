# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ARG HOSTTYPE

## dont use debian:11 here since the word 'bullseye' is used for rvm precompiled binaries
FROM debian:bullseye as build_x86_64
# Install go
ONBUILD ADD generated/go1.17.1.linux-amd64.tar.gz /usr/local/
ONBUILD RUN ln -s /usr/local/go/bin/go /usr/local/bin/
# Install nodejs and npm
ONBUILD ADD generated/node-v10.23.1-linux-x64.tar.xz /usr/local/
ONBUILD RUN ln -s /usr/local/node-v10.23.1-linux-x64/bin/* /usr/local/bin/
# On x86, we want some cross-compilation support for arm64
# Add gcc-aarch64-linux-gnu to compile go binaries for arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y gcc-aarch64-linux-gnu
# We also need libpam compiled for arm64
ONBUILD RUN /usr/bin/dpkg --add-architecture arm64
ONBUILD RUN /usr/bin/apt-get update && /usr/bin/apt-get install -o APT::Immediate-Configure=0 -q -y libpam0g-dev:arm64

FROM debian:bullseye as build_aarch64
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
RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y python3 python3-setuptools python3-pip libcurl4-gnutls-dev curl git procps libattr1-dev libfuse-dev libgnutls28-dev libpq-dev unzip python3-venv python3-dev libpam-dev equivs

# Install virtualenv
RUN /usr/bin/pip3 install 'virtualenv<20'

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.5 -j $(grep -c processor /proc/cpuinfo) && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.5 && \
    echo "gem: --no-document" >> ~/.gemrc && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm --version 1.10.2

RUN /usr/local/rvm/bin/rvm-exec default bundle config --global jobs $(let a=$(grep -c processor /proc/cpuinfo )-1; echo $a)
# Cf. https://build.betterup.com/one-weird-trick-that-will-speed-up-your-bundle-install/
ENV MAKE "make --jobs $(grep -c processor /proc/cpuinfo)"

RUN echo $MAKE && git clone -b 17417-switch-to-mini_racer --depth 1 git://git.arvados.org/arvados.git /tmp/arvados && \
    cd /tmp/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle install && \
    cd /tmp/arvados/apps/workbench && \
    /usr/local/rvm/bin/rvm-exec default bundle install

ENV WORKSPACE /arvados
CMD ["/usr/local/rvm/bin/rvm-exec", "default", "bash", "/jenkins/run-build-packages.sh", "--target", "debian11"]
