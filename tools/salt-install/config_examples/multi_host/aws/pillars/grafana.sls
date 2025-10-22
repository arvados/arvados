---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set enable_smtp = ("__GRAFANA_SMTP_SERVER__" != "") %}
{%- set smtp_server = "__GRAFANA_SMTP_SERVER__" %}
{%- set smtp_user = "__GRAFANA_SMTP_USER__" %}
{%- set smtp_pwd = "__GRAFANA_SMTP_PASSWORD__" %}
{%- set smtp_from = ("__GRAFANA_SMTP_FROM_EMAIL__" or "grafana@__DOMAIN__") %}
{%- set smtp_name = ("__GRAFANA_SMTP_FROM_NAME__" or "Grafana __CLUSTER__") %}

grafana:
  pkg:
    name: grafana
    use_upstream_archive: false
    use_upstream_repo: true
    repo:
      humanname: grafana_official
      name: deb https://apt.grafana.com/ stable main
      file: /etc/apt/sources.list.d/grafana.list
      key_url: https://apt.grafana.com/gpg.key
      require_in:
        - pkg: grafana
  config:
    default:
      instance_name: __DOMAIN__
    security:
      admin_user: {{ "__MONITORING_USERNAME__" | yaml_dquote }}
      admin_password: {{ "__MONITORING_PASSWORD__" | yaml_dquote }}
      admin_email: {{ "__MONITORING_EMAIL__" | yaml_dquote }}
    server:
      protocol: http
      http_addr: 127.0.0.1
      http_port: 3000
      domain: grafana.__DOMAIN__
      root_url: https://grafana.__DOMAIN__
{%- if enable_smtp %}
    smtp:
      enabled: yes
      host: {{ smtp_server }}
  {%- if smtp_user != '' and smtp_pwd != '' %}
      user: {{ smtp_user | yaml_dquote }}
      password: {{ smtp_pwd | yaml_dquote }}
  {%- endif %}
      from_address: {{ smtp_from | yaml_dquote }}
      from_name: {{ smtp_name | yaml_dquote }}
      skip_verify: false
{%- endif %}