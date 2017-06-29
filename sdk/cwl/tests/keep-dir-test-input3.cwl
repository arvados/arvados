# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

class: CommandLineTool
cwlVersion: v1.0
requirements:
  - class: ShellCommandRequirement
inputs:
  indir:
    type: Directory
    inputBinding:
      prefix: cd
      position: -1
    default:
      class: Directory
      location: keep:d7514270f356df848477718d58308cc4+94/
outputs:
  outlist:
    type: File
    outputBinding:
      glob: output.txt
arguments: [
  {shellQuote: false, valueFrom: "&&"},
  "find", ".",
  {shellQuote: false, valueFrom: "|"},
  "sort"]
stdout: output.txt