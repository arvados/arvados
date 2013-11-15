FROM arvados/base
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install Warehouse.

ADD generated/warehouse.tar.gz /usr/src/

# Temporary: remove non-running test
RUN /bin/rm /usr/src/warehouse-apps/libwarehouse-perl/t/pod-coverage.t

RUN apt-get update && \
    apt-get -q -y install dpkg-dev debhelper libdbi-perl libwww-perl \
      libtest-pod-perl libtest-pod-coverage-perl libjson-perl flex \
      libgnupg-interface-perl libunix-syslog-perl libbsd-resource-perl \
      bioperl perlmagick imagemagick gnuplot libbz2-dev libfftw3-3 libfftw3-dev \
      xsltproc realpath supervisor libgpgme11-dev libcache-memcached-perl \
      libio-compress-perl mysql-server mysql-client-5.5 && \
    perl -MCPAN -e 'install MogileFS::Client;' \
                -e 'install Crypt::GpgME'

RUN \
    cd /usr/src/warehouse-apps && \
    sh install.sh /usr/local && \
    dpkg -i libwarehouse-perl*.deb && \
    /bin/mkdir -p /data/keep-0

ADD supervisor.conf /etc/supervisor/conf.d/keepd.conf
ADD generated/warehouse.conf /etc/warehouse/warehouse-client.conf
ADD generated/config_mysqld.sh /tmp/config_mysqld.sh
RUN /bin/sh /tmp/config_mysqld.sh && \
    /bin/rm /tmp/config_mysqld.sh

# Start the supervisor.
CMD ["/usr/bin/supervisord", "-n"]
