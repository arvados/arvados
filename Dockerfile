# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM node:8-buster
MAINTAINER Ward Vandewege <ward@curoverse.com>
RUN apt-get update
RUN apt-get -q -y install libsecret-1-0 libsecret-1-dev rpm
RUN apt-get install -q -y ruby ruby-dev rubygems build-essential
RUN gem install --no-ri --no-rdoc fpm
