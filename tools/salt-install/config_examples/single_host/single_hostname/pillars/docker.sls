---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

docker:
  pkg:
    docker:
      use_upstream: package
      daemon_config: {"dns": ["__IP_INT__"]}
