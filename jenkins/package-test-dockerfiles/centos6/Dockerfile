FROM centos:6
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN yum -q install --assumeyes scl-utils centos-release-SCL \
    which tar

# Install RVM
RUN touch /var/lib/rpm/* && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundle fpm

RUN cd / root && \
    curl -OL 'http://pkgs.repoforge.org/rpmforge-release/rpmforge-release-0.5.3-1.el6.rf.x86_64.rpm' && \
    rpm -ivh rpmforge-release-0.5.3-1.el6.rf.x86_64.rpm

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo