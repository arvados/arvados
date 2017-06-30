# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
$graph:
- class: CommandLineTool
  requirements:
  - class: DockerRequirement
    dockerPull: debian:8
  inputs:
  - id: '#submit_tool.cwl/x'
    type: File
    default:
      class: File
      location: keep:5d373e7629203ce39e7c22af98a0f881+52/blub.txt
    inputBinding:
      position: 1
  outputs: []
  baseCommand: cat
  id: '#submit_tool.cwl'
- class: Workflow
  inputs:
  - id: '#main/x'
    type: File
  - id: '#main/y'
    type: Directory
  - id: '#main/z'
    type: Directory
  outputs: []
  steps:
  - id: '#main/step1'
    in:
    - {id: '#main/step1/x', source: '#main/x'}
    out: []
    run: '#submit_tool.cwl'
  id: '#main'
