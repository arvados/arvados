# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set grafana_server = salt['pillar.get']('grafana', {}) %}

{%- if grafana_server %}
extra_grafana_datasource_prometheus:
  file.managed:
    - name: /etc/grafana/provisioning/datasources/prometheus.yaml
    - contents: |
        apiVersion: 1
        datasources:
          - name: Prometheus
            type: prometheus
            uid: ArvadosPromDataSource
            url: http://127.0.0.1:9090
            is_default: true
    - require:
      - pkg: grafana-package-install-pkg-installed

  cmd.run:
    - name: systemctl restart grafana-server
    - require:
      - file: extra_grafana_datasource_prometheus
    - onchanges:
      - file: extra_grafana_datasource_prometheus
{%- endif %}