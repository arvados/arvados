# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import sys
import time
import arvados

api = arvados.api()
current_container = api.containers().current().execute()

if current_container["runtime_constraints"]["ram"] < (512*1024*1024):
    print("Whoops")
    sys.exit(1)
