# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: Workflow

requirements:
  InlineJavascriptRequirement: {}

inputs:
  file1:
    type: File?
    secondaryFiles:
      - pattern: .tbi
        required: true
  file2:
    type: File
    secondaryFiles:
      - pattern: |
          ${
          return self.basename + '.tbi';
          }
        required: true
outputs:
  out:
    type: File
    outputSource: cat/out
  out2:
    type: File
    outputSource: cat2/out
steps:
  cat:
    in:
      inp: file1
    run: cat2.cwl
    out: [out]
  cat2:
    in:
      inp: file2
    run: cat2.cwl
    out: [out]
