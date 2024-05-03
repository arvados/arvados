# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import json

api = arvados.api()

with open("config.json") as f:
    config = json.load(f)

success = True
for c in config["check_collections"]:
    try:
        api.collections().get(uuid=c).execute()
    except Exception as e:
        print("Checking for %s got exception %s" % (c, e))
        success = False

with open("success", "w") as f:
    if success:
        f.write("true")
    else:
        f.write("false")
