# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: Workflow
inputs:
  file1:
    type: File?
    secondaryFiles:
      - pattern: .tbi
        required: true
outputs:
  out:
    type: File
    outputSource: cat/out
steps:
  cat:
    in:
      inp: file1
    run: cat2.cwl
    out: [out]
