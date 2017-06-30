#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

exec arvados-cwl-runner --disable-reuse --compute-checksum "$@"
