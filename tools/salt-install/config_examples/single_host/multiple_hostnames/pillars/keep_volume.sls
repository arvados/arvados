# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

/var/lib/arvados/keep:
  file.directory:
    - user: root
    - group: root
    - mode: '0770'
    - makedirs: True
