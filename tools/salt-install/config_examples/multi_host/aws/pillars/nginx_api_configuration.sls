---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SITES
  servers:
    managed:
      arvados_api.conf:
        enabled: false
        overwrite: false
        config:
          - server:
            - listen: 'localhost:8004'
            - server_name: api
            - root: /var/www/arvados-api/current/public
            - index:  index.html index.htm
            - access_log: /var/log/nginx/api.__DOMAIN__-upstream.access.log combined
            - error_log: /var/log/nginx/api.__DOMAIN__-upstream.error.log
            - client_max_body_size: 128m
