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
          - server: 'collections.internal:9002 fail_timeout=10s'

  servers:
    managed:
      ### DEFAULT
      arvados_collections_download_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: collections.__CLUSTER__.__DOMAIN__ download.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      ### COLLECTIONS / DOWNLOAD
      {%- for vh in [
        'collections',
        'download'
        ]
      %}
      arvados_{{ vh }}.conf:
        enabled: true
        overwrite: true
        requires:
          file: extra_custom_certs_{{ vh }}_cert_file_copy
        config:
          - server:
            - server_name: {{ vh }}.__CLUSTER__.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
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
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: /etc/nginx/ssl/arvados-{{ vh }}.pem
            - ssl_certificate_key: /etc/nginx/ssl/arvados-{{ vh }}.key
            - access_log: /var/log/nginx/{{ vh }}.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/{{ vh }}.__CLUSTER__.__DOMAIN__.error.log
      {%- endfor %}
