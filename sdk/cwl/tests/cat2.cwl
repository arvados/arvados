# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: CommandLineTool
inputs:
  - id: inp
    type: File
    secondaryFiles:
      - pattern: .tbi
        required: true
stdout: catted
outputs:
  out:
    type: stdout
arguments: [cat, '$(inputs.inp.path)', '$(inputs.inp.secondaryFiles[0].path)']
