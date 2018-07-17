# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
requirements:
  ScatterFeatureRequirement: {}
inputs:
  dir: Directory
outputs:
  out:
    type: File[]
    outputSource: tool/out
steps:
  ex:
    in:
      dir: dir
    out: [out]
    run: 12213-keepref-expr.cwl
  tool:
    in:
      fastqsdir: ex/out
    out: [out]
    scatter: fastqsdir
    run: 12213-keepref-tool.cwl