# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

## dont use debian:11 here since the word 'bullseye' is used for rvm precompiled binaries
FROM debian:bullseye
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

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
    /usr/local/rvm/bin/rvm install 2.5 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.5 && \
    echo "gem: --no-document" >> /etc/gemrc && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler --version 2.2.19 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm --version 1.10.2

# Install golang binary
ADD generated/go1.17.1.linux-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

# Install nodejs and npm
ADD generated/node-v10.23.1-linux-x64.tar.xz /usr/local/
RUN ln -s /usr/local/node-v10.23.1-linux-x64/bin/* /usr/local/bin/

RUN git clone --depth 1 git://git.arvados.org/arvados.git /tmp/arvados && cd /tmp/arvados/services/api && /usr/local/rvm/bin/rvm-exec default bundle && cd /tmp/arvados/apps/workbench && /usr/local/rvm/bin/rvm-exec default bundle

ENV WORKSPACE /arvados
CMD ["/usr/local/rvm/bin/rvm-exec", "default", "bash", "/jenkins/run-build-packages.sh", "--target", "debian11"]
