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
        upstream collections_downloads_upstream:
          - server: '__HOSTNAME_INT__:9003 fail_timeout=10s'

  servers:
    managed:
      ### COLLECTIONS / DOWNLOAD
      arvados_collections_download_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __KEEPWEB_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://collections_downloads_upstream'
              - proxy_read_timeout: 90
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
              - proxy_buffering: 'off'
            - client_max_body_size: 0
            - proxy_http_version: '1.1'
            - proxy_request_buffering: 'off'
            - include: 'snippets/arvados-snakeoil.conf'
            - access_log: /var/log/nginx/keepweb.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/keepweb.__CLUSTER__.__DOMAIN__.error.log
