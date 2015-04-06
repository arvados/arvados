# Arvados cwl runner image

# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

ADD apt.arvados.org.list /etc/apt/sources.list.d/
RUN apt-key adv --keyserver pool.sks-keyservers.net --recv 1078ECD7

RUN apt-get update -qq
RUN apt-get install -qqy apt-utils \
  iptables dmsetup ca-certificates lxc apt-transport-https docker.io \
  procps git python-setuptools

ADD wrapdocker /usr/local/bin/wrapdocker.sh

ADD generated/sdk.tar.gz /usr/src/arvados/sdk

RUN cd /usr/src/arvados/sdk/python && python setup.py install

VOLUME /var/lib/docker

ENTRYPOINT ["/usr/local/bin/wrapdocker.sh"]
