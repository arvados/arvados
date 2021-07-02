# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM node:12.22.1-buster
LABEL maintainer="Arvados Package Maintainers <packaging@arvados.org>"

RUN echo deb http://deb.debian.org/debian buster-backports main >> /etc/apt/sources.list.d/backports.list
RUN apt-get update && \
    apt-get -yq --no-install-recommends -o Acquire::Retries=6 install \
    libsecret-1-0 libsecret-1-dev rpm ruby ruby-dev rubygems build-essential \
    libpam0g-dev libgbm1 && \
    apt-get clean
RUN apt-get -yq --no-install-recommends -t buster-backports install golang-go && \
    apt-get clean
RUN gem install --no-ri --no-rdoc fpm
RUN git clone https://git.arvados.org/arvados.git && cd arvados && \
    go mod download && \
    go run ./cmd/arvados-server install -type test && cd .. && \
    rm -rf arvados && \
    apt-get clean
