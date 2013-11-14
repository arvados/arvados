FROM arvados/base
MAINTAINER Tim Pierce <twp@clinicalfuture.com>

# Install Warehouse.
RUN apt-get -q -y install dpkg-dev debhelper libdbi-perl libwww-perl \
      libtest-pod-perl libtest-pod-coverage-perl libjson-perl && \
    git clone git://github.com/clinicalfuture/warehouse-apps.git /usr/src/warehouse-apps && \
    cd /usr/src/warehouse-apps && \
    sh install.sh /usr/local && \
    dpkg -i libwarehouse-perl.deb
