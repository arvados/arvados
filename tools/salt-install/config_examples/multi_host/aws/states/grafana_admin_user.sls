# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set grafana_server = salt['pillar.get']('grafana', {}) %}

{%- if grafana_server %}
extra_grafana_admin_user:
  cmd.run:
    - name: grafana-cli admin reset-admin-password {{ grafana_server.config.security.admin_password }}
    - require:
      - service: grafana-service-running-service-running
{%- endif %}