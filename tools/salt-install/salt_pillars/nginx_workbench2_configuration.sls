---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

{% set nginx_log = '/var/log/nginx' %}

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench2_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.arv.local
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench2:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.arv.local
            - listen:
              - 443 http2 ssl
            - index: index.html index.htm
            - location /:
              - root: /var/www/arvados-workbench2/workbench2
              - try_files: '$uri $uri/ /index.html'
              - 'if (-f $document_root/maintenance.html)':
                - return: 503
            # - include: 'snippets/letsencrypt.conf'
            - include: 'snippets/snakeoil.conf'
            - access_log: {{ nginx_log }}/workbench2.arv.local.access.log combined
            - error_log: {{ nginx_log }}/workbench2.arv.local.error.log
