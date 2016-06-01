FROM centos:7
MAINTAINER Brett Smith <brett@curoverse.com>

RUN yum -q -y install scl-utils centos-release-scl which tar

# Install RVM
RUN touch /var/lib/rpm/* && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundle && \
    /usr/local/rvm/bin/rvm-exec default gem install cure-fpm --version 1.6.0b

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
