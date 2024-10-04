# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set loki = pillar.get('loki', {'enabled': False}) %}

{%- if loki.enabled %}
extra_install_loki:
  pkg.installed:
    - name: {{ loki['package'] }}
    - refresh: true
    - require:
      - pkgrepo: grafana-package-repo-install-pkgrepo-managed

extra_loki_config:
  file.managed:
    - name: {{ loki['config_path'] }}
    - contents: {{ loki['config_contents'] | yaml_dquote }}
    - mode: '0644'
    - user: root
    - group: root
    - require:
      - pkg: extra_install_loki

extra_loki_data_dir:
  file.directory:
    - name: {{ loki['data_path'] }}
    - user: loki
    - mode: '0750'
    - require:
      - pkg: extra_install_loki

extra_loki_service:
  service.running:
    - name: {{ loki['service'] }}
    - enable: true
    - require:
      - pkg: extra_install_loki
      - file: extra_loki_config
      - file: extra_loki_data_dir
    - watch:
      - file: extra_loki_config
{%- endif %}