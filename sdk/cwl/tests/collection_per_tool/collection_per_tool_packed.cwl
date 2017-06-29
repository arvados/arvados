# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
$graph:
- class: Workflow
  inputs: []
  outputs: []
  steps:
  - in: []
    out: []
    run: '#step1.cwl'
    id: '#main/step1'
  - in: []
    out: []
    run: '#step2.cwl'
    id: '#main/step2'
  id: '#main'
- class: CommandLineTool
  inputs:
  - type: File
    default:
      class: File
      location: keep:b9fca8bf06b170b8507b80b2564ee72b+57/a.txt
    id: '#step1.cwl/a'
  - type: File
    default:
      class: File
      location: keep:b9fca8bf06b170b8507b80b2564ee72b+57/b.txt
    id: '#step1.cwl/b'
  outputs: []
  arguments: [echo, $(inputs.a), $(inputs.b)]
  id: '#step1.cwl'
- class: CommandLineTool
  inputs:
  - type: File
    default:
      class: File
      location: keep:8e2d09a066d96cdffdd2be41579e4e2e+57/b.txt
    id: '#step2.cwl/b'
  - type: File
    default:
      class: File
      location: keep:8e2d09a066d96cdffdd2be41579e4e2e+57/c.txt
    id: '#step2.cwl/c'
  outputs: []
  arguments: [echo, $(inputs.c), $(inputs.b)]
  id: '#step2.cwl'
