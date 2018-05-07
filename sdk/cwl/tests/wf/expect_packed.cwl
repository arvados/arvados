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
    default: {class: File, location: keep:169f39d466a5438ac4a90e779bf750c7+53/blorp.txt,
      size: 16, basename: blorp.txt, nameroot: blorp, nameext: .txt}
  - id: '#main/y'
    type: Directory
    default: {class: Directory, location: keep:99999999999999999999999999999998+99,
      basename: 99999999999999999999999999999998+99}
  - id: '#main/z'
    type: Directory
    default: {class: Directory, basename: anonymous, listing: [{basename: renamed.txt,
          class: File, location: keep:99999999999999999999999999999998+99/file1.txt,
          nameroot: renamed, nameext: .txt}]}
  outputs: []
  steps:
  - id: '#main/step1'
    in:
    - {id: '#main/step1/x', source: '#main/x'}
    out: []
    run: '#submit_tool.cwl'
  id: '#main'
