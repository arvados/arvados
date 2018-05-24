# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
requirements:
  ResourceRequirement:
    coresMin: 3
    outdirMin: 2048
inputs: []
outputs: []
baseCommand: echo
arguments:
  - "b"
