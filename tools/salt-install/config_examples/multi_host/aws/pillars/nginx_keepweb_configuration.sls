---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Keepweb upstream is common to both downloads and collections
### NGINX
nginx:
  ### SERVER
  server:
    config:
      ### STREAMS
      http:
        upstream collections_downloads_upstream:
          - server: '__KEEPWEB_INT_IP__:9002 fail_timeout=10s'
