# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import sys
import os

if "JOB_UUID" in os.environ:
    requested = arvados.api().jobs().get(uuid=os.environ["JOB_UUID"]).execute()["runtime_constraints"]["min_ram_mb_per_node"]
else:
    requested = arvados.api().containers().current().execute()["runtime_constraints"]["ram"] // (1024*1024)

print("Requested %d expected %d" % (requested, int(sys.argv[1])))

exit(0 if requested == int(sys.argv[1]) else 1)
