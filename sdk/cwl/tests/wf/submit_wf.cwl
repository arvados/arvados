# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Test case for arvados-cwl-runner
#
# Used to test whether scanning a workflow file for dependencies
# (e.g. submit_tool.cwl) and uploading to Keep works as intended.

class: Workflow
cwlVersion: v1.2
inputs:
  - id: x
    type: File
  - id: y
    type: Directory
  - id: z
    type: Directory
outputs: []
steps:
  - id: step1
    in:
      - { id: x, source: "#x" }
    out: []
    run: ../tool/submit_tool.cwl
