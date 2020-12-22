# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Test case for arvados-cwl-runner
#
# Used to test whether scanning a tool file for dependencies (e.g. default
# value blub.txt) and uploading to Keep works as intended.

class: CommandLineTool
cwlVersion: v1.0
requirements:
  - class: DockerRequirement
    dockerPull: debian:buster-slim
inputs:
  - id: x
    type: File
    secondaryFiles:
      - .cat
    inputBinding:
      valueFrom: $(self.path).cat
      position: 1
outputs: []
baseCommand: cat
