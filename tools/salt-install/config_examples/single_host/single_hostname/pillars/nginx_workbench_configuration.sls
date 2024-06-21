---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- if grains.os_family in ('RedHat',) %}
  {%- set group = 'nginx' %}
{%- else %}
  {%- set group = 'www-data' %}
{%- endif %}

### ARVADOS
arvados:
  config:
    group: {{ group }}

### NGINX
nginx:
  ### SERVER
  server:
    config:

      ### STREAMS
      http:
        upstream workbench_upstream:
          - server: '__IP_INT__:9000 fail_timeout=10s'

  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __WORKBENCH1_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://workbench_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.error.log

      arvados_workbench_upstream:
        enabled: true
        overwrite: true
        config:
          - server:
            - listen: '__IP_INT__:9000'
            - server_name: workbench
            - root: /var/www/arvados-workbench/current/public
            - index:  index.html index.htm
            - passenger_enabled: 'on'
            - passenger_preload_bundler: 'on'
            # yamllint disable-line rule:line-length
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.error.log
