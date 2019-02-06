# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM centos:7
MAINTAINER Ward Vandewege <wvandewege@veritasgenetics.com>

# Install dependencies.
RUN yum -q -y install scl-utils centos-release-scl which tar wget

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN touch /var/lib/rpm/* && \
    gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.3 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.3

# Add epel, we need it for the python-pam dependency
RUN wget http://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
RUN rpm -ivh epel-release-latest-7.noarch.rpm

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
