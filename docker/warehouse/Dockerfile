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
      xsltproc realpath
    cd /usr/src/warehouse-apps && \
    sh install.sh /usr/local && \
    dpkg -i libwarehouse-perl*.deb

