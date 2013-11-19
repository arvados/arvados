# Arvados passenger image

FROM arvados/base
MAINTAINER Ward Vandewege <ward@clinicalfuture.com>

# Install packages and build the passenger apache module
RUN apt-get update ;\
    apt-get install -q -y apt-utils git curl procps apache2-mpm-worker \
                          libcurl4-openssl-dev apache2-threaded-dev libapr1-dev libaprutil1-dev ;\
    passenger-install-apache2-module --auto

