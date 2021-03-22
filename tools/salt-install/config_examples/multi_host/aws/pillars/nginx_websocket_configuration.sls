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
          - server: 'localhost:8005 fail_timeout=10s'

  servers:
    managed:
      ### DEFAULT
      arvados_websocket_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: ws.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - include: snippets/letsencrypt_well_known.conf
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_websocket_ssl:
        enabled: true
        overwrite: true
        requires:
          cmd: create-initial-cert-ws.__CLUSTER__.__DOMAIN__-ws.__CLUSTER__.__DOMAIN__
        config:
          - server:
            - server_name: ws.__CLUSTER__.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
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
            - include: snippets/ssl_hardening_default.conf
            - include: snippets/ws.__CLUSTER__.__DOMAIN___letsencrypt_cert[.]conf
            - access_log: /var/log/nginx/ws.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/ws.__CLUSTER__.__DOMAIN__.error.log
