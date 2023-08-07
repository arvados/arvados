---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- import_yaml "ssl_key_encrypted.sls" as ssl_key_encrypted_pillar %}
{%- set domain = "__DOMAIN__" %}
{%- set balancer_backends = "__CONTROLLER_NODES__".split(",") %}
{%- set controller_nr = balancer_backends|length %}
{%- set disabled_controller = "__DISABLED_CONTROLLER__" %}
{%- if disabled_controller != "" %}
  {%- set controller_nr = controller_nr - 1 %}
{%- endif %}
{%- set max_reqs = ("__CONTROLLER_MAX_QUEUED_REQUESTS__" or 128)|int %}

### NGINX
nginx:
  ### SERVER
  server:
    config:
      {%- if max_reqs != "" %}
      worker_rlimit_nofile: {{ (max_reqs|int * 3 * controller_nr)|round|int }}
      events:
        worker_connections: {{ (max_reqs|int * 3 * controller_nr)|round|int }}
      {%- else %}
      worker_rlimit_nofile: 4096
      events:
        worker_connections: 1024
      {%- endif %}
      ### STREAMS
      http:
        'geo $external_client':
          default: 1
          '127.0.0.0/8': 0
          '__CLUSTER_INT_CIDR__': 0
        upstream controller_upstream:
        {%- for backend in balancer_backends %}
          {%- if disabled_controller == "" or not backend.startswith(disabled_controller) %}
          'server {{ backend }}:80': ''
          {%- else %}
          'server {{ backend }}:80 down': ''
          {% endif %}
        {%- endfor %}

  ### SNIPPETS
  snippets:
    # Based on https://ssl-config.mozilla.org/#server=nginx&version=1.14.2&config=intermediate&openssl=1.1.1d&guideline=5.4
    ssl_hardening_default.conf:
      - ssl_session_timeout: 1d
      - ssl_session_cache: 'shared:arvadosSSL:10m'
      - ssl_session_tickets: 'off'

      # intermediate configuration
      - ssl_protocols: TLSv1.2 TLSv1.3
      - ssl_ciphers: ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
      - ssl_prefer_server_ciphers: 'off'

      # HSTS (ngx_http_headers_module is required) (63072000 seconds)
      - add_header: 'Strict-Transport-Security "max-age=63072000" always'

      # OCSP stapling
      - ssl_stapling: 'on'
      - ssl_stapling_verify: 'on'

      # verify chain of trust of OCSP response using Root CA and Intermediate certs
      # - ssl_trusted_certificate /path/to/root_CA_cert_plus_intermediates

      # curl https://ssl-config.mozilla.org/ffdhe2048.txt > /path/to/dhparam
      # - ssl_dhparam: /path/to/dhparam

      # replace with the IP address of your resolver
      # - resolver: 127.0.0.1

  ### SITES
  servers:
    managed:
      # Remove default webserver
      default:
        enabled: false
      ### DEFAULT
      arvados_balancer_default.conf:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: {{ domain }}
            - listen:
              - 80 default
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_balancer_ssl.conf:
        enabled: true
        overwrite: true
        requires:
          __CERT_REQUIRES__
        config:
          - server:
            - server_name: {{ domain }}
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
            - access_log: /var/log/nginx/{{ domain }}.access.log combined
            - error_log: /var/log/nginx/{{ domain }}.error.log
            - client_max_body_size: 128m
