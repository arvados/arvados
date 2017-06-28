# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import json
import arvados

_rootDesc = None

def get_rootDesc():
    global _rootDesc
    if not _rootDesc:
        try:
            _rootDesc = arvados.api('v1')._rootDesc
        except ValueError:
            raise Exception("Test requires an running API server to fetch discovery document")
    return _rootDesc
