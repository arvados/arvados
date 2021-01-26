---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

locale:
  present:
    - "en_US.UTF-8 UTF-8"
  default:
    # Note: On debian systems don't write the second 'UTF-8' here or you will
    # experience salt problems like: LookupError: unknown encoding: utf_8_utf_8
    # Restart the minion after you corrected this!
    name: 'en_US.UTF-8'
    requires: 'en_US.UTF-8 UTF-8'
