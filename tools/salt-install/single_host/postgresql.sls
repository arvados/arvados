---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### POSTGRESQL
postgres:
  use_upstream_repo: false
  pkgs_extra:
    - postgresql-contrib
  postgresconf: |-
    listen_addresses = '*'  # listen on all interfaces
  acls:
    - ['local', 'all', 'postgres', 'peer']
    - ['local', 'all', 'all', 'peer']
    - ['host', 'all', 'all', '127.0.0.1/32', 'md5']
    - ['host', 'all', 'all', '::1/128', 'md5']
    - ['host', 'arvados', 'arvados', '127.0.0.1/32']
  users:
    arvados:
      ensure: present
      password: changeme_arvados

  # tablespaces:
  #   arvados_tablespace:
  #     directory: /path/to/some/tbspace/arvados_tbsp
  #     owner: arvados

  databases:
    arvados:
      owner: arvados
      template: template0
      lc_ctype: en_US.utf8
      lc_collate: en_US.utf8
      # tablespace: arvados_tablespace
      schemas:
        public:
          owner: arvados
      extensions:
        pg_trgm:
          if_not_exists: true
          schema: public
