---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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
  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench2_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.__CLUSTER__.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench2_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          file: nginx_snippet_arvados-snakeoil.conf
        config:
          - server:
            - server_name: workbench2.__CLUSTER__.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - root: /var/www/arvados-workbench2/workbench2
              - try_files: '$uri $uri/ /index.html'
              - 'if (-f $document_root/maintenance.html)':
                - return: 503
            - location /config.json:
              - return: {{ "200 '" ~ '{"API_HOST":"__CLUSTER__.__DOMAIN__:__CONTROLLER_EXT_SSL_PORT__"}' ~ "'" }}
            - include: snippets/ssl_hardening_default.conf
            - include: snippets/arvados-snakeoil.conf
            - access_log: /var/log/nginx/workbench2.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench2.__CLUSTER__.__DOMAIN__.error.log
