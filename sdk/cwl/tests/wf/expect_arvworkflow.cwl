# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
$graph:
- class: Workflow
  id: '#main'
  inputs:
  - id: '#main/x'
    type: string
  outputs: []
  steps:
  - id: '#main/step1'
    in:
    - {id: '#main/step1/x', source: '#main/x'}
    out: []
    run: '#submit_tool.cwl'
- baseCommand: cat
  class: CommandLineTool
  id: '#submit_tool.cwl'
  inputs:
  - id: '#submit_tool.cwl/x'
    inputBinding: {position: 1}
    type: string
  outputs: []
  requirements:
  - {class: DockerRequirement, dockerPull: 'debian:8'}
