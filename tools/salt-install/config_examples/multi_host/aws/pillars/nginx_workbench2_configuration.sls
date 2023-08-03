---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}

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
      arvados_workbench2_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.__DOMAIN__
            - listen:
              - 80
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench2_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: workbench2.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl

            - location /:
              - return: '301 https://workbench.__DOMAIN__$request_uri'

            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: __CERT_PEM__
            - ssl_certificate_key: __CERT_KEY__
            {%- if ssl_key_encrypted_pillar.ssl_key_encrypted.enabled %}
            - ssl_password_file: {{ '/run/arvados/' | path_join(ssl_key_encrypted_pillar.ssl_key_encrypted.privkey_password_filename) }}
            {%- endif %}
            - access_log: /var/log/nginx/workbench2.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench2.__DOMAIN__.error.log
