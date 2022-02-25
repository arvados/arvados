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
        'geo $external_client':
          default: 1
          '127.0.0.0/8': 0
        upstream controller_upstream:
          - server: '__IP_INT__:8003  fail_timeout=10s'

  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_controller_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: _
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_controller_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl default_server
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://controller_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
              - proxy_set_header: 'X-External-Client $external_client'
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            - access_log: /var/log/nginx/__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/__CLUSTER__.__DOMAIN__.error.log
            - client_max_body_size: 128m
