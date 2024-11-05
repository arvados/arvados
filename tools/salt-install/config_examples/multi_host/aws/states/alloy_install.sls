# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set alloy = pillar.get('alloy', {'enabled': False}) %}

{%- if alloy.enabled %}
extra_grafana_package_repo:
  pkgrepo.managed:
    - humanname: grafana_official
    - name: deb https://apt.grafana.com/ stable main
    - file: /etc/apt/sources.list.d/grafana.list
    - key_url: https://apt.grafana.com/gpg.key

extra_install_alloy:
  pkg.installed:
    - name: {{ alloy.package }}
    - refresh: true
    - require:
      - pkgrepo: extra_grafana_package_repo

extra_alloy_config:
  file.managed:
    - name: {{ alloy.config_path }}
    - contents: {{ alloy.config_contents | yaml_dquote }}
    - mode: '0640'
    - user: alloy
    - group: root
    - require:
      - pkg: extra_install_alloy

extra_alloy_service:
  service.running:
    - name: {{ alloy.service }}
    - enable: true
    - require:
      - pkg: extra_install_alloy
      - file: extra_alloy_config
    - watch:
      - file: extra_alloy_config
{%- endif %}