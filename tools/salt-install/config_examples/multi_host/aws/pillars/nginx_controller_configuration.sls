---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}
{%- set balanced_controller = ("__ENABLE_BALANCER__"|to_bool) %}
{%- set server_name = grains['fqdn'] if balanced_controller else "__DOMAIN__" %}

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
          '__CLUSTER_INT_CIDR__': 0
        upstream controller_upstream:
          - server: 'localhost:8003  fail_timeout=10s'

  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_controller_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: {{ server_name }}
            - listen:
              - 80 default
            - location /.well-known:
              - root: /var/www
            {%- if balanced_controller %}
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://controller_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_max_temp_file_size: 0
              - proxy_request_buffering: 'off'
              - proxy_buffering: 'off'
              - proxy_http_version: '1.1'
            - access_log: /var/log/nginx/{{ server_name }}.access.log combined
            - error_log: /var/log/nginx/{{ server_name }}.error.log
            - client_max_body_size: 128m
            {%- else %}
            - location /:
              - return: '301 https://$host$request_uri'
            {%- endif %}

      {%- if not balanced_controller %}
      arvados_controller_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: {{ server_name }}
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
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
              - proxy_set_header: 'Upgrade $http_upgrade'
              - proxy_set_header: 'Connection "upgrade"'
              - proxy_max_temp_file_size: 0
              - proxy_request_buffering: 'off'
              - proxy_buffering: 'off'
              - proxy_http_version: '1.1'
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            {%- if ssl_key_encrypted_pillar.ssl_key_encrypted.enabled %}
            - ssl_password_file: {{ '/run/arvados/' | path_join(ssl_key_encrypted_pillar.ssl_key_encrypted.privkey_password_filename) }}
            {%- endif %}
            - access_log: /var/log/nginx/{{ server_name }}.access.log combined
            - error_log: /var/log/nginx/{{ server_name }}.error.log
            - client_max_body_size: 128m
      {%- endif %}
