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
  ### SERVER
  server:
    config:

      ### STREAMS
      http:
        upstream workbench_upstream:
          - server: '__HOSTNAME_INT__:9000 fail_timeout=10s'

  ### SITES
  servers:
    managed:
      arvados_workbench_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __WORKBENCH1_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://workbench_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
            - include: 'snippets/arvados-snakeoil.conf'
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.error.log

      arvados_workbench_upstream:
        enabled: true
        overwrite: true
        config:
          - server:
            - listen: '__HOSTNAME_INT__:9000'
            - server_name: workbench
            - root: /var/www/arvados-workbench/current/public
            - index:  index.html index.htm
            - passenger_enabled: 'on'
            # yamllint disable-line rule:line-length
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.error.log
