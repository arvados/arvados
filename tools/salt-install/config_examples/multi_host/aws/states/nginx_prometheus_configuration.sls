# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- if salt['pillar.get']('nginx:servers:managed:prometheus-ssl') %}

extra_nginx_prometheus_conf_user___MONITORING_USERNAME__:
  webutil.user_exists:
    - name: __MONITORING_USERNAME__
    - password: {{ "__MONITORING_PASSWORD__" | yaml_dquote }}
    - htpasswd_file: /etc/nginx/htpasswd
    - options: d
    - force: true
    - require:
      - pkg: extra_nginx_prometheus_conf_pkgs

extra_nginx_prometheus_conf_pkgs:
  pkg.installed:
    - name: apache2-utils

{%- endif %}