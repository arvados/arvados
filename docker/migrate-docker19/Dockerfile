# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

FROM debian:8

ENV DEBIAN_FRONTEND noninteractive

RUN apt-key adv --keyserver pool.sks-keyservers.net --recv 1078ECD7 && \
    gpg --keyserver pool.sks-keyservers.net --recv-keys D39DC0E3 && \
    apt-key adv --keyserver hkp://pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D || \
    apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D

VOLUME /var/lib/docker

RUN mkdir -p /etc/apt/sources.list.d && \
    echo deb http://apt.arvados.org/jessie jessie main > /etc/apt/sources.list.d/apt.arvados.org.list && \
    apt-get clean && \
    apt-get update && \
    apt-get install -yq --no-install-recommends -o Acquire::Retries=6 \
        git curl python-arvados-python-client apt-transport-https ca-certificates && \
    apt-get clean

RUN echo deb https://apt.dockerproject.org/repo debian-jessie main > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -yq --no-install-recommends -o Acquire::Retries=6 \
        docker-engine=1.9.1-0~jessie && \
    apt-get clean

RUN mkdir /root/pkgs && \
    cd /root/pkgs && \
    curl -L -O https://apt.dockerproject.org/repo/pool/main/d/docker-engine/docker-engine_1.13.1-0~debian-jessie_amd64.deb && \
    curl -L -O http://httpredir.debian.org/debian/pool/main/libt/libtool/libltdl7_2.4.2-1.11+b1_amd64.deb

ADD migrate.sh dnd.sh /root/
