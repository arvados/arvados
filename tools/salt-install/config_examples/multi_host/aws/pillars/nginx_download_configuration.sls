---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}

### NGINX
nginx:
  servers:
    managed:
      ### DEFAULT
      arvados_download_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: download.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - location /:
              - return: '301 https://$host$request_uri'

      ### DOWNLOAD
      arvados_download_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: download.__CLUSTER__.__DOMAIN__
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
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            {%- if ssl_key_encrypted_pillar.ssl_key_encrypted.enabled %}
            - ssl_password_file: {{ '/run/arvados/' | path_join(ssl_key_encrypted_pillar.ssl_key_encrypted.privkey_password_filename) }}
            {%- endif %}
            - access_log: /var/log/nginx/download.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/download.__CLUSTER__.__DOMAIN__.error.log
