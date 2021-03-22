---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### POSTGRESQL
postgres:
  use_upstream_repo: true
  version: '11'
  postgresconf: |-
    listen_addresses = '*'  # listen on all interfaces
  acls:
    - ['local', 'all', 'postgres', 'peer']
    - ['local', 'all', 'all', 'peer']
    - ['host', 'all', 'all', '127.0.0.1/32', 'md5']
    - ['host', 'all', 'all', '::1/128', 'md5']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '127.0.0.1/32']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '__CONTROLLER_INT_IP__/32']
  users:
    __CLUSTER___arvados:
      ensure: present
      password: __DATABASE_PASSWORD__

  # tablespaces:
  #   arvados_tablespace:
  #     directory: /path/to/some/tbspace/arvados_tbsp
  #     owner: arvados

  databases:
    __CLUSTER___arvados:
      owner: __CLUSTER___arvados
      template: template0
      lc_ctype: en_US.utf8
      lc_collate: en_US.utf8
      # tablespace: arvados_tablespace
      schemas:
        public:
          owner: __CLUSTER___arvados
      extensions:
        pg_trgm:
          if_not_exists: true
          schema: public
