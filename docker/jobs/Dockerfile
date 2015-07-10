# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

ADD apt.arvados.org.list /etc/apt/sources.list.d/
RUN apt-key adv --keyserver pool.sks-keyservers.net --recv 1078ECD7
RUN apt-get update -q

RUN apt-get install -qy git python-minimal python-arvados-python-client

RUN gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3

# Install dependencies and set up system.
# The FUSE packages help ensure that we can install the Python SDK (arv-mount).
RUN /usr/sbin/adduser --disabled-password \
      --gecos 'Crunch execution user' crunch && \
    /usr/bin/install --directory --owner=crunch --group=crunch --mode=0700 /keep /tmp/crunch-src /tmp/crunch-job

USER crunch
