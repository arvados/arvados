# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set grafana_server = salt['pillar.get']('grafana', {}) %}
{%- set grafana_dashboards_orig_dir = '/srv/salt/dashboards' %}
{%- set grafana_dashboards_dest_dir = '/var/lib/grafana/dashboards' %}

{%- if grafana_server %}
extra_grafana_dashboard_directory:
  file.directory:
    - name: {{ grafana_dashboards_dest_dir }}
    - require:
      - pkg: grafana-package-install-pkg-installed

extra_grafana_dashboard_default_yaml:
  file.managed:
    - name: /etc/grafana/provisioning/dashboards/default.yaml
    - contents: |
        apiVersion: 1
        providers:
          - name: 'General'
            folder: 'Arvados Cluster'
            type: file
            options:
              path: {{ grafana_dashboards_dest_dir }}
    - require:
      - pkg: grafana-package-install-pkg-installed
      - file: extra_grafana_dashboard_directory

extra_grafana_dashboard_files:
  file.copy:
    - name: {{ grafana_dashboards_dest_dir }}
    - source: {{ grafana_dashboards_orig_dir }}
    - force: true
    - recurse: true
    - require:
      - file: extra_grafana_dashboard_default_yaml

extra_grafana_dashboards_service_restart:
  cmd.run:
    - name: systemctl restart grafana-server
    - require:
      - file: extra_grafana_dashboard_default_yaml
    - onchanges:
      - file: extra_grafana_dashboard_default_yaml
      - file: extra_grafana_dashboard_files
{%- endif %}