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

extra_grafana_datasource_loki:
  file.managed:
    - name: /etc/grafana/provisioning/datasources/loki.yaml
    - contents: |
        apiVersion: 1
        datasources:
          - name: Loki
            type: loki
            uid: ArvadosLokiDataSource
            url: http://127.0.0.1:3100
    - require:
      - pkg: grafana-package-install-pkg-installed

  cmd.run:
    - name: systemctl restart grafana-server
    - require:
      - file: extra_grafana_datasource_prometheus
      - file: extra_grafana_datasource_loki
    - onchanges:
      - file: extra_grafana_datasource_prometheus
      - file: extra_grafana_datasource_loki
{%- endif %}