# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

FROM debian:12
COPY *.tar.gz /root
RUN DEBIAN_FRONTEND=noninteractive apt-get update -q && apt-get install -qy --no-install-recommends \
    python3-dev python3-venv libcurl4-gnutls-dev libgnutls28-dev build-essential
RUN python3 -mvenv /usr/local/cluster-activity && \
    /usr/local/cluster-activity/bin/pip install \
    $(ls /root/arvados-python-client-*.tar.gz) \
    $(ls /root/crunchstat_summary-*.tar.gz) \
    $(ls /root/arvados-cluster-activity-*.tar.gz)\[prometheus\] && \
    ln -s /usr/local/cluster-activity/bin/arv-cluster-activity /usr/local/bin
