# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.2
class: Workflow
inputs:
  file1:
    type: Directory
    loadListing: deep_listing
    default:
      class: Directory
      location: ../testdir

steps:
  step1:
    in:
      file1: file1
    run: output_dir.cwl
    out: [val, val2]

outputs:
  val:
    type: Directory
    outputSource: step1/val
  val2:
    type: File[]
    outputSource: step1/val2
