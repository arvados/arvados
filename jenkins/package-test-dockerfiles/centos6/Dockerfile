FROM centos:6
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN yum -q install --assumeyes scl-utils centos-release-SCL && \
    yum -q install --assumeyes python27

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo