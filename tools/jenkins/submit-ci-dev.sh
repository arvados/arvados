#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

head=$(git log --first-parent --max-count=1 --format=%H)
curl -X POST https://ci.arvados.org/job/developer-run-tests/build \
  --user $(cat ~/.jenkins.ci.arvados.org) \
  --data-urlencode json='{"parameter": [{"name":"git_hash", "value":"'$head'"}]}'
