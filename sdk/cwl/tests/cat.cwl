# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
inputs:
  - id: inp
    type: File
    inputBinding: {}
outputs: []
baseCommand: cat
