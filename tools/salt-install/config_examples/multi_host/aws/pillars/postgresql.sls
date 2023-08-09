---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set domain = "__DOMAIN__" %}
{%- set controller_nodes = "__CONTROLLER_NODES__".split(",") %}
{%- set websocket_ip = "__WEBSOCKET_INT_IP__" %}
{%- set keepbalance_ip = "__KEEPBALANCE_INT_IP__" %}

### POSTGRESQL
postgres:
  pkgs_extra:
    - postgresql-contrib
  use_upstream_repo: true
  version: '12'
  postgresconf: |-
    listen_addresses = '*'  # listen on all interfaces
  acls:
    - ['local', 'all', 'postgres', 'peer']
    - ['local', 'all', 'all', 'peer']
    - ['host', 'all', 'all', '127.0.0.1/32', 'md5']
    - ['host', 'all', 'all', '::1/128', 'md5']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '127.0.0.1/32']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '{{ websocket_ip }}/32']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '{{ keepbalance_ip }}/32']
    {%- for controller_hostname in controller_nodes %}
    {%- set controller_ip = salt['cmd.run']("getent hosts "+controller_hostname+" | awk '{print $1 ; exit}'", python_shell=True) %}
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '{{ controller_ip }}/32']
    {%- endfor %}
  users:
    __CLUSTER___arvados:
      ensure: present
      password: "__DATABASE_PASSWORD__"
    prometheus:
      ensure: present
  databases:
    __CLUSTER___arvados:
      owner: __CLUSTER___arvados
      template: template0
      lc_ctype: en_US.utf8
      lc_collate: en_US.utf8
      schemas:
        public:
          owner: __CLUSTER___arvados
      extensions:
        pg_trgm:
          if_not_exists: true
          schema: public
