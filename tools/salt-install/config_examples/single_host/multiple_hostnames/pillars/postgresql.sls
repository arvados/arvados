---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### POSTGRESQL
postgres:
  # Centos-7's postgres package is too old, so we need to force using upstream's
  # This is not required in Debian's family as they already ship with PG +11
  {%- if salt['grains.get']('os_family') == 'RedHat' %}
  use_upstream_repo: true
  version: '12'

  pkgs_deps:
    - libicu
    - libxslt
    - systemd-sysv

  pkgs_extra:
    - postgresql12-contrib

  {%- else %}
  pkgs_extra:
    - postgresql-contrib
  {%- endif %}
  postgresconf: |-
    listen_addresses = '*'  # listen on all interfaces
    #ssl = on
    #ssl_cert_file = '/etc/ssl/certs/arvados-snakeoil-cert.pem'
    #ssl_key_file = '/etc/ssl/private/arvados-snakeoil-cert.key'
  acls:
    - ['local', 'all', 'postgres', 'peer']
    - ['local', 'all', 'all', 'peer']
    - ['host', 'all', 'all', '127.0.0.1/32', 'md5']
    - ['host', 'all', 'all', '::1/128', 'md5']
    - ['host', '__CLUSTER___arvados', '__CLUSTER___arvados', '127.0.0.1/32']
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
