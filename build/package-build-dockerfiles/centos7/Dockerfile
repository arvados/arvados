FROM centos:7
MAINTAINER Brett Smith <brett@curoverse.com>

# Install build dependencies provided in base distribution
RUN yum -q -y install make automake gcc gcc-c++ libyaml-devel patch readline-devel zlib-devel libffi-devel openssl-devel bzip2 libtool bison sqlite-devel rpm-build git perl-ExtUtils-MakeMaker libattr-devel nss-devel libcurl-devel which tar unzip scl-utils centos-release-scl postgresql-devel python-devel python-setuptools fuse-devel xz-libs git

# Install golang binary
ADD generated/go1.6.2.linux-amd64.tar.gz /usr/local/
RUN ln -s /usr/local/go/bin/go /usr/local/bin/

# Install RVM
RUN gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundler && \
    /usr/local/rvm/bin/rvm-exec default gem install cure-fpm --version 1.6.0b

# Need to "touch" RPM database to workaround bug in interaction between
# overlayfs and yum (https://bugzilla.redhat.com/show_bug.cgi?id=1213602)
RUN touch /var/lib/rpm/* && yum -q -y install python33
RUN scl enable python33 "easy_install-3.3 pip" && easy_install-2.7 pip

ENV WORKSPACE /arvados
CMD ["scl", "enable", "python33", "/usr/local/rvm/bin/rvm-exec default bash /jenkins/run-build-packages.sh --target centos7"]
