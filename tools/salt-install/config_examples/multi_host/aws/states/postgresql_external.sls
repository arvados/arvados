# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set pg_svc = pillar.get('postgresql_external_service', {}) %}

{%- if pg_svc %}
__CLUSTER___external_trgm_extension:
  postgres_extension.present:
    - name: pg_trgm
    - if_not_exists: true
    - schema: public
    - db_host: {{ pg_svc.db_host }}
    - db_port: 5432
    - db_user: {{ pg_svc.db_user }}
    - db_password: {{ pg_svc.db_password }}
    - require:
      - pkg: postgresql-client-libs
{%- endif %}