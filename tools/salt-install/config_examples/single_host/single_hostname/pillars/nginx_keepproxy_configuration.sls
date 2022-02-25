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
        upstream keepproxy_upstream:
          - server: '__IP_INT__:25100 fail_timeout=10s'

  servers:
    managed:
      arvados_keepproxy_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __KEEP_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://keepproxy_upstream'
              - proxy_read_timeout: 90
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
              - proxy_buffering: 'off'
            - client_body_buffer_size: 64M
            - client_max_body_size: 64M
            - proxy_http_version: '1.1'
            - proxy_request_buffering: 'off'
            - include: 'snippets/arvados-snakeoil.conf'
            - access_log: /var/log/nginx/keepproxy.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/keepproxy.__CLUSTER__.__DOMAIN__.error.log
