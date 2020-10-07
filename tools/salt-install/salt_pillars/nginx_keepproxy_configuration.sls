---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

{% set nginx_log = '/var/log/nginx' %}

### NGINX
nginx:
  ### SERVER
  server:
    config:
      ### STREAMS
      http:
        upstream keepproxy_upstream:
          - server: '127.0.0.1:25107 fail_timeout=10s'

  servers:
    managed:
      ### DEFAULT
      arvados_keep_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: keep.arv.local
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_keepproxy:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: keep.arv.local
            - listen:
              - 443 http2 ssl
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
            # - include: 'snippets/letsencrypt.conf'
            - include: 'snippets/snakeoil.conf'
            - access_log: {{ nginx_log }}/keepproxy.arv.local.access.log combined
            - error_log: {{ nginx_log }}/keepproxy.arv.local.error.log
