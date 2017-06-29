# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Test case for arvados-cwl-runner
#
# Used to test whether scanning a workflow file for dependencies
# (e.g. submit_tool.cwl) and uploading to Keep works as intended.

class: Workflow
cwlVersion: v1.0
inputs:
  x:
    type: File
    default:
      class: File
      location: keep:99999999999999999999999999999994+99/blorp.txt
outputs: []
steps:
  step1:
    in:
      x: x
    out: []
    run: ../tool/submit_tool.cwl
