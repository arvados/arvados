# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: CommandLineTool
inputs:
  testcase: string
outputs:
  imagename:
    type: string
    outputBinding:
      outputEval: $(inputs.testcase)
requirements:
  InitialWorkDirRequirement:
    listing:
      - entryname: Dockerfile
        entry: |-
          FROM debian@sha256:0a5fcee6f52d5170f557ee2447d7a10a5bdcf715dd7f0250be0b678c556a501b
          LABEL org.arvados.testcase="$(inputs.testcase)"
arguments: [docker, build, -t, $(inputs.testcase), "."]
