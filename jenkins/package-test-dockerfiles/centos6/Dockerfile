FROM centos:6
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN yum -q install --assumeyes scl-utils && \
    curl -L -O https://www.softwarecollections.org/en/scls/rhscl/python27/epel-6-x86_64/download/rhscl-python27-epel-6-x86_64.noarch.rpm && \
    yum -q install --assumeyes rhscl-python27-epel-6-x86_64.noarch.rpm && \
    yum -q install --assumeyes python27

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo