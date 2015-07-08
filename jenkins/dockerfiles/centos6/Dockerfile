FROM centos:6
MAINTAINER Brett Smith <brett@curoverse.com>

# Install build dependencies provided in base distribution
RUN yum -q -y install make automake gcc gcc-c++ libyaml-devel patch readline-devel zlib-devel libffi-devel openssl-devel bzip2 libtool bison sqlite-devel rpm-build git perl-ExtUtils-MakeMaker fuse-devel libattr-devel nss-devel libcurl-devel which tar scl-utils

# Install golang binary
ADD generated/golang-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

# Install RVM
RUN gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install fpm

# Download and install Software Collections for newer requirements
ADD https://www.softwarecollections.org/en/scls/rhscl/rh-python34/epel-6-x86_64/download/rhscl-rh-python34-epel-6-x86_64.noarch.rpm https://www.softwarecollections.org/en/scls/rhscl/python27/epel-6-x86_64/download/rhscl-python27-epel-6-x86_64.noarch.rpm /opt/

RUN yum -q -y install /opt/rhscl-*.rpm
RUN yum -q -y install python27 rh-python34

ENV WORKSPACE /arvados
CMD ["scl", "enable", "rh-python34", "python27", "/usr/local/rvm/bin/rvm-exec default bash /jenkins/run-build-packages.sh --target centos6"]
