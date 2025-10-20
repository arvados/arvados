# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set _workers = ("__CONTROLLER_MAX_WORKERS__" or grains['num_cpus']*2)|int %}
{%- set max_workers = [_workers, 8]|max %}

{%- if salt['pillar.get']('nginx:servers:managed:arvados_controller_default.conf') %}

# Make the passenger queue small (twice the concurrency, so
# there's at most one pending request for each busy worker)
# because controller reorders requests based on priority, and
# won't send more than API.MaxConcurrentRailsRequests to passenger
# (which is max_workers * 2), so things that are moved to the head
# of the line get processed quickly.
extra_railsapi_passenger_configs:
  file.managed:
    - name: /etc/systemd/system/arvados-railsapi.service.d/override.conf
    - contents: |
        ### This file managed by Salt, do not edit by hand!!
        [Service]
        Environment=PASSENGER_MAX_POOL_SIZE={{ max_workers }}
        Environment=PASSENGER_MAX_REQUEST_QUEUE_SIZE={{ max_workers * 2 + 1 }}
    - user: root
    - group: root
    - mode: '0644'
    - makedirs: True
    - require_in:
      - service: arvados-api-service-running-service-running

{%- endif %}
