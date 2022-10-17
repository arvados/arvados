# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

var_lib_arvados_keep_dir:
  file.directory:
    - name: /var/lib/arvados/keep
    - user: root
    - group: root
    - mode: '0770'
    - makedirs: true
    - require_in:
      - pkg: {{ arvados.keepstore.pkg.name }}
