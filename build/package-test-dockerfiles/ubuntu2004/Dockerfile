# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:focal
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

RUN mkdir -p /etc/udev/disabled && \
    echo "deb [trusted=yes] file:///arvados/packages/ubuntu2004/ /" >>/etc/apt/sources.list
