# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM arvados/arvbox-base
ARG arvados_version
ARG sso_version=master
ARG composer_version=master
ARG workbench2_version=master

RUN cd /usr/src && \
    git clone --no-checkout https://github.com/curoverse/arvados.git && \
    git -C arvados checkout ${arvados_version} && \
    git clone --no-checkout https://github.com/curoverse/sso-devise-omniauth-provider.git sso && \
    git -C sso checkout ${sso_version} && \
    git clone --no-checkout https://github.com/curoverse/composer.git && \
    git -C composer checkout ${composer_version} && \
    git clone --no-checkout https://github.com/curoverse/arvados-workbench2.git workbench2 && \
    git -C workbench2 checkout ${workbench2_version}

ADD service/ /var/lib/arvbox/service
RUN ln -sf /var/lib/arvbox/service /etc
RUN mkdir -p /var/lib/arvados
RUN echo "production" > /var/lib/arvados/api_rails_env
RUN echo "production" > /var/lib/arvados/sso_rails_env
RUN echo "production" > /var/lib/arvados/workbench_rails_env

RUN chown -R 1000:1000 /usr/src && /usr/local/lib/arvbox/createusers.sh

RUN sudo -u arvbox /var/lib/arvbox/service/composer/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/workbench2/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/keep-web/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/sso/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/api/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/workbench/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/doc/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/vm/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/keepproxy/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/arv-git-httpd/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/crunch-dispatch-local/run-service --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/websockets/run-service --only-deps
RUN sudo -u arvbox /usr/local/lib/arvbox/keep-setup.sh --only-deps
RUN sudo -u arvbox /var/lib/arvbox/service/sdk/run-service
