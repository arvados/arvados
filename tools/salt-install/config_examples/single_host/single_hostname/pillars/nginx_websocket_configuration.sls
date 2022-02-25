---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### NGINX
nginx:
  ### SERVER
  server:
    config:
      ### STREAMS
      http:
        upstream websocket_upstream:
          - server: '__IP_INT__:8005 fail_timeout=10s'

  servers:
    managed:
      arvados_websocket_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __WEBSOCKET_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://websocket_upstream'
              - proxy_read_timeout: 600
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: 'Host $host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'Upgrade $http_upgrade'
              - proxy_set_header: 'Connection "upgrade"'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
              - proxy_buffering: 'off'
            - client_body_buffer_size: 64M
            - client_max_body_size: 64M
            - proxy_http_version: '1.1'
            - proxy_request_buffering: 'off'
            - include: 'snippets/arvados-snakeoil.conf'
            - access_log: /var/log/nginx/ws.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/ws.__CLUSTER__.__DOMAIN__.error.log
