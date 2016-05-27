FROM centos:6
MAINTAINER Peter Amstutz <peter.amstutz@curoverse.com>

RUN yum -q install --assumeyes scl-utils centos-release-scl \
    which tar

# Install RVM
RUN touch /var/lib/rpm/* && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    curl -L https://get.rvm.io | bash -s stable && \
    /usr/local/rvm/bin/rvm install 2.1 && \
    /usr/local/rvm/bin/rvm alias create default ruby-2.1 && \
    /usr/local/rvm/bin/rvm-exec default gem install bundle && \
    /usr/local/rvm/bin/rvm-exec default gem install cure-fpm --version 1.6.0b

RUN cd /tmp && \
    curl -OL 'http://pkgs.repoforge.org/rpmforge-release/rpmforge-release-0.5.3-1.el6.rf.x86_64.rpm' && \
    rpm -ivh rpmforge-release-0.5.3-1.el6.rf.x86_64.rpm && \
    sed -i 's/enabled = 0/enabled = 1/' /etc/yum.repos.d/rpmforge.repo

COPY localrepo.repo /etc/yum.repos.d/localrepo.repo
