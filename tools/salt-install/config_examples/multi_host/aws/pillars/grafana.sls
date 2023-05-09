---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
      instance_name: __CLUSTER__.__DOMAIN__
    security:
      admin_user: {{ "__MONITORING_USERNAME__" | yaml_dquote }}
      admin_password: {{ "__MONITORING_PASSWORD__" | yaml_dquote }}
      admin_email: {{ "__MONITORING_EMAIL__" | yaml_dquote }}
    server:
      protocol: http
      http_addr: 127.0.0.1
      http_port: 3000
      domain: grafana.__CLUSTER__.__DOMAIN__
      root_url: https://grafana.__CLUSTER__.__DOMAIN__
