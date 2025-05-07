# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM ubuntu:noble
LABEL org.opencontainers.image.authors="Arvados Package Maintainers <packaging@arvados.org>"

ENV DEBIAN_FRONTEND noninteractive

RUN mkdir -p /etc/udev/disabled && \
    echo "deb [trusted=yes] file:///arvados/packages/ubuntu2404/ /" >>/etc/apt/sources.list
