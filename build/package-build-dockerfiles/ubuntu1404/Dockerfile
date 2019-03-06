# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:trusty
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

# Install dependencies.
RUN /usr/bin/apt-get update && /usr/bin/apt-get install -q -y python2.7-dev python3 python-setuptools python3-setuptools python3-pip libcurl4-gnutls-dev curl git libattr1-dev libfuse-dev libpq-dev python-pip unzip python3.4-venv python3.4-dev

# Install virtualenv
RUN /usr/bin/pip install virtualenv

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm --version 1.10.2

# Install golang binary
ADD generated/go1.10.1.linux-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

# Install nodejs and npm
ADD generated/node-v6.11.2-linux-x64.tar.xz /usr/local/
RUN ln -s /usr/local/node-v6.11.2-linux-x64/bin/* /usr/local/bin/

RUN git clone --depth 1 git://git.curoverse.com/arvados.git /tmp/arvados && cd /tmp/arvados/services/api && /usr/local/rvm/bin/rvm-exec default bundle && cd /tmp/arvados/apps/workbench && /usr/local/rvm/bin/rvm-exec default bundle && rm -rf /tmp/arvados

ENV WORKSPACE /arvados
CMD ["/usr/local/rvm/bin/rvm-exec", "default", "bash", "/jenkins/run-build-packages.sh", "--target", "ubuntu1404"]
