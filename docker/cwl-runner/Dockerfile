# Arvados cwl runner image

# Based on Debian Wheezy
FROM arvados/debian:wheezy
MAINTAINER Ward Vandewege <ward@curoverse.com>

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -qqy git python-setuptools

ADD generated/sdk.tar.gz /usr/src/arvados/sdk

RUN cd /usr/src/arvados/sdk/python && python setup.py install
