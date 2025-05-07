# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:bookworm-slim
MAINTAINER Arvados Package Maintainers <packaging@arvados.org>

ENV DEBIAN_FRONTEND noninteractive

RUN mkdir -p /etc/udev/disabled && \
    echo "deb file:///arvados/packages/debian12/ /" >>/etc/apt/sources.list
