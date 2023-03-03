---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}

### NGINX
nginx:
  ### SERVER
  server:
    config:
      ### STREAMS
      http:
        upstream prometheus_upstream:
          - server: '127.0.0.1:9090 fail_timeout=10s'

  ### SITES
  servers:
    managed:
      ### PROMETHEUS
      prometheus:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: prometheus.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      prometheus-ssl:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: prometheus.__CLUSTER__.__DOMAIN__
            - listen:
              - 443 http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://prometheus_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            - include: snippets/ssl_hardening_default.conf
            {%- if ssl_key_encrypted_pillar.ssl_key_encrypted.enabled %}
            - ssl_password_file: {{ '/run/arvados/' | path_join(ssl_key_encrypted_pillar.ssl_key_encrypted.privkey_password_filename) }}
            {%- endif %}
            - auth_basic: '"Restricted Area"'
            - auth_basic_user_file: htpasswd
            - access_log: /var/log/nginx/prometheus.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/prometheus.__CLUSTER__.__DOMAIN__.error.log
