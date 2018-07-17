# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
requirements:
  ResourceRequirement:
    coresMin: 2
    outdirMin: 1024
inputs: []
outputs: []
baseCommand: echo
arguments:
  - "a"
