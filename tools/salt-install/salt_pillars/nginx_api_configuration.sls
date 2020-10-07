---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

{% set nginx_log = '/var/log/nginx' %}

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SITES
  servers:
    managed:
      arvados_api:
        enabled: true
        overwrite: true
        config:
          - server:
            - listen: '127.0.0.1:8004'
            - server_name: api
            - root: /var/www/arvados-api/current/public
            - index:  index.html index.htm
            - access_log: {{ nginx_log }}/api.arv.local-upstream.access.log combined
            - error_log: {{ nginx_log }}/api.arv.local-upstream.error.log
            - passenger_enabled: 'on'
            - client_max_body_size: 128m
