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

RUN /usr/bin/gem install --no-document fpm
RUN cd /usr/src && git clone git://git.arvados.org/arvados.git && \
    cd arvados && \
    GO_VERSION=$(grep 'goversion =' lib/install/deps.go |awk -F'"' '{print $2}') && \
    echo $GO_VERSION && \
    cd /usr/src && \
    wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    ln -s /usr/src/go/bin/go /usr/local/bin/go-${GO_VERSION} && \
    ln -s /usr/src/go/bin/gofmt /usr/local/bin/gofmt-${GO_VERSION} && \
    ln -s /usr/local/bin/go-${GO_VERSION} /usr/local/bin/go && \
    ln -s /usr/local/bin/gofmt-${GO_VERSION} /usr/local/bin/gofmt

# preseed arvados build dependencies
RUN cd /usr/src/arvados && \
    apt-get update && \
    go mod download && \
    go run ./cmd/arvados-server install -type test && cd .. && \
    rm -rf arvados && \
    apt-get clean

RUN git config --global --add safe.directory /usr/src/arvados && \
    git config --global --add safe.directory /usr/src/workbench2