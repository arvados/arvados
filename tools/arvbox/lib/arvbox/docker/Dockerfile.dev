# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM arvados/arvbox-base
ARG arvados_version

ADD service/ /var/lib/arvbox/service
RUN ln -sf /var/lib/arvbox/service /etc
RUN mkdir -p /var/lib/arvados
RUN echo "development" > /var/lib/arvados/api_rails_env
RUN echo "development" > /var/lib/arvados/sso_rails_env
RUN echo "development" > /var/lib/arvados/workbench_rails_env

RUN mkdir /etc/test-service && \
    ln -sf /var/lib/arvbox/service/postgres /etc/test-service && \
    ln -sf /var/lib/arvbox/service/certificate /etc/test-service
RUN mkdir /etc/devenv-service