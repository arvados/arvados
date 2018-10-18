# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM centos:7
MAINTAINER Ward Vandewege <wvandewege@veritasgenetics.com>

# Install dependencies.
RUN yum -q -y install scl-utils centos-release-scl which tar

# Install RVM
ADD generated/rvm.asc /tmp/
RUN touch /var/lib/rpm/* && \
    gpg --import /tmp/rvm.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundle && \
    /usr/local/rvm/bin/rvm-exec default gem install cure-fpm --version 1.6.0b

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
