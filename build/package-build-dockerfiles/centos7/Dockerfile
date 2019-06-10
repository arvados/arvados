# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM centos:7
MAINTAINER Ward Vandewege <ward@curoverse.com>

# Install dependencies.
RUN yum -q -y install make automake gcc gcc-c++ libyaml-devel patch readline-devel zlib-devel libffi-devel openssl-devel bzip2 libtool bison sqlite-devel rpm-build git perl-ExtUtils-MakeMaker libattr-devel nss-devel libcurl-devel which tar unzip scl-utils centos-release-scl postgresql-devel python-devel python-setuptools fuse-devel xz-libs git python-virtualenv wget

# Install RVM
ADD generated/mpapis.asc /tmp/
ADD generated/pkuczynski.asc /tmp/
RUN gpg --import --no-tty /tmp/mpapis.asc && \
    gpg --import --no-tty /tmp/pkuczynski.asc && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.5 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.5 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm --version 1.10.2

# Install golang binary
ADD generated/go1.10.1.linux-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

# Install nodejs and npm
ADD generated/node-v6.11.2-linux-x64.tar.xz /usr/local/
RUN ln -s /usr/local/node-v6.11.2-linux-x64/bin/* /usr/local/bin/

# Need to "touch" RPM database to workaround bug in interaction between
# overlayfs and yum (https://bugzilla.redhat.com/show_bug.cgi?id=1213602)
RUN touch /var/lib/rpm/* && yum -q -y install rh-python35
RUN scl enable rh-python35 "easy_install-3.5 pip" && easy_install-2.7 pip

# Add epel, we need it for the python-pam dependency
RUN wget http://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm
RUN rpm -ivh epel-release-latest-7.noarch.rpm

RUN git clone --depth 1 git://git.curoverse.com/arvados.git /tmp/arvados && cd /tmp/arvados/services/api && /usr/local/rvm/bin/rvm-exec default bundle && cd /tmp/arvados/apps/workbench && /usr/local/rvm/bin/rvm-exec default bundle && rm -rf /tmp/arvados

# The version of setuptools that comes with CentOS is way too old
RUN pip install --upgrade setuptools

ENV WORKSPACE /arvados
CMD ["scl", "enable", "rh-python35", "/usr/local/rvm/bin/rvm-exec default bash /jenkins/run-build-packages.sh --target centos7"]
