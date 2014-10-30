# Arvados passenger image

FROM arvados/base
MAINTAINER Ward Vandewege <ward@curoverse.com>

# Install packages and build the passenger apache module

RUN apt-get update -qq
RUN apt-get install -qqy \
        apt-utils git curl procps apache2-mpm-worker \
        libcurl4-openssl-dev apache2-threaded-dev \
        libapr1-dev libaprutil1-dev

RUN cd /usr/src/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle exec passenger-install-apache2-module --auto

RUN cd /usr/src/arvados/services/api && \
    /usr/local/rvm/bin/rvm-exec default bundle exec passenger-install-apache2-module --snippet > /etc/apache2/conf.d/passenger

