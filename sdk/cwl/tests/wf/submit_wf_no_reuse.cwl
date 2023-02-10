# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Test case for arvados-cwl-runner. Disables job/container reuse.

class: Workflow
cwlVersion: v1.2
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
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
hints:
  WorkReuse:
    enableReuse: false
