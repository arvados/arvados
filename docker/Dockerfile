# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM node:12.22.3-buster
LABEL maintainer="Arvados Package Maintainers <packaging@arvados.org>"

RUN echo deb http://deb.debian.org/debian buster-backports main >> /etc/apt/sources.list.d/backports.list
RUN apt-get update && \
    apt-get -yq --no-install-recommends -o Acquire::Retries=6 install \
    libsecret-1-0 libsecret-1-dev rpm ruby ruby-dev rubygems build-essential \
    libpam0g-dev libgbm1 git && \
    apt-get clean

# Get Go 1.16.9
RUN cd /usr/src && \
    wget https://golang.org/dl/go1.16.9.linux-amd64.tar.gz && \
    tar xzf go1.16.9.linux-amd64.tar.gz && \
    ln -s /usr/src/go/bin/go /usr/local/bin/go-1.16.9 && \
    ln -s /usr/src/go/bin/gofmt /usr/local/bin/gofmt-1.16.9 && \
    ln -s /usr/local/bin/go-1.16.9 /usr/local/bin/go && \
    ln -s /usr/local/bin/gofmt-1.16.9 /usr/local/bin/gofmt

RUN gem install --no-ri --no-rdoc fpm
RUN git clone https://git.arvados.org/arvados.git && cd arvados && \
    go mod download && \
    go run ./cmd/arvados-server install -type test && cd .. && \
    rm -rf arvados && \
    apt-get clean
