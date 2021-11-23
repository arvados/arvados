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
          - server: 'workbench.internal:9000 fail_timeout=10s'

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
          file: extra_custom_certs_file_copy_arvados-workbench.pem
        config:
          - server:
            - server_name: workbench.__CLUSTER__.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
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
            - ssl_certificate: /etc/nginx/ssl/arvados-workbench.pem
            - ssl_certificate_key: /etc/nginx/ssl/arvados-workbench.key
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__.error.log

      arvados_workbench_upstream.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - listen: 'workbench.internal:9000'
            - server_name: workbench
            - root: /var/www/arvados-workbench/current/public
            - index:  index.html index.htm
            - passenger_enabled: 'on'
            # yamllint disable-line rule:line-length
            - access_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.access.log combined
            - error_log: /var/log/nginx/workbench.__CLUSTER__.__DOMAIN__-upstream.error.log
